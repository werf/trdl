package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/hashicorp/go-hclog"
)

type S3Filesystem struct {
	AwsConfig  *aws.Config
	BucketName string

	// TODO: cache opened session
}

func NewS3Filesystem(awsConfig *aws.Config, bucketName string) *S3Filesystem {
	return &S3Filesystem{AwsConfig: awsConfig, BucketName: bucketName}
}

func (fs *S3Filesystem) IsFileExist(ctx context.Context, path string) (bool, error) {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return false, fmt.Errorf("error opening s3 session: %s", err)
	}

	svc := s3.New(sess)

	_, err = svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &fs.BucketName,
		Key:    &path,
	})
	log.L().Debug("-- S3Filesystem.IsFileExist %q err=%v\n", path, err)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("error heading s3 object by key %q: %s", path, err)
	}

	return true, nil
}

func (fs *S3Filesystem) ReadFile(ctx context.Context, path string, writerAt io.WriterAt) error {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %s", err)
	}

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(writerAt,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return fmt.Errorf("unable to download item %q: %s", path, err)
	}

	fmt.Println("Downloaded", path, numBytes, "bytes")

	return nil
}

type fakeWriterAt struct {
	Writer io.Writer
}

func (fw fakeWriterAt) WriteAt(p []byte, offset int64) (int, error) {
	// ignore 'offset' because we forced sequential downloads

	n, err := fw.Writer.Write(p)

	// DEBUG
	// log.L().Debug("-- fakeWriterAt.WriteAt(%p, %d) -> %d, %v\n", p, offset, n, err)

	return n, err
}

func (fs *S3Filesystem) ReadFileStream(ctx context.Context, path string, writer io.Writer) error {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %s", err)
	}

	downloader := s3manager.NewDownloader(sess)
	downloader.Concurrency = 1

	writerAt := fakeWriterAt{Writer: writer}

	numBytes, err := downloader.Download(writerAt,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return fmt.Errorf("unable to download item %q: %s", path, err)
	}

	fmt.Println("Downloaded", path, numBytes, "bytes")

	return nil
}

func (fs *S3Filesystem) ReadFileBytes(ctx context.Context, path string) ([]byte, error) {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening s3 session: %s", err)
	}

	downloader := s3manager.NewDownloader(sess)

	buf := aws.NewWriteAtBuffer([]byte{})

	numBytes, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to download item %q: %s", path, err)
	}

	fmt.Println("Downloaded", path, numBytes, "bytes")

	return buf.Bytes(), nil
}

func (fs *S3Filesystem) WriteFileBytes(ctx context.Context, path string, data []byte) error {
	return fs.WriteFileStream(ctx, path, bytes.NewReader(data))
}

func (fs *S3Filesystem) WriteFileStream(ctx context.Context, path string, data io.Reader) error {
	// TODO: cache opened session

	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %s", err)
	}

	uploader := s3manager.NewUploader(sess)

	upParams := &s3manager.UploadInput{
		Bucket: &fs.BucketName,
		Key:    &path,
		// DEBUG
		// Body:   &debugReader{data},
		Body: data,
	}

	// TODO: set file mode bits

	result, err := uploader.UploadWithContext(ctx, upParams, func(u *s3manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024
		u.LeavePartsOnError = false
	})
	if err != nil {
		return fmt.Errorf("error uploading %q: %s", path, err)
	}

	log.L().Debug("Uploaded %q\n", result.Location)

	return nil
}

type debugReader struct {
	origReader io.Reader
}

func (o *debugReader) Read(p []byte) (int, error) {
	n, err := o.origReader.Read(p)

	log.L().Debug("-- debugReader Read(%p) -> %d, %v\n", p, n, err)

	return n, err
}
