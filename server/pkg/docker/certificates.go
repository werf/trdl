package docker

import (
	"fmt"
	"os"

	"github.com/werf/trdl/server/pkg/certificates"
)

func GetCertificatesCommandMounts(certificates []certificates.Certificate) []string {
	args := make([]string, 0, len(certificates)*2)
	for _, cert := range certificates {
		args = append(args, "--secret", fmt.Sprintf("id=%s", cert.Name))
		if cert.Password != "" {
			args = append(args, "--secret", fmt.Sprintf("id=%s_password", cert.Name))
		}
	}
	return args
}

func SetCertificatesTempEnvVars(certificates []certificates.Certificate) error {
	for _, cert := range certificates {
		err := os.Setenv(cert.Name, string(cert.Data))
		if err != nil {
			return fmt.Errorf("unable to use certificate data")
		}
		if cert.Password != "" {
			err = os.Setenv(cert.Name+"_password", cert.Password)
			if err != nil {
				return fmt.Errorf("unable to use certificate password")
			}
		}
	}
	return nil
}
