package trdl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
)

type Configuration struct {
	GitRepoUrl                                 string `json:"git_repo_url"`
	GitTrdlChannelsBranch                      string `json:"git_trdl_channels_branch"`
	RequiredNumberOfVerifiedSignaturesOnCommit int    `json:"required_number_of_verified_signatures_on_commit"`
	GitCredential                              struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	TrustedPGPPublicKeys []string

	S3Endpoint        string `json:"s3_endpoint"`
	S3Region          string `json:"s3_region"`
	S3AccessKeyID     string `json:"s3_access_key_id"`
	S3SecretAccessKey string `json:"s3_secret_access_key"`
	S3BucketName      string `json:"s3_bucket_name"`
}

func GetAndValidateConfiguration(ctx context.Context, storage logical.Storage) (*Configuration, *logical.Response, error) {
	var c Configuration

	// parse and validate base configuration part
	{
		e, err := storage.Get(ctx, storageKeyConfigurationBase)
		if err != nil {
			return nil, nil, err
		}

		if e == nil {
			return nil, logical.ErrorResponse("configuration not found"), nil
		}

		if err := json.Unmarshal(e.Value, &c); err != nil {
			return nil, nil, fmt.Errorf("unable to unmarshal configuration json:\n%s\n---\n%s", e.Value, err)
		}

		if c.GitRepoUrl == "" {
			return nil, logical.ErrorResponse("required configuration field %q must be set", fieldNameGitRepoUrl), nil
		}

		// TODO
		//if c.RequiredNumberOfVerifiedSignaturesOnCommit <= 0 {
		//	return nil, logical.ErrorResponse("required configuration field %q must be set and be more than 0", fieldNameRequiredNumberOfVerifiedSignaturesOnCommit), nil
		//}
	}

	// parse git credential
	{
		e, err := storage.Get(ctx, storageKeyConfigurationGitCredential)
		if err != nil {
			return nil, nil, err
		}

		if e != nil {
			if err := json.Unmarshal(e.Value, &c.GitCredential); err != nil {
				return nil, nil, fmt.Errorf("unable to unmarshal configuration git credential: %s", err)
			}
		}
	}

	{
		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, storage)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get trusted public keys: %s", err)
		}

		if len(trustedPGPPublicKeys) < c.RequiredNumberOfVerifiedSignaturesOnCommit {
			return nil, logical.ErrorResponse("not enough trusted PGP public keys defined (%d): must be equal or more than %q (%d)", len(trustedPGPPublicKeys), fieldNameRequiredNumberOfVerifiedSignaturesOnCommit, c.RequiredNumberOfVerifiedSignaturesOnCommit), nil
		}

		c.TrustedPGPPublicKeys = trustedPGPPublicKeys
	}

	return &c, nil, nil
}
