package publisher

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Filesystem interface {
	IsFileExist(ctx context.Context, path string) (bool, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte) error
}

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
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("error heading s3 object by key %q: %s", path, err)
	}

	return false, nil
}

func (fs *S3Filesystem) ReadFile(ctx context.Context, path string) ([]byte, error) {
	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening s3 session: %s", err)
	}

	downloader := s3manager.NewDownloader(sess)

	file := &aws.WriteAtBuffer{}

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: &fs.BucketName,
			Key:    &path,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to download item %q: %s", path, err)
	}

	fmt.Println("Downloaded", path, numBytes, "bytes")

	return file.Bytes(), nil
}

func (fs *S3Filesystem) WriteFile(ctx context.Context, path string, data []byte) error {
	// TODO: cache opened session

	sess, err := session.NewSession(fs.AwsConfig)
	if err != nil {
		return fmt.Errorf("error opening s3 session: %s", err)
	}

	uploader := s3manager.NewUploader(sess)

	dataReader := strings.NewReader(string(data))

	upParams := &s3manager.UploadInput{
		Bucket: &fs.BucketName,
		Key:    &path,
		Body:   dataReader,
	}

	// TODO: set file mode bits

	result, err := uploader.UploadWithContext(ctx, upParams, func(u *s3manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024
		u.LeavePartsOnError = false
	})
	if err != nil {
		return fmt.Errorf("error uploading %q: %s", path, err)
	}

	fmt.Printf("Uploaded %q\n", result.Location)

	return nil
}
