package secrets

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameSecretId   = "id"
	fieldNameSecretData = "data"
)

type Secret struct {
	Id   string
	Data []byte
}

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/build/secrets/?",
			HelpSynopsis:    "Add a build secret",
			HelpDescription: "Add a build secret",
			Fields: map[string]*framework.FieldSchema{
				fieldNameSecretId: {
					Type:        framework.TypeNameString,
					Description: "Secret Id",
					Required:    true,
				},
				fieldNameSecretData: {
					Type:        framework.TypeString,
					Description: "Secret data",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Add a build secret",
					Callback:    pathSecretCreate,
				},
			},
		},
		{
			Pattern:         "configure/build/secrets/" + framework.GenericNameRegex(fieldNameSecretId) + "$",
			HelpSynopsis:    "Delete a build secret",
			HelpDescription: "Delete a build secret",
			Fields: map[string]*framework.FieldSchema{
				fieldNameSecretId: {
					Type:        framework.TypeNameString,
					Description: "Secret Id",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Delete a build secret",
					Callback:    pathSecretDelete,
				},
			},
		},
	}
}

func pathSecretCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}
	return CreateSecret(ctx, req, secretStorage{
		Id:   fields.Get(fieldNameSecretId).(string),
		Data: fields.Get(fieldNameSecretData).(string),
	})
}

func pathSecretDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}
	err := DeleteSecret(ctx, req, secretStorage{
		Id: fields.Get(fieldNameSecretId).(string),
	})
	if err != nil {
		return nil, fmt.Errorf("error delete secret: %w", err)
	}
	return nil, nil
}
