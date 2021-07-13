package trdl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/sign"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameGitTrdlChannelsBranch                      = "git_trdl_channels_branch"
	fieldNameInitialLastPublishedGitCommit              = "initial_last_published_git_commit"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	fieldNameS3Endpoint                                 = "s3_endpoint"
	fieldNameS3Region                                   = "s3_region"
	fieldNameS3AccessKeyID                              = "s3_access_key_id"
	fieldNameS3SecretAccessKey                          = "s3_secret_access_key"
	fieldNameS3BucketName                               = "s3_bucket_name"

	storageKeyConfiguration             = "configuration"
	storageKeyTufRepositoryRootKey      = "tuf_repository_root_key"
	storageKeyTufRepositoryTargetsKey   = "tuf_repository_targets_key"
	storageKeyTufRepositorySnapshotKey  = "tuf_repository_snapshot_key"
	storageKeyTufRepositoryTimestampKey = "tuf_repository_timestamp_key"
)

func configurePath(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "configure/?",
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitRepoUrl: {
				Type:        framework.TypeString,
				Description: "Git repository url",
				Required:    true,
			},
			fieldNameGitTrdlChannelsBranch: {
				Type:        framework.TypeString,
				Description: `Set branch of the configured git repository which contains trdl_channels.yaml configuration (will use "trdl" branch by default)`,
				Required:    false,
			},
			fieldNameInitialLastPublishedGitCommit: {
				Type:        framework.TypeString,
				Description: "Set or override last published git commit which contains trdl channels",
				Required:    false,
			},
			fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: {
				Type:        framework.TypeInt,
				Description: "Required number of verified signatures on commit",
				Required:    true,
			},
			fieldNameS3BucketName: {
				Type:        framework.TypeString,
				Description: "S3 storage bucket name",
				Required:    true,
			},
			fieldNameS3Endpoint: {
				Type:        framework.TypeString,
				Description: "S3 storage endpoint",
				Required:    true,
			},
			fieldNameS3Region: {
				Type:        framework.TypeString,
				Description: "S3 storage region",
				Required:    true,
			},
			fieldNameS3AccessKeyID: {
				Type:        framework.TypeString,
				Description: "S3 storage access key id",
				Required:    true,
			},
			fieldNameS3SecretAccessKey: {
				Type:        framework.TypeString,
				Description: "S3 storage access key id",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathConfigureCreateOrUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigureCreateOrUpdate,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigureRead,
			},
		},
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if err := util.CheckRequiredFields(req, fields); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	c := &configuration{
		GitRepoUrl:                                 fields.Get(fieldNameGitRepoUrl).(string),
		GitTrdlChannelsBranch:                      fields.Get(fieldNameGitTrdlChannelsBranch).(string),
		InitialLastPublishedGitCommit:              fields.Get(fieldNameInitialLastPublishedGitCommit).(string),
		RequiredNumberOfVerifiedSignaturesOnCommit: fields.Get(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(int),
		S3Endpoint:                                 fields.Get(fieldNameS3Endpoint).(string),
		S3Region:                                   fields.Get(fieldNameS3Region).(string),
		S3AccessKeyID:                              fields.Get(fieldNameS3AccessKeyID).(string),
		S3SecretAccessKey:                          fields.Get(fieldNameS3SecretAccessKey).(string),
		S3BucketName:                               fields.Get(fieldNameS3BucketName).(string),
	}

	if err := b.initTufRepoAndRelatedData(ctx, req.Storage, c); err != nil {
		return nil, fmt.Errorf("unable to init tuf repo and related data: %s", err)
	}

	if err := putConfiguration(ctx, req.Storage, c); err != nil {
		return nil, fmt.Errorf("unable to put configuration into storage: %s", err)
	}

	return nil, nil
}

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	cfg, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration: %s", err)
	}

	if cfg == nil {
		return logical.ErrorResponse("configuration not found"), nil
	}

	return &logical.Response{Data: structs.Map(cfg)}, nil
}

