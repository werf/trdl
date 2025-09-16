package certificates

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameCertificateName     = "name"
	fieldNameCertificateData     = "data"
	fieldNameCertificatePassword = "password"
)

type Certificate struct {
	Name     string
	Data     string
	Password string
}

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/build/certificates/?",
			HelpSynopsis:    "Add a build certificate",
			HelpDescription: "Add a build certificate",
			Fields: map[string]*framework.FieldSchema{
				fieldNameCertificateName: {
					Type:        framework.TypeNameString,
					Description: "Certificate name",
					Required:    true,
				},
				fieldNameCertificateData: {
					Type:        framework.TypeString,
					Description: "Certificate data base64 encoded",
					Required:    true,
				},
				fieldNameCertificatePassword: {
					Type:        framework.TypeString,
					Description: "Certificate password",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Add a build certificate",
					Callback:    pathCertificateCreate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Add a build certificate",
					Callback:    pathCertificateCreate,
				},
			},
		},
		{
			Pattern:         "configure/build/certificate/" + framework.GenericNameRegex(fieldNameCertificateName) + "$",
			HelpSynopsis:    "Delete a build certificate",
			HelpDescription: "Delete a build certificate",
			Fields: map[string]*framework.FieldSchema{
				fieldNameCertificateName: {
					Type:        framework.TypeNameString,
					Description: "Certificate name",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Delete a build certificate",
					Callback:    pathCertificateDelete,
				},
			},
		},
	}
}

func pathCertificateCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}
	return PutCertificate(ctx, req, certificateStorage{
		Name:     fields.Get(fieldNameCertificateName).(string),
		Data:     fields.Get(fieldNameCertificateData).(string),
		Password: fields.Get(fieldNameCertificatePassword).(string),
	})
}

func pathCertificateDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}
	err := DeleteCertificate(ctx, req, certificateStorage{
		Name: fields.Get(fieldNameCertificateName).(string),
	})
	if err != nil {
		return nil, fmt.Errorf("error delete certificate: %w", err)
	}
	return nil, nil
}
