package publisher

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func (publisher *Publisher) Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:      "configure/pgp_signing_key",
			HelpSynopsis: "Configure a PGP key for signing release artifacts",
			Fields:       map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get a public part of a PGP signing key",
					Callback:    publisher.pathConfigurePGPSigningKeyRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Summary:     "Delete the current PGP signing key",
					Description: "Delete the current PGP signing key (new key will be generated automatically on demand)",
					Callback:    publisher.pathConfigurePGPSigningKeyDelete,
				},
			},
		},
	}
}

func (publisher *Publisher) pathConfigurePGPSigningKeyRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	key, err := publisher.fetchPGPSigningKey(ctx, req.Storage, true)
	if err != nil {
		return nil, fmt.Errorf("error fetching pgp signing key: %s", err)
	}

	pk := bytes.NewBuffer(nil)
	if err := key.SerializePublicKey(pk); err != nil {
		return nil, fmt.Errorf("unable to get public key text: %s", err)
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"public_key": pk.String(),
		},
	}, nil
}

func (publisher *Publisher) pathConfigurePGPSigningKeyDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if err := publisher.deletePGPSigningKey(ctx, req.Storage); err != nil {
		return nil, fmt.Errorf("error deleting pgp signing key: %s", err)
	}
	return &logical.Response{Data: map[string]interface{}{}}, nil
}
