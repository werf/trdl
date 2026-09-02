package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/samber/lo"

	"github.com/werf/trdl/server/pkg/docker"
	"github.com/werf/trdl/server/pkg/elf_signing"
	"github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/pgp"
	"github.com/werf/trdl/server/pkg/publisher"
	"github.com/werf/trdl/server/pkg/secrets"
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
	fieldNameBuildkitdAddress                           = "buildkitd_address"
	fieldNameBuildxDriver                               = "buildx_driver"
	fieldNameBuildxDriverOpts                           = "buildx_driver_opts"
	fieldNameBuildkitdDriver                            = "buildkitd_driver"
	fieldNameBuildkitdDriverOpts                        = "buildkitd_driver_opts"

	storageKeyConfiguration = "configuration"
)

var errorResponseConfigurationNotFound = logical.ErrorResponse("Configuration not found")

func configurePaths(b *Backend) []*framework.Path {
	return framework.PathAppend(
		[]*framework.Path{
			configurePath(b),
			configureLastPublishedGitCommitPath(b),
		},
		git.CredentialsPaths(),
		pgp.Paths(),
		secrets.Paths(),
		mac_signing.Paths(),
		elf_signing.Paths(),
	)
}

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
				Description: "The S3 storage secret access key",
				Required:    true,
			},
			fieldNameBuildkitdAddress: {
				Type:        framework.TypeString,
				Description: "An address of a running buildkitd (unix://, tcp://, docker-container:// or kube-pod:// scheme) to build release artifacts with the BuildKit client; the docker CLI is used only when neither this nor buildkitd_driver is set. Build secrets are sent to that daemon, and tcp:// is neither encrypted nor authenticated, so securing the channel and isolating the daemon is the administrator's responsibility",
				Required:    false,
			},
			fieldNameBuildxDriver: {
				Type:        framework.TypeString,
				Description: "The buildx driver to build release artifacts with: docker-container (used by default) or kubernetes. Takes precedence over the TRDL_BUILDX_DRIVER environment variable, and cannot be combined with buildkitd_address or buildkitd_driver",
				Required:    false,
			},
			fieldNameBuildxDriverOpts: {
				Type:        framework.TypeStringSlice,
				Description: "The buildx driver options, one --driver-opt per element (e.g. namespace=trdl-build), passed through as is. Take precedence over the TRDL_BUILDX_DRIVER_OPTS_* environment variables, and cannot be combined with buildkitd_address or buildkitd_driver",
				Required:    false,
			},
			fieldNameBuildkitdDriver: {
				Type:        framework.TypeString,
				Description: "Provision an ephemeral buildkitd per build instead of using the docker CLI: kubernetes runs it as a pod and needs no docker binary next to the plugin. Cannot be combined with buildkitd_address, buildx_driver or buildx_driver_opts. A TRDL_BUILDKITD_ADDRESS set on the process wins over a stored driver, and the build reports the driver as unused",
				Required:    false,
			},
			fieldNameBuildkitdDriverOpts: {
				Type:        framework.TypeStringSlice,
				Description: "The buildkitd driver options, one name=value pair per element (e.g. namespace=trdl-build); they require buildkitd_driver to be set. The kubernetes driver accepts annotations, deadline, image, labels, limits.cpu, limits.ephemeral-storage, limits.memory, namespace, nodeselector, requests.cpu, requests.ephemeral-storage, requests.memory, rootless, serviceaccount and timeout; anything else is rejected. When deadline is not set, it defaults to the release task's remaining time at pod creation plus a five-minute margin, so a plugin crash cannot leave the builder pod running indefinitely",
				Required:    false,
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

func isConfigurationFieldSet(fields *framework.FieldData, name string) bool {
	switch value := fields.Get(name).(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case []string:
		return lo.SomeBy(value, func(item string) bool { return strings.TrimSpace(item) != "" })
	default:
		panic(fmt.Sprintf("field %q has no emptiness rule", name))
	}
}

func firstSetConfigurationField(fields *framework.FieldData, names ...string) string {
	name, _ := lo.Find(names, func(name string) bool { return isConfigurationFieldSet(fields, name) })

	return name
}

func (b *Backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	if err := docker.ValidateBuildkitdAddress(ctx, fields.Get(fieldNameBuildkitdAddress).(string)); err != nil {
		return logical.ErrorResponse("%s validation failed: %s", fieldNameBuildkitdAddress, err), nil
	}

	if err := docker.ValidateBuildxDriver(ctx, fields.Get(fieldNameBuildxDriver).(string)); err != nil {
		return logical.ErrorResponse("%s validation failed: %s", fieldNameBuildxDriver, err), nil
	}

	if err := docker.ValidateBuildkitdDriver(ctx, fields.Get(fieldNameBuildkitdDriver).(string)); err != nil {
		return logical.ErrorResponse("%s validation failed: %s", fieldNameBuildkitdDriver, err), nil
	}

	// Each of the three build backends replaces the others entirely, so a setting
	// belonging to one of the others would silently do nothing. Blank values mean
	// "not set" here, exactly as they do when the settings are resolved.
	if isConfigurationFieldSet(fields, fieldNameBuildkitdAddress) {
		if conflictingField := firstSetConfigurationField(fields, fieldNameBuildxDriver, fieldNameBuildxDriverOpts, fieldNameBuildkitdDriver, fieldNameBuildkitdDriverOpts); conflictingField != "" {
			return logical.ErrorResponse("%s cannot be combined with %s: no builder is provisioned when building against a buildkitd address", conflictingField, fieldNameBuildkitdAddress), nil
		}
	}

	if isConfigurationFieldSet(fields, fieldNameBuildkitdDriver) {
		if conflictingField := firstSetConfigurationField(fields, fieldNameBuildxDriver, fieldNameBuildxDriverOpts); conflictingField != "" {
			return logical.ErrorResponse("%s cannot be combined with %s: the docker CLI is not used when the plugin provisions buildkitd itself", conflictingField, fieldNameBuildkitdDriver), nil
		}
	}

	if err := docker.ValidateBuildkitdDriverOpts(ctx, fields.Get(fieldNameBuildkitdDriver).(string), fields.Get(fieldNameBuildkitdDriverOpts).([]string)); err != nil {
		return logical.ErrorResponse("%s validation failed: %s", fieldNameBuildkitdDriverOpts, err), nil
	}

	cfg := &configuration{
		GitRepoUrl:                    fields.Get(fieldNameGitRepoUrl).(string),
		GitTrdlPath:                   fields.Get(fieldNameGitTrdlPath).(string),
		GitTrdlChannelsPath:           fields.Get(fieldNameGitTrdlChannelsPath).(string),
		GitTrdlChannelsBranch:         fields.Get(fieldNameGitTrdlChannelsBranch).(string),
		InitialLastPublishedGitCommit: fields.Get(fieldNameInitialLastPublishedGitCommit).(string),
		RequiredNumberOfVerifiedSignaturesOnCommit: fields.Get(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(int),
		S3Endpoint:          fields.Get(fieldNameS3Endpoint).(string),
		S3Region:            fields.Get(fieldNameS3Region).(string),
		S3AccessKeyID:       fields.Get(fieldNameS3AccessKeyID).(string),
		S3SecretAccessKey:   fields.Get(fieldNameS3SecretAccessKey).(string),
		S3BucketName:        fields.Get(fieldNameS3BucketName).(string),
		BuildkitdAddress:    fields.Get(fieldNameBuildkitdAddress).(string),
		BuildxDriver:        fields.Get(fieldNameBuildxDriver).(string),
		BuildxDriverOpts:    fields.Get(fieldNameBuildxDriverOpts).([]string),
		BuildkitdDriver:     fields.Get(fieldNameBuildkitdDriver).(string),
		BuildkitdDriverOpts: fields.Get(fieldNameBuildkitdDriverOpts).([]string),
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
	GitRepoUrl                                 string   `structs:"git_repo_url" json:"git_repo_url"`
	GitTrdlPath                                string   `structs:"git_trdl_path" json:"git_trdl_path"`
	GitTrdlChannelsPath                        string   `structs:"git_trdl_channels_path" json:"git_trdl_channels_path"`
	GitTrdlChannelsBranch                      string   `structs:"git_trdl_channels_branch" json:"git_trdl_channels_branch"`
	InitialLastPublishedGitCommit              string   `structs:"initial_last_published_git_commit" json:"initial_last_published_git_commit"`
	RequiredNumberOfVerifiedSignaturesOnCommit int      `structs:"required_number_of_verified_signatures_on_commit" json:"required_number_of_verified_signatures_on_commit"`
	S3Endpoint                                 string   `structs:"s3_endpoint" json:"s3_endpoint"`
	S3Region                                   string   `structs:"s3_region" json:"s3_region"`
	S3AccessKeyID                              string   `structs:"s3_access_key_id" json:"s3_access_key_id"`
	S3SecretAccessKey                          string   `structs:"s3_secret_access_key" json:"s3_secret_access_key"`
	S3BucketName                               string   `structs:"s3_bucket_name" json:"s3_bucket_name"`
	BuildkitdAddress                           string   `structs:"buildkitd_address" json:"buildkitd_address"`
	BuildxDriver                               string   `structs:"buildx_driver" json:"buildx_driver"`
	BuildxDriverOpts                           []string `structs:"buildx_driver_opts" json:"buildx_driver_opts"`
	BuildkitdDriver                            string   `structs:"buildkitd_driver" json:"buildkitd_driver"`
	BuildkitdDriverOpts                        []string `structs:"buildkitd_driver_opts" json:"buildkitd_driver_opts"`
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

func (c *configuration) maskConfigSensitiveDataForDebug() (string, error) {
	jsonData, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &configMap); err != nil {
		return "", err
	}

	sensitiveKeys := []string{"s3_secret_access_key", "s3_access_key_id"}
	for _, key := range sensitiveKeys {
		if _, exists := configMap[key]; exists {
			configMap[key] = "******"
		}
	}

	maskedJSON, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return "", err
	}

	return string(maskedJSON), nil
}

func configureLastPublishedGitCommitPath(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern:         "configure/last_published_git_commit$",
		HelpSynopsis:    "Read or delete the last published Git commit",
		HelpDescription: "This endpoint allows reading or deleting the last published Git commit recorded by the plugin.",
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Description: "Get the last published Git commit",
				Callback:    b.pathConfigureLastPublishedGitCommitRead,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Description: "Delete the last published Git commit",
				Callback:    b.pathConfigureLastPublishedGitCommitDelete,
			},
		},
	}
}

func (b *Backend) pathConfigureLastPublishedGitCommitRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entry, err := req.Storage.Get(ctx, storageKeyLastPublishedGitCommit)
	if err != nil {
		return nil, fmt.Errorf("unable to get %q from storage: %w", storageKeyLastPublishedGitCommit, err)
	}

	var lastPublishedGitCommit string
	if entry != nil {
		lastPublishedGitCommit = string(entry.Value)
	}

	return &logical.Response{
		Data: map[string]interface{}{
			storageKeyLastPublishedGitCommit: lastPublishedGitCommit,
		},
	}, nil
}

func (b *Backend) pathConfigureLastPublishedGitCommitDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	return nil, req.Storage.Delete(ctx, storageKeyLastPublishedGitCommit)
}
