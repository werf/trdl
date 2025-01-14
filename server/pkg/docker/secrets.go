package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/werf/trdl/server/pkg/secrets"
)

func GetSecretsRunMounts(secrets []secrets.Secret) string {
	mounts := buildStringInstruction(secrets)
	return fmt.Sprintf("RUN %s", mounts)
}

func GetSecretsCommandMounts(secrets []secrets.Secret) []string {
	args := make([]string, 0, len(secrets))
	for _, s := range secrets {
		args = append(args, "--secret", fmt.Sprintf("id=%s", s.Id))
	}
	return args
}

func SetTempEnvVars(secrets []secrets.Secret) error {
	for _, s := range secrets {
		err := os.Setenv(s.Id, string(s.Data))
		if err != nil {
			return fmt.Errorf("unable to use secret data")
		}
	}
	return nil
}

func buildStringInstruction(secrets []secrets.Secret) string {
	var builder strings.Builder
	for _, secret := range secrets {
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(fmt.Sprintf("--mount=type=secret,id=%s", secret.Id))
	}

	return builder.String()
}
