package trdl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	fieldNameLastPublishSuccessfulCommit                = "last_successful_commit"
	// TODO
	//fieldNameTaskTimeout                                = "task_timeout"
	//fieldNameTaskHistoryLimit                           = "task_history_limit"
	fieldNameGitCredentialUsername   = "username"
	fieldNameGitCredentialPassword   = "password"
	fieldNameTrustedGpgPublicKeyName = "name"
	fieldNameTrustedGpgPublicKeyData = "public_key"

	storageKeyConfigurationBase          = "configuration_base"
	storageKeyConfigurationGitCredential = "configuration_git_credential"
	storageKeyPrefixTrustedGPGPublicKey  = "trusted_gpg_public_key-"
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
				fieldNameLastPublishSuccessfulCommit: {
					Type:        framework.TypeString,
					Description: "The commit on which the publication was successfully completed",
				},
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

	entry, err := logical.StorageEntryJSON(storageKeyConfigurationBase, fields.Raw)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
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