func (b *backend) initTufRepoAndRelatedData(ctx context.Context, storage logical.Storage, c *configuration) error {
	awsConfig := &aws.Config{
		Endpoint:    aws.String(c.S3Endpoint),
		Region:      aws.String(c.S3Region),
		Credentials: credentials.NewStaticCredentials(c.S3AccessKeyID, c.S3SecretAccessKey, ""),
	}

	publisherRepository, err := publisher.NewRepositoryWithOptions(
		publisher.S3Options{AwsConfig: awsConfig, BucketName: c.S3BucketName},
		publisher.TufRepoOptions{},
	)
	if err != nil {
		return fmt.Errorf("error initializing publisher repository: %s", err)
	}

	if err := publisherRepository.TufRepo.Init(false); err == tuf.ErrInitNotAllowed {
		if os.Getenv("TRDL_DEV") != "1" {
			return fmt.Errorf("found existing targets in the tuf repository in the s3 storage, cannot reinitialize already initialized repository. Please use new s3 bucket or remove existing targets")
		}
	} else if err != nil {
		return fmt.Errorf("unable to init tuf repository: %s", err)
	}

	if os.Getenv("TRDL_DEV") == "1" {
		devKeys, err := LoadDevPublisherKeys()
		if err != nil {
			return fmt.Errorf("error loading dev mode publisher keys: %s", err)
		}

		if err := publisherRepository.SetPrivKeys(devKeys); err != nil {
			return fmt.Errorf("unable to set dev private keys: %s", err)
		}
	} else {
		_, err = publisherRepository.TufRepo.GenKey("root")
		if err != nil {
			return fmt.Errorf("error generating tuf repository root key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("targets")
		if err != nil {
			return fmt.Errorf("error generating tuf repository targets key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("snapshot")
		if err != nil {
			return fmt.Errorf("error generating tuf repository snapshot key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("timestamp")
		if err != nil {
			return fmt.Errorf("error generating tuf repository timestamp key: %s", err)
		}
	}

	for _, storeKey := range []struct {
		Key        *sign.PrivateKey
		StorageKey string
	}{
		{publisherRepository.TufStore.PrivKeys.Root, storageKeyTufRepositoryRootKey},
		{publisherRepository.TufStore.PrivKeys.Targets, storageKeyTufRepositoryTargetsKey},
		{publisherRepository.TufStore.PrivKeys.Snapshot, storageKeyTufRepositorySnapshotKey},
		{publisherRepository.TufStore.PrivKeys.Timestamp, storageKeyTufRepositoryTimestampKey},
	} {
		entry, err := logical.StorageEntryJSON(storeKey.StorageKey, storeKey.Key)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %s", storeKey.StorageKey, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting json entry by key %q into the storage: %s", storeKey.StorageKey, err)
		}
	}

	if err := publisherRepository.PublishTarget(ctx, "initialized_at", bytes.NewBuffer([]byte(time.Now().UTC().String()+"\n"))); err != nil {
		return fmt.Errorf("unable to publish initialization timestamp: %s", err)
	}

	if err := publisherRepository.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit initialized tuf repository: %s", err)
	}

	return nil
}

type configuration struct {
	GitRepoUrl                                 string `structs:"git_repo_url" json:"git_repo_url"`
	GitTrdlChannelsBranch                      string `structs:"git_trdl_channels_branch" json:"git_trdl_channels_branch"`
	InitialLastPublishedGitCommit              string `structs:"initial_last_published_git_commit" json:"initial_last_published_git_commit"`
	RequiredNumberOfVerifiedSignaturesOnCommit int    `structs:"required_number_of_verified_signatures_on_commit" json:"required_number_of_verified_signatures_on_commit"`
	S3Endpoint                                 string `structs:"s3_endpoint" json:"s3_endpoint"`
	S3Region                                   string `structs:"s3_region" json:"s3_region"`
	S3AccessKeyID                              string `structs:"s3_access_key_id" json:"s3_access_key_id"`
	S3SecretAccessKey                          string `structs:"s3_secret_access_key" json:"s3_secret_access_key"`
	S3BucketName                               string `structs:"s3_bucket_name" json:"s3_bucket_name"`
}

func getConfiguration(ctx context.Context, storage logical.Storage) (*configuration, error) {
	raw, err := storage.Get(ctx, storageKeyConfiguration)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	config := new(configuration)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func putConfiguration(ctx context.Context, storage logical.Storage, config *configuration) error {
	entry, err := logical.StorageEntryJSON(storageKeyConfiguration, config)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, entry); err != nil {
		return err
	}

	return err
}
