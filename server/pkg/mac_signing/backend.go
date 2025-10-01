package mac_signing

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameMacSigningCertificateData = "data"
	fieldNameMacSigningPassword        = "password"
	fieldNameMacSigningNotaryKeyID     = "notary_key_id"
	fieldNameMacSigningNotaryKey       = "notary_key"
	fieldNameMacSigningNotaryIssuer    = "notary_issuer"
)

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/build/mac_signing_identity",
			HelpSynopsis:    "Add or update build signing credentials",
			HelpDescription: "Add or update build signing credentials for macOS builds",
			Fields: map[string]*framework.FieldSchema{
				fieldNameMacSigningCertificateData: {
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
					Description: "Notary key",
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

	creds := Credentials{
		Name:         macSigningCertificateName,
		Certificate:  fields.Get(fieldNameMacSigningCertificateData).(string),
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

func pathMacSigningDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := DeleteCredentials(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil, nil
}
