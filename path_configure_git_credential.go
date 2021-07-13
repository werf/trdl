package trdl

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitCredentialUsername = "username"
	fieldNameGitCredentialPassword = "password"

	storageKeyGitCredential = "git_credential"
)

func configureGitCredentialPath(b *backend) *framework.Path {
	return &framework.Path{
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
				Callback: b.pathConfigureGitCredentialCreateOrUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigureGitCredentialCreateOrUpdate,
			},
		},
	}
}

func (b *backend) pathConfigureGitCredentialCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if err := util.CheckRequiredFields(req, fields); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	credential := &gitCredential{
		Username: fields.Get(fieldNameGitCredentialUsername).(string),
		Password: fields.Get(fieldNameGitCredentialPassword).(string),
	}

	if err := putGitCredential(ctx, req.Storage, credential); err != nil {
		return nil, fmt.Errorf("unable to put git credential into storage: %s", err)
	}

	return nil, nil
}

type gitCredential struct {
	Username string `structs:"username" json:"username"`
	Password string `structs:"password" json:"password"`
}

func getGitCredential(ctx context.Context, storage logical.Storage) (*gitCredential, error) {
	raw, err := storage.Get(ctx, storageKeyGitCredential)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	credential := new(gitCredential)
	if err := raw.DecodeJSON(credential); err != nil {
		return nil, err
	}

	return credential, nil
}

func putGitCredential(ctx context.Context, storage logical.Storage, credential *gitCredential) error {
	entry, err := logical.StorageEntryJSON(storageKeyGitCredential, credential)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, entry); err != nil {
		return err
	}

	return err
}
