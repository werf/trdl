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
			HelpSynopsis:    "Configure ELF binary signing via Delivery Kit",
			HelpDescription: "Configure ELF binary signing via Delivery Kit. Signing buffers each ELF artifact in memory up to 512 MiB.",
			Fields: map[string]*framework.FieldSchema{
				fieldNameELFSigningKey: {
					Type:        framework.TypeString,
					Description: "Private key data base64 encoded or a Vault key reference in the form hashivault://<key>. When a hashivault:// reference is used, configure the vault_* parameters",
					Required:    true,
				},
				fieldNameELFSigningKeyPass: {
					Type:        framework.TypeString,
					Description: "Private key password. Must not be set when key is a hashivault:// reference",
				},
				fieldNameELFSigningCertificate: {
					Type:        framework.TypeString,
					Description: "Certificate data base64 encoded",
					Required:    true,
				},
				fieldNameELFSigningIntermediates: {
					Type:        framework.TypeString,
					Description: "Certificate chain (intermediates and root) base64 encoded, as a single PEM bundle",
				},
				fieldNameELFSigningVaultAddr: {
					Type:        framework.TypeString,
					Description: "Vault server address. Applies only when key is a hashivault:// reference",
				},
				fieldNameELFSigningVaultTransitPath: {
					Type:        framework.TypeString,
					Description: "Mount path of Vault transit engine. Applies only when key is a hashivault:// reference",
				},
				fieldNameELFSigningVaultAuthPath: {
					Type:        framework.TypeString,
					Description: "Mount path of Vault auth method. Applies only when key is a hashivault:// reference",
					Default:     "ar",
				},
				fieldNameELFSigningVaultAuthRoleID: {
					Type:        framework.TypeString,
					Description: "AppRole RoleID used to authenticate to Vault. Applies only when key is a hashivault:// reference",
				},
				fieldNameELFSigningVaultAuthSecretID: {
					Type:        framework.TypeString,
					Description: "AppRole SecretID used to authenticate to Vault. Applies only when key is a hashivault:// reference",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Configure ELF signing",
					Callback:    pathELFSigningCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Configure ELF signing",
					Callback:    pathELFSigningCreateOrUpdate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Description: "Reset ELF signing",
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
