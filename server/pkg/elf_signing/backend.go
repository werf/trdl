package elf_signing

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameELFSigningKey               = "key"
	fieldNameELFSigningKeyPass           = "password"
	fieldNameELFSigningCertificate       = "certificate"
	fieldNameELFSigningIntermediates     = "intermediates"
	fieldNameELFSigningVaultAddr         = "vault_addr"
	fieldNameELFSigningVaultTransitPath  = "vault_transit_path"
	fieldNameELFSigningVaultAuthPath     = "vault_auth_path"
	fieldNameELFSigningVaultAuthRoleID   = "vault_auth_role_id"
	fieldNameELFSigningVaultAuthSecretID = "vault_auth_secret_id"
)

func Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern:         "configure/delivery_kit_elf_signing",
			HelpSynopsis:    "Add or update Delivery Kit elf signing settings",
			HelpDescription: "Add or update Delivery Kit elf signing settings",
			Fields: map[string]*framework.FieldSchema{
				fieldNameELFSigningKey: {
					Type:        framework.TypeString,
					Description: "Private key data base64 encoded or Vault url",
					Required:    true,
				},
				fieldNameELFSigningKeyPass: {
					Type:        framework.TypeString,
					Description: "Private key password",
				},
				fieldNameELFSigningCertificate: {
					Type:        framework.TypeString,
					Description: "Certificate data base64 encoded",
					Required:    true,
				},
				fieldNameELFSigningIntermediates: {
					Type:        framework.TypeString,
					Description: "Intermediates certificates data base64 encoded",
				},
				fieldNameELFSigningVaultAddr: {
					Type:        framework.TypeString,
					Description: "Vault server address",
				},
				fieldNameELFSigningVaultTransitPath: {
					Type:        framework.TypeString,
					Description: "Vault transit path",
				},
				fieldNameELFSigningVaultAuthPath: {
					Type:        framework.TypeString,
					Description: "Vault auth path",
				},
				fieldNameELFSigningVaultAuthRoleID: {
					Type:        framework.TypeString,
					Description: "Vault auth role id",
				},
				fieldNameELFSigningVaultAuthSecretID: {
					Type:        framework.TypeString,
					Description: "Vault auth secret id",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Add or update ELF signing settings",
					Callback:    pathELFSigningCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Add or update ELF signing settings",
					Callback:    pathELFSigningCreateOrUpdate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Delete ELF signing settings",
					Callback:    pathELFSigningDelete,
				},
			},
		},
	}
}

func pathELFSigningCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	settings := SignerSettings{
		KeyRef:           fields.Get(fieldNameELFSigningKey).(string),
		KeyPassword:      fields.Get(fieldNameELFSigningKeyPass).(string),
		CertRef:          fields.Get(fieldNameELFSigningCertificate).(string),
		IntermediatesRef: fields.Get(fieldNameELFSigningIntermediates).(string),
		VaultOpts: VaultSignerOpts{
			Address:      fields.Get(fieldNameELFSigningVaultAddr).(string),
			TransitPath:  fields.Get(fieldNameELFSigningVaultTransitPath).(string),
			AuthPath:     fields.Get(fieldNameELFSigningVaultAuthPath).(string),
			AuthRoleID:   fields.Get(fieldNameELFSigningVaultAuthRoleID).(string),
			AuthSecretID: fields.Get(fieldNameELFSigningVaultAuthSecretID).(string),
		},
	}

	if err := PutSettings(ctx, req, settings); err != nil {
		return nil, fmt.Errorf("failed to put elf signing settings: %w", err)
	}

	return nil, nil
}

func pathELFSigningDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := DeleteSettings(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to delete elf signing settings: %w", err)
	}

	return nil, nil
}
