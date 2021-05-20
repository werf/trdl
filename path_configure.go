package trdl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/sign"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	// TODO
	// fieldNameLastPublishSuccessfulCommit                = "last_successful_commit"
	// TODO
	// fieldNameTaskTimeout                                = "task_timeout"
	// fieldNameTaskHistoryLimit                           = "task_history_limit"
	fieldNameGitCredentialUsername   = "username"
	fieldNameGitCredentialPassword   = "password"
	fieldNameTrustedGpgPublicKeyName = "name"
	fieldNameTrustedGpgPublicKeyData = "public_key"

	fieldNameS3Endpoint        = "s3_endpoint"
	fieldNameS3Region          = "s3_region"
	fieldNameS3AccessKeyID     = "s3_access_key_id"
	fieldNameS3SecretAccessKey = "s3_secret_access_key"
	fieldNameS3BucketName      = "s3_bucket_name"

	storageKeyConfigurationBase          = "configuration_base"
	storageKeyConfigurationGitCredential = "configuration_git_credential"
	storageKeyPrefixTrustedGPGPublicKey  = "trusted_gpg_public_key-"

	storageKeyTufRepositoryRootKey      = "tuf_repository_root_key"
	storageKeyTufRepositoryTargetsKey   = "tuf_repository_targets_key"
	storageKeyTufRepositorySnapshotKey  = "tuf_repository_snapshot_key"
	storageKeyTufRepositoryTimestampKey = "tuf_repository_timestamp_key"
)

func configurePaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "configure/?",
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitRepoUrl: {
					Type:        framework.TypeString,
					Description: "Git repository url",
					Required:    true,
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

				// TODO
				//fieldNameLastPublishSuccessfulCommit: {
				//	Type:        framework.TypeString,
				//	Description: "The commit on which the publication was successfully completed",
				//},
				// TODO
				//fieldNameTaskTimeout: {
				//	Type:        framework.TypeDurationSecond,
				//	Description: "Task time limit",
				//	Default:     "10m",
				//},
				//fieldNameTaskHistoryLimit: {
				//	Type:        framework.TypeInt,
				//	Description: "Task history limit",
				//	Default:     10,
				//},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigure,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigure,
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureRead,
				},
			},
		},
		{
			Pattern: "configure/git_credential/?",
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitCredentialUsername: {
					Type:        framework.TypeString,
					Description: "Git username",
					Required:    true,
				},
				fieldNameGitCredentialPassword: {
					Type:        framework.TypeString,
					Description: "Git password",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredential,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredential,
				},
			},
		},
		{
			Pattern: "configure/trusted_gpg_public_key",
			Fields: map[string]*framework.FieldSchema{
				fieldNameTrustedGpgPublicKeyName: {
					Type:        framework.TypeNameString,
					Description: "Trusted GPG public key name",
					Required:    true,
				},
				fieldNameTrustedGpgPublicKeyData: {
					Type:        framework.TypeString,
					Description: "Trusted GPG public key",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureTrustedGPGPublicKeyCreate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureTrustedGPGPublicKeyCreate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureTrustedGPGPublicKeyList,
				},
			},
		},
		{
			Pattern: "configure/trusted_gpg_public_key/" + framework.GenericNameRegex(fieldNameTrustedGpgPublicKeyName) + "$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameTrustedGpgPublicKeyName: {
					Type:        framework.TypeNameString,
					Description: "Trusted GPG public key name",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureTrustedGPGPublicKeyRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureTrustedGPGPublicKeyDelete,
				},
			},
		},
	}
}

