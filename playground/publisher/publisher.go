package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/keyhelper"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
)

type Snapshot struct {
	Files map[string][]byte
}

func doMain() error {
	privKeys := publisher.TufRepoPrivKeys{}

	{
		path := os.Getenv("TRDL_ROOT_KEY_PATH")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open %q: %s", path, err)
		}
		if keys, err := keyhelper.LoadKeys(file, []byte(os.Getenv("TRDL_ROOT_KEY_PASSPHRASE"))); err != nil {
			return fmt.Errorf("error loading %q: %s", path, err)
		} else {
			for _, key := range keys {
				privKeys.Root = append(privKeys.Root, key.Signer())
			}
		}
	}

	{
		path := os.Getenv("TRDL_SNAPSHOT_KEY_PATH")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open %q: %s", path, err)
		}
		if keys, err := keyhelper.LoadKeys(file, []byte(os.Getenv("TRDL_SNAPSHOT_KEY_PASSPHRASE"))); err != nil {
			return fmt.Errorf("error loading %q: %s", path, err)
		} else {
			for _, key := range keys {
				privKeys.Snapshot = append(privKeys.Snapshot, key.Signer())
			}
		}
	}

	{
		path := os.Getenv("TRDL_TARGETS_KEY_PATH")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open %q: %s", path, err)
		}
		if keys, err := keyhelper.LoadKeys(file, []byte(os.Getenv("TRDL_TARGETS_KEY_PASSPHRASE"))); err != nil {
			return fmt.Errorf("error loading %q: %s", path, err)
		} else {
			for _, key := range keys {
				privKeys.Targets = append(privKeys.Targets, key.Signer())
			}
		}
	}

	{
		path := os.Getenv("TRDL_TIMESTAMP_KEY_PATH")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open %q: %s", path, err)
		}
		if keys, err := keyhelper.LoadKeys(file, []byte(os.Getenv("TRDL_TIMESTAMP_KEY_PASSPHRASE"))); err != nil {
			return fmt.Errorf("error loading %q: %s", path, err)
		} else {
			for _, key := range keys {
				privKeys.Timestamp = append(privKeys.Timestamp, key.Signer())
			}
		}
	}

	fmt.Printf("privKeys: %#v\n", privKeys)

	awsConfig := &aws.Config{
		Endpoint:    aws.String("https://storage.yandexcloud.net"),
		Region:      aws.String("ru-central1"),
		Credentials: credentials.NewStaticCredentials(os.Getenv("TRDL_AWS_ACCESS_KEY_ID"), os.Getenv("TRDL_AWS_SECRET_ACCESS_KEY"), ""),
	}

	snapshotFiles := publisher.NewInMemorySnapshotFiles()
	snapshotFiles.Files = append(snapshotFiles.Files, publisher.InMemoryFile{
		Name: "a/b/c/data.txt",
		Data: []byte("HELLO WORLD"),
	})

	return publisher.PublishSnapshotIntoS3(context.Background(), snapshotFiles.Iterator(),
		publisher.TufRepoOptions{PrivKeys: privKeys},
		publisher.S3Options{AwsConfig: awsConfig, BucketName: "trdl-test-project"})
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
