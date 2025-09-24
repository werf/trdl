package docker

import (
	"fmt"
	"os"

	"github.com/werf/trdl/server/pkg/mac_signing"
)

func GetMacSigningCommandMounts(creds *mac_signing.Credentials) []string {
	args := make([]string, 0, 5)

	if creds != nil {
		args = append(args, "--secret", fmt.Sprintf("id=%s_cert", creds.Name))
		if creds.Password != "" {
			args = append(args, "--secret", fmt.Sprintf("id=%s_password", creds.Name))
		}
		args = append(args, "--secret", fmt.Sprintf("id=%s_notary_key_id", creds.Name))
		args = append(args, "--secret", fmt.Sprintf("id=%s_notary_key", creds.Name))
		args = append(args, "--secret", fmt.Sprintf("id=%s_notary_issuer", creds.Name))
	}
	return args
}

func SetMacSigningTempEnvVars(creds *mac_signing.Credentials) error {
	if creds == nil {
		return nil
	}

	if err := os.Setenv(creds.Name+"_cert", creds.Certificate); err != nil {
		return fmt.Errorf("unable to set certificate env var: %w", err)
	}

	if creds.Password != "" {
		if err := os.Setenv(creds.Name+"_password", creds.Password); err != nil {
			return fmt.Errorf("unable to set password env var: %w", err)
		}
	}

	if err := os.Setenv(creds.Name+"_notary_key_id", creds.NotaryKeyID); err != nil {
		return fmt.Errorf("unable to set notary key id env var: %w", err)
	}

	if err := os.Setenv(creds.Name+"_notary_key", creds.NotaryKey); err != nil {
		return fmt.Errorf("unable to set notary key env var: %w", err)
	}

	if err := os.Setenv(creds.Name+"_notary_issuer", creds.NotaryIssuer); err != nil {
		return fmt.Errorf("unable to set notary issuer env var: %w", err)
	}

	return nil
}
