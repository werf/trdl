package pgp

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameTrustedPGPPublicKeyName = "name"
	fieldNameTrustedPGPPublicKeyData = "public_key"
)

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/trusted_pgp_public_key/?",
			HelpSynopsis:    "Configure trusted PGP public keys",
			HelpDescription: "Configure trusted PGP public keys to check git repository commit signatures",
			Fields: map[string]*framework.FieldSchema{
				fieldNameTrustedPGPPublicKeyName: {
					Type:        framework.TypeNameString,
					Description: "Key name",
					Required:    true,
				},
				fieldNameTrustedPGPPublicKeyData: {
					Type:        framework.TypeString,
					Description: "Key data",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Add a trusted PGP public key",
					Callback:    pathConfigureTrustedPGPPublicKeyCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Add a trusted PGP public key",
					Callback:    pathConfigureTrustedPGPPublicKeyCreateOrUpdate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get the list of trusted PGP public keys",
					Callback:    pathConfigureTrustedPGPPublicKeyReadOrList,
				},
				logical.ListOperation: &framework.PathOperation{
					Description: "Get the list of trusted PGP public keys",
					Callback:    pathConfigureTrustedPGPPublicKeyReadOrList,
				},
			},
		},
		{
			Pattern:         "configure/trusted_pgp_public_key/" + framework.GenericNameRegex(fieldNameTrustedPGPPublicKeyName) + "$",
			HelpSynopsis:    "Read or delete the configured trusted PGP public key",
			HelpDescription: "Read or delete the configured trusted PGP public key",
			Fields: map[string]*framework.FieldSchema{
				fieldNameTrustedPGPPublicKeyName: {
					Type:        framework.TypeNameString,
					Description: "Key name",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get the trusted PGP public key",
					Callback:    pathConfigureTrustedPGPPublicKeyRead,
				},
				logical.ListOperation: &framework.PathOperation{
					Description: "Get the trusted PGP public key",
					Callback:    pathConfigureTrustedPGPPublicKeyRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Delete the trusted PGP public key",
					Callback:    pathConfigureTrustedPGPPublicKeyDelete,
				},
			},
		},
	}
}

func pathConfigureTrustedPGPPublicKeyCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	name := fields.Get(fieldNameTrustedPGPPublicKeyName).(string)
	key := fields.Get(fieldNameTrustedPGPPublicKeyData).(string)

	if err := req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   trustedPGPPublicKeyStorageKey(name),
		Value: []byte(key),
	}); err != nil {
		return nil, fmt.Errorf("unable to put trusted pgp public key: %s", err)
	}

	return nil, nil
}

func pathConfigureTrustedPGPPublicKeyReadOrList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	list, err := req.Storage.List(ctx, storageKeyPrefixTrustedPGPPublicKey)
	if err != nil {
		return nil, fmt.Errorf("unable to list %q in storage: %s", storageKeyPrefixTrustedPGPPublicKey, err)
	}

	return logical.ListResponse(list), nil
}

func pathConfigureTrustedPGPPublicKeyRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get(fieldNameTrustedPGPPublicKeyName).(string)

	e, err := req.Storage.Get(ctx, trustedPGPPublicKeyStorageKey(name))
	if err != nil {
		return nil, err
	}

	if e == nil {
		return logical.ErrorResponse("PGP public key %q not found in storage", name), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"name":       name,
			"public_key": string(e.Value),
		},
	}, nil
}

func pathConfigureTrustedPGPPublicKeyDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get(fieldNameTrustedPGPPublicKeyName).(string)
	if err := req.Storage.Delete(ctx, trustedPGPPublicKeyStorageKey(name)); err != nil {
		return nil, err
	}

	return nil, nil
}