func (b *backend) pathConfigure(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	convertedFields := make(map[string]interface{})
	for k := range fields.Raw {
		v, ok := fields.GetOk(k)
		if !ok {
			panic(fmt.Sprintf("bad field value %q", k))
		}

		convertedFields[k] = v
	}

	entry, err := logical.StorageEntryJSON(storageKeyConfigurationBase, convertedFields)
	if err != nil {
		return nil, fmt.Errorf("error creating storage json entry: %s", err)
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, fmt.Errorf("error putting json entry into the storage: %s", err)
	}

	s3BucketName := req.GetString(fieldNameS3BucketName)
	s3AccessKeyID := req.GetString(fieldNameS3AccessKeyID)
	s3SecretAccessKey := req.GetString(fieldNameS3SecretAccessKey)
	s3Endpoint := req.GetString(fieldNameS3Endpoint)
	s3Region := req.GetString(fieldNameS3Region)

	awsConfig := &aws.Config{
		Endpoint:    aws.String(s3Endpoint),
		Region:      aws.String(s3Region),
		Credentials: credentials.NewStaticCredentials(s3AccessKeyID, s3SecretAccessKey, ""),
	}

	publisherRepository, err := publisher.NewRepositoryWithOptions(
		publisher.S3Options{AwsConfig: awsConfig, BucketName: s3BucketName},
		publisher.TufRepoOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository: %s", err)
	}

	if err := publisherRepository.TufRepo.Init(false); err == tuf.ErrInitNotAllowed {
		if os.Getenv("TRDL_DEV") != "1" {
			return nil, fmt.Errorf("Found existing targets in the tuf repository in the s3 storage, cannot reinitialize already initialized repository. Please use new s3 bucket or remove existing targets.")
		}
	} else if err != nil {
		return nil, fmt.Errorf("unable to init tuf repository: %s", err)
	}

	if os.Getenv("TRDL_DEV") == "1" {
		devKeys, err := LoadDevPublisherKeys()
		if err != nil {
			return nil, fmt.Errorf("error loading dev mode publisher keys: %s", err)
		}

		if err := publisherRepository.SetPrivKeys(devKeys); err != nil {
			return nil, fmt.Errorf("unable to set dev private keys: %s", err)
		}
	} else {
		_, err = publisherRepository.TufRepo.GenKey("root")
		if err != nil {
			return nil, fmt.Errorf("error generating tuf repository root key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("targets")
		if err != nil {
			return nil, fmt.Errorf("error generating tuf repository targets key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("snapshot")
		if err != nil {
			return nil, fmt.Errorf("error generating tuf repository snapshot key: %s", err)
		}

		_, err = publisherRepository.TufRepo.GenKey("timestamp")
		if err != nil {
			return nil, fmt.Errorf("error generating tuf repository timestamp key: %s", err)
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
			return nil, fmt.Errorf("error creating storage json entry by key %q: %s", storeKey.StorageKey, err)
		}

		if err := req.Storage.Put(ctx, entry); err != nil {
			return nil, fmt.Errorf("error putting json entry by key %q into the storage: %s", storeKey.StorageKey, err)
		}
	}

	if err := publisherRepository.PublishTarget(ctx, "initialized_at", bytes.NewBuffer([]byte(time.Now().UTC().String()+"\n"))); err != nil {
		return nil, fmt.Errorf("unable to publish initialization timestamp: %s", err)
	}

	if err := publisherRepository.Commit(ctx); err != nil {
		return nil, fmt.Errorf("unable to commit initialized tuf repository: %s", err)
	}

	return nil, nil
}

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	v, err := req.Storage.Get(ctx, storageKeyConfigurationBase)
	if err != nil {
		return nil, fmt.Errorf("unable to get storage entry %q: %s", storageKeyConfigurationBase, err)
	}

	if v == nil {
		return logical.ErrorResponse("configuration not found"), nil
	}

	var res map[string]interface{}
	if err := json.Unmarshal(v.Value, &res); err != nil {
		return nil, fmt.Errorf("unable to unmarshal storage entry %q: %s", storageKeyConfigurationBase, err)
	}

	return &logical.Response{Data: res}, nil
}

func (b *backend) pathConfigureGitCredential(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	entry, err := logical.StorageEntryJSON(storageKeyConfigurationGitCredential, fields.Raw)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureTrustedGPGPublicKeyList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	list, err := req.Storage.List(ctx, storageKeyPrefixTrustedGPGPublicKey)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"names": list,
		},
	}, nil
}

func (b *backend) pathConfigureTrustedGPGPublicKeyRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get(fieldNameTrustedGpgPublicKeyName).(string)
	e, err := req.Storage.Get(ctx, storageKeyPrefixTrustedGPGPublicKey+name)
	if err != nil {
		return nil, err
	}

	if e == nil {
		return logical.ErrorResponse(fmt.Sprintf("GPG public key %q not found in storage", name)), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"name":       name,
			"public_key": string(e.Value),
		},
	}, nil
}

func (b *backend) pathConfigureTrustedGPGPublicKeyCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get(fieldNameTrustedGpgPublicKeyName).(string)
	key := fields.Get(fieldNameTrustedGpgPublicKeyData).(string)
	if err := req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKeyPrefixTrustedGPGPublicKey + name,
		Value: []byte(key),
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureTrustedGPGPublicKeyDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get(fieldNameTrustedGpgPublicKeyName).(string)
	if err := req.Storage.Delete(ctx, storageKeyPrefixTrustedGPGPublicKey+name); err != nil {
		return nil, err
	}

	return nil, nil
}
