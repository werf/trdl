package mac_signing

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameMacSigningName         = "name"
	fieldNameMacSigningCertificate  = "certificate"
	fieldNameMacSigningPassword     = "password"
	fieldNameMacSigningNotaryKeyID  = "notary_key_id"
	fieldNameMacSigningNotaryKey    = "notary_key"
	fieldNameMacSigningNotaryIssuer = "notary_issuer"
)

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/build/mac_signing/?",
			HelpSynopsis:    "Add or update build signing credentials",
			HelpDescription: "Add or update build signing credentials for macOS builds",
			Fields: map[string]*framework.FieldSchema{
				fieldNameMacSigningName: {
					Type:        framework.TypeNameString,
					Description: "Credentials name",
					Required:    true,
				},
				fieldNameMacSigningCertificate: {
					Type:        framework.TypeString,
					Description: "Certificate data base64 encoded",
					Required:    true,
				},
				fieldNameMacSigningPassword: {
					Type:        framework.TypeString,
					Description: "Certificate password",
					Required:    true,
				},
				fieldNameMacSigningNotaryKeyID: {
					Type:        framework.TypeString,
					Description: "Notary key ID",
					Required:    true,
				},
				fieldNameMacSigningNotaryKey: {
					Type:        framework.TypeString,
					Description: "Notary key ID",
					Required:    true,
				},
				fieldNameMacSigningNotaryIssuer: {
					Type:        framework.TypeString,
					Description: "Notary issuer",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Add or update mac signing credentials",
					Callback:    pathMacSigningCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Add or update mac signing credentials",
					Callback:    pathMacSigningCreateOrUpdate,
				},
			},
		},
		{
			Pattern:         "configure/build/mac_signing/" + framework.GenericNameRegex(fieldNameMacSigningName) + "$",
			HelpSynopsis:    "Delete a build signing credentials",
			HelpDescription: "Delete a build signing credentials by name",
			Fields: map[string]*framework.FieldSchema{
				fieldNameMacSigningName: {
					Type:        framework.TypeNameString,
					Description: "Credentials name",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Delete mac signing credentials",
					Callback:    pathMacSigningDelete,
				},
			},
		},
	}
}

func pathMacSigningCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	creds := MacSigningCredentials{
		Name:         fields.Get(fieldNameMacSigningName).(string),
		Certificate:  fields.Get(fieldNameMacSigningCertificate).(string),
		Password:     fields.Get(fieldNameMacSigningPassword).(string),
		NotaryKeyID:  fields.Get(fieldNameMacSigningNotaryKeyID).(string),
		NotaryKey:    fields.Get(fieldNameMacSigningNotaryKey).(string),
		NotaryIssuer: fields.Get(fieldNameMacSigningNotaryIssuer).(string),
	}

	if err := PutCredentials(ctx, req, creds); err != nil {
		return nil, fmt.Errorf("failed to put credentials: %w", err)
	}

	return nil, nil
}

func pathMacSigningDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	name := fields.Get(fieldNameMacSigningName).(string)
	if err := DeleteCredentials(ctx, req, name); err != nil {
		return nil, fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil, nil
}
