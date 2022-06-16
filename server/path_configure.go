package server

import (
	"context"
	"fmt"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/publisher"
	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameGitTrdlPath                                = "git_trdl_path"
	fieldNameGitTrdlChannelsPath                        = "git_trdl_channels_path"
	fieldNameGitTrdlChannelsBranch                      = "git_trdl_channels_branch"
	fieldNameInitialLastPublishedGitCommit              = "initial_last_published_git_commit"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	fieldNameS3Endpoint                                 = "s3_endpoint"
	fieldNameS3Region                                   = "s3_region"
	fieldNameS3AccessKeyID                              = "s3_access_key_id"
	fieldNameS3SecretAccessKey                          = "s3_secret_access_key"
	fieldNameS3BucketName                               = "s3_bucket_name"

	storageKeyConfiguration = "configuration"
)

var errorResponseConfigurationNotFound = logical.ErrorResponse("Configuration not found")

func configurePath(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern:      "configure/?",
		HelpSynopsis: "Configure the plugin",
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitRepoUrl: {
				Type:        framework.TypeString,
				Description: "URL of the Git repository",
				Required:    true,
			},
			fieldNameGitTrdlPath: {
				Type:        framework.TypeString,
				Description: "A path in the Git repository to the release trdl configuration file (trdl.yaml is used by default)",
				Required:    false,
			},
			fieldNameGitTrdlChannelsPath: {
				Type:        framework.TypeString,
				Description: "A path in the Git repository to the trdl channels configuration file (trdl_channels.yaml is used by default)",
				Required:    false,
			},
			fieldNameGitTrdlChannelsBranch: {
				Type:        framework.TypeString,
				Description: "A special Git branch to store the trdl channels configuration file",
				Required:    false,
			},
			fieldNameInitialLastPublishedGitCommit: {
				Type:        framework.TypeString,
				Description: "The initial commit for the last successful publication",
				Required:    false,
			},
			fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: {
				Type:        framework.TypeInt,
				Description: "The required number of verified signatures for a commit",
				Required:    true,
			},
			fieldNameS3BucketName: {
				Type:        framework.TypeString,
				Description: "The S3 storage bucket name",
				Required:    true,
			},
			fieldNameS3Endpoint: {
				Type:        framework.TypeString,
				Description: "The S3 storage endpoint",
				Required:    true,
			},
			fieldNameS3Region: {
				Type:        framework.TypeString,
				Description: "The S3 storage region",
				Required:    true,
			},
			fieldNameS3AccessKeyID: {
				Type:        framework.TypeString,
				Description: "The S3 storage access key id",
				Required:    true,
			},
			fieldNameS3SecretAccessKey: {
				Type:        framework.TypeString,
				Description: "The S3 storage access key id",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Description: "Configure plugin",
				Callback:    b.pathConfigureCreateOrUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Description: "Configure plugin",
				Callback:    b.pathConfigureCreateOrUpdate,
			},
			logical.ReadOperation: &framework.PathOperation{
				Description: "Read the plugin configuration",
				Callback:    b.pathConfigureRead,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Description: "Reset the plugin configuration",
				Callback:    b.pathConfigureDelete,
			},
		},
	}
}

func (b *Backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	cfg := &configuration{
		GitRepoUrl:                    fields.Get(fieldNameGitRepoUrl).(string),
		GitTrdlPath:                   fields.Get(fieldNameGitTrdlPath).(string),
		GitTrdlChannelsPath:           fields.Get(fieldNameGitTrdlChannelsPath).(string),
		GitTrdlChannelsBranch:         fields.Get(fieldNameGitTrdlChannelsBranch).(string),
		InitialLastPublishedGitCommit: fields.Get(fieldNameInitialLastPublishedGitCommit).(string),
		RequiredNumberOfVerifiedSignaturesOnCommit: fields.Get(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(int),
		S3Endpoint:        fields.Get(fieldNameS3Endpoint).(string),
		S3Region:          fields.Get(fieldNameS3Region).(string),
		S3AccessKeyID:     fields.Get(fieldNameS3AccessKeyID).(string),
		S3SecretAccessKey: fields.Get(fieldNameS3SecretAccessKey).(string),
		S3BucketName:      fields.Get(fieldNameS3BucketName).(string),
	}

	if err := putConfiguration(ctx, req.Storage, cfg); err != nil {
		return nil, fmt.Errorf("unable to put configuration into storage: %w", err)
	}

	return nil, nil
}

func (b *Backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	cfg, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration: %w", err)
	}

	if cfg == nil {
		return errorResponseConfigurationNotFound, nil
	}

	return &logical.Response{Data: structs.Map(cfg)}, nil
}

func (b *Backend) pathConfigureDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := deleteConfiguration(ctx, req.Storage); err != nil {
		return nil, fmt.Errorf("unable to delete configuration: %w", err)
	}

	return nil, nil
}

type configuration struct {
	GitRepoUrl                                 string `structs:"git_repo_url" json:"git_repo_url"`
	GitTrdlPath                                string `structs:"git_trdl_path" json:"git_trdl_path"`
	GitTrdlChannelsPath                        string `structs:"git_trdl_channels_path" json:"git_trdl_channels_path"`
	GitTrdlChannelsBranch                      string `structs:"git_trdl_channels_branch" json:"git_trdl_channels_branch"`
	InitialLastPublishedGitCommit              string `structs:"initial_last_published_git_commit" json:"initial_last_published_git_commit"`
	RequiredNumberOfVerifiedSignaturesOnCommit int    `structs:"required_number_of_verified_signatures_on_commit" json:"required_number_of_verified_signatures_on_commit"`
	S3Endpoint                                 string `structs:"s3_endpoint" json:"s3_endpoint"`
	S3Region                                   string `structs:"s3_region" json:"s3_region"`
	S3AccessKeyID                              string `structs:"s3_access_key_id" json:"s3_access_key_id"`
	S3SecretAccessKey                          string `structs:"s3_secret_access_key" json:"s3_secret_access_key"`
	S3BucketName                               string `structs:"s3_bucket_name" json:"s3_bucket_name"`
}

func (cfg *configuration) RepositoryOptions() publisher.RepositoryOptions {
	return publisher.RepositoryOptions{
		S3Endpoint:        cfg.S3Endpoint,
		S3Region:          cfg.S3Region,
		S3AccessKeyID:     cfg.S3AccessKeyID,
		S3SecretAccessKey: cfg.S3SecretAccessKey,
		S3BucketName:      cfg.S3BucketName,
	}
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

func deleteConfiguration(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, storageKeyConfiguration)
}
