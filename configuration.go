package trdl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

type Configuration struct {
	GitRepoUrl                                 string `json:"git_repo_url"`
	RequiredNumberOfVerifiedSignaturesOnCommit int    `json:"required_number_of_verified_signatures_on_commit"`
	GitCredential                              struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	TrustedGPGPublicKeys []string
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
			return nil, nil, fmt.Errorf("unable to unmarshal configuration: %s", err)
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

	// parse gpg public keys
	{
		list, err := storage.List(ctx, storageKeyPrefixTrustedGPGPublicKey)
		if err != nil {
			return nil, nil, err
		}

		for _, postfix := range list {
			storageEntryKey := storageKeyPrefixTrustedGPGPublicKey + postfix
			e, err := storage.Get(ctx, storageEntryKey)
			if err != nil {
				return nil, nil, err
			}

			c.TrustedGPGPublicKeys = append(c.TrustedGPGPublicKeys, string(e.Value))
		}

		if len(c.TrustedGPGPublicKeys) < c.RequiredNumberOfVerifiedSignaturesOnCommit {
			return nil, logical.ErrorResponse("not enough trusted GPG public keys defined (%d): must be equal or more than %q (%d)", len(c.TrustedGPGPublicKeys), fieldNameRequiredNumberOfVerifiedSignaturesOnCommit, c.RequiredNumberOfVerifiedSignaturesOnCommit), nil
		}
	}

	return &c, nil, nil
}
