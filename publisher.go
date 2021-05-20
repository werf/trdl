package trdl

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/theupdateframework/go-tuf/sign"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
)

func GetPublisherRepository(ctx context.Context, cfg *Configuration, storage logical.Storage) (*publisher.S3Repository, error) {
	awsConfig := &aws.Config{
		Endpoint:    aws.String(cfg.S3Endpoint),
		Region:      aws.String(cfg.S3Region),
		Credentials: credentials.NewStaticCredentials(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
	}

	publisherRepository, err := publisher.NewRepositoryWithOptions(
		publisher.S3Options{AwsConfig: awsConfig, BucketName: cfg.S3BucketName},
		publisher.TufRepoOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository handle: %s", err)
	}

	for _, desc := range []struct {
		storageKey    string
		targetPrivKey **sign.PrivateKey
	}{
		{storageKeyTufRepositoryRootKey, &publisherRepository.TufStore.PrivKeys.Root},
		{storageKeyTufRepositoryTargetsKey, &publisherRepository.TufStore.PrivKeys.Targets},
		{storageKeyTufRepositorySnapshotKey, &publisherRepository.TufStore.PrivKeys.Snapshot},
		{storageKeyTufRepositoryTimestampKey, &publisherRepository.TufStore.PrivKeys.Timestamp},
	} {
		entry, err := storage.Get(ctx, desc.storageKey)
		if err != nil {
			return nil, fmt.Errorf("error getting storage json entry by the key %q: %s", desc.storageKey, err)
		}

		if entry == nil {
			return nil, fmt.Errorf("%q storage key not found", desc.storageKey)
		}

		privKey := &sign.PrivateKey{}

		if err := entry.DecodeJSON(privKey); err != nil {
			return nil, fmt.Errorf("unable to decode json by the %q storage key:\n%s---\n%s", desc.storageKey, entry.Value, err)
		}

		*desc.targetPrivKey = privKey
	}

	return publisherRepository, nil
}
