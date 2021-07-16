package git

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	FieldNameGitCredentialUsername = "username"
	FieldNameGitCredentialPassword = "password"

	StorageKeyConfigurationGitCredential = "configuration_git_credential"
)

type GitCredential struct {
	Username string `structs:"username" json:"username"`
	Password string `structs:"password" json:"password"`
}

func CredentialsPaths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/git_credential/?$",
			Fields: map[string]*framework.FieldSchema{
				FieldNameGitCredentialUsername: {
					Type:        framework.TypeString,
					Description: "Git username. Required for CREATE, UPDATE.",
				},
				FieldNameGitCredentialPassword: {
					Type:        framework.TypeString,
					Description: "Git password. Required for CREATE, UPDATE.",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Configure git credential",
					Callback:    pathConfigureGitCredentialCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Configure git credential",
					Callback:    pathConfigureGitCredentialCreateOrUpdate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Reset git credential",
					Callback:    pathConfigureGitCredentialDelete,
				},
			},
		},
	}
}

func pathConfigureGitCredentialCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	gitCredential := GitCredential{
		Username: fields.Get(FieldNameGitCredentialUsername).(string),
		Password: fields.Get(FieldNameGitCredentialPassword).(string),
	}

	if gitCredential.Username == "" {
		return logical.ErrorResponse("%q field value should not be empty", FieldNameGitCredentialUsername), nil
	}
	if gitCredential.Password == "" {
		return logical.ErrorResponse("%q field value should not be empty", FieldNameGitCredentialPassword), nil
	}

	if err := PutGitCredential(ctx, req.Storage, gitCredential); err != nil {
		return nil, err
	}

	return nil, nil
}

func pathConfigureGitCredentialDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if err := DeleteGitCredential(ctx, req.Storage); err != nil {
		return nil, fmt.Errorf("unable to delete git credentials configuration: %s", err)
	}

	return nil, nil
}

func GetGitCredential(ctx context.Context, storage logical.Storage) (*GitCredential, error) {
	storageEntry, err := storage.Get(ctx, StorageKeyConfigurationGitCredential)
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var config *GitCredential
	if err := storageEntry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func PutGitCredential(ctx context.Context, storage logical.Storage, gitCredential GitCredential) error {
	storageEntry, err := logical.StorageEntryJSON(StorageKeyConfigurationGitCredential, gitCredential)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, storageEntry); err != nil {
		return err
	}

	return err
}

func DeleteGitCredential(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, StorageKeyConfigurationGitCredential)
}
