package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/go-hclog"
)

type S3Filesystem struct {
	AwsConfig  *aws.Config
	BucketName string

	logger hclog.Logger

	// TODO: cache opened session
}

func NewS3Filesystem(awsConfig *aws.Config, bucketName string, logger hclog.Logger) *S3Filesystem {
	if !strings.Contains(*awsConfig.Endpoint, "s3.amazonaws.com") {
		awsConfig.S3ForcePathStyle = new(bool)
		*awsConfig.S3ForcePathStyle = true
	}

	return &S3Filesystem{AwsConfig: awsConfig, BucketName: bucketName, logger: logger}
}

func (fs *S3Filesystem) IsFileExist(ctx context.Context, path string) (bool, error) {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return false, fmt.Errorf("error opening s3 session: %w", err)
	}

	svc := s3.New(sess)

	_, err = svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &fs.BucketName,
		Key:    &path,
	})
	fs.logger.Debug(fmt.Sprintf("-- S3Filesystem.IsFileExist %q err=%v", path, err))
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("error heading s3 object by key %q: %w", path, err)
	}

	return true, nil
}

func (fs *S3Filesystem) ReadFile(ctx context.Context, path string, writerAt io.WriterAt) error {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %w", err)
	}

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(writerAt,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return fmt.Errorf("unable to download item %q: %w", path, err)
	}

	fs.logger.Debug(fmt.Sprintf("Downloaded %q %d bytes", path, numBytes))

	return nil
}

// Use this writer only when Concurrency is set to 1
type sequentialWriterAt struct {
	Writer io.Writer
}

func (fw sequentialWriterAt) WriteAt(p []byte, offset int64) (int, error) {
	// ignore 'offset' because we forced sequential downloads

	n, err := fw.Writer.Write(p)

	// DEBUG
	// fs.logger.Debug(fmt.Sprintf("-- sequentialWriterAt.WriteAt(%p, %d) -> %d, %v", p, offset, n, err))

	return n, err
}

func (fs *S3Filesystem) ReadFileStream(ctx context.Context, path string, writer io.Writer) error {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %w", err)
	}

	downloader := s3manager.NewDownloader(sess)

	downloader.Concurrency = 1
	writerAt := sequentialWriterAt{Writer: writer}

	numBytes, err := downloader.Download(writerAt,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return fmt.Errorf("unable to download item %q: %w", path, err)
	}

	fs.logger.Debug(fmt.Sprintf("-- S3Filesystem.ReadFileStream downloaded %q %d bytes", path, numBytes))

	return nil
}

func (fs *S3Filesystem) ReadFileBytes(ctx context.Context, path string) ([]byte, error) {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening s3 session: %w", err)
	}

	downloader := s3manager.NewDownloader(sess)

	buf := aws.NewWriteAtBuffer([]byte{})

	numBytes, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to download item %q: %w", path, err)
	}

	fs.logger.Debug(fmt.Sprintf("-- S3Filesystem.ReadFileBytes downloaded %q %d bytes", path, numBytes))

	return buf.Bytes(), nil
}

func (fs *S3Filesystem) WriteFileBytes(ctx context.Context, path string, data []byte) error {
	return fs.WriteFileStream(ctx, path, bytes.NewReader(data))
}

func (fs *S3Filesystem) WriteFileStream(ctx context.Context, path string, data io.Reader) error {
	// TODO: cache opened session
	cacheControl := "no-store"

	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %w", err)
	}

	uploader := s3manager.NewUploader(sess)

	upParams := &s3manager.UploadInput{
		Bucket: &fs.BucketName,
		Key:    &path,
		// DEBUG
		// Body:   &debugReader{origReader: data, logger: fs.logger},
		Body:         data,
		CacheControl: &cacheControl,
	}

	result, err := uploader.UploadWithContext(ctx, upParams, func(u *s3manager.Uploader) {
		u.LeavePartsOnError = false
		u.PartSize = 1024 * 1024 * 10
	})
	if err != nil {
		return fmt.Errorf("error uploading %q: %w", path, err)
	}

	fs.logger.Debug(fmt.Sprintf("Uploaded %q", result.Location))

	return nil
}

// type debugReader struct {
// 	origReader io.Reader
// 	logger     hclog.Logger
// }

// func (o *debugReader) Read(p []byte) (int, error) {
// 	n, err := o.origReader.Read(p)

// 	o.logger.Debug(fmt.Sprintf("-- debugReader Read(%p) -> %d, %v", p, n, err))

// 	return n, err
// }
