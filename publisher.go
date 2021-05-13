package trdl

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
)

func GetPublisherRepository(storage logical.Storage) (*publisher.S3Repository, error) {
	awsAccessKeyID, err := GetAwsAccessKeyID() // TODO: get from vault storage, should be configured by the user
	if err != nil {
		return nil, fmt.Errorf("unable to get aws access key ID: %s", err)
	}

	awsSecretAccessKey, err := GetAwsSecretAccessKey() // TODO: get from vault storage, should be configured by the user
	if err != nil {
		return nil, fmt.Errorf("unable to get aws secret access key: %s", err)
	}

	// TODO: get from vault storage, should be configured by the user
	awsConfig := &aws.Config{
		Endpoint:    aws.String("https://storage.yandexcloud.net"),
		Region:      aws.String("ru-central1"),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	}

	// TODO: get from vault storage, should be generated automatically by the plugin, user never has an access to these private keys
	publisherKeys, err := LoadFixturePublisherKeys()
	if err != nil {
		return nil, fmt.Errorf("error loading publisher fixture keys")
	}

	// Initialize repository before any operations, to ensure everything is setup correctly before building artifact
	publisherRepository, err := publisher.NewRepositoryWithOptions(
		publisher.S3Options{AwsConfig: awsConfig, BucketName: "trdl-test-project"}, // TODO: get from vault storage, should be configured by the user
		publisher.TufRepoOptions{PrivKeys: publisherKeys},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository: %s", err)
	}

	return publisherRepository, nil
}
