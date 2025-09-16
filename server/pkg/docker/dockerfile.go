package docker

import (
	"archive/tar"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/werf/trdl/server/pkg/certificates"
	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	ContainerSourceDir    = "git"
	ContainerArtifactsDir = "result"
	quill_image           = "registry.werf.io/trdl/quill:028f446b1b76be918781b24e7f77a6b4c0c74972"
)

type DockerfileOpts struct {
	EnvVars      map[string]string
	Labels       map[string]string
	Secrets      []secrets.Secret
	Certificates []certificates.Certificate
}

func GenerateAndAddDockerfileToTar(tw *tar.Writer, dockerfileTarPath, fromImage string, runCommands []string, dockerfileOpts DockerfileOpts) error {
	dockerfileData := generateDockerfile(fromImage, runCommands, dockerfileOpts)
	header := &tar.Header{
		Format:     tar.FormatGNU,
		Name:       dockerfileTarPath,
		Size:       int64(len(dockerfileData)),
		Mode:       int64(os.ModePerm),
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("unable to write tar entry %q header: %w", dockerfileTarPath, err)
	}

	if _, err := tw.Write(dockerfileData); err != nil {
		return fmt.Errorf("unable to write tar entry %q data: %w", dockerfileTarPath, err)
	}

	return nil
}

func generateDockerfile(fromImage string, runCommands []string, opts DockerfileOpts) []byte {
	var data []byte
	addLineFunc := func(line string) {
		data = append(data, []byte(line+"\n")...)
	}

	// we use stages to reduce the size of output data to stdout
	addLineFunc(fmt.Sprintf("FROM %s AS builder", fromImage))

	for labelName, labelVal := range opts.Labels {
		addLineFunc(fmt.Sprintf("LABEL %s=%q", labelName, labelVal))
	}

	for envVarName, envVarVal := range opts.EnvVars {
		addLineFunc(fmt.Sprintf("ENV %s=%q", envVarName, envVarVal))
	}

	// copy source code and set workdir for the following docker instructions
	addLineFunc(fmt.Sprintf("COPY . /%s", ContainerSourceDir))
	addLineFunc(fmt.Sprintf("WORKDIR /%s", ContainerSourceDir))

	addLineFunc(fmt.Sprintf("RUN %s", fmt.Sprintf("mkdir -p /%s", ContainerArtifactsDir)))

	// run user's build commands
	if len(runCommands) != 0 {
		if len(opts.Secrets) > 0 {
			mounts := GetSecretsRunMounts(opts.Secrets)
			addLineFunc(fmt.Sprintf("%s %s", mounts, strings.Join(runCommands, " && ")))
		} else {
			addLineFunc(fmt.Sprintf("RUN %s", strings.Join(runCommands, " && ")))
		}
	}

	if len(opts.Certificates) > 0 {
		addLineFunc(fmt.Sprintf("FROM %s AS signer", quill_image))
		addLineFunc(fmt.Sprintf("COPY --from=builder /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))
		for _, cert := range opts.Certificates {
			addLineFunc(fmt.Sprintf(`RUN --mount=type=secret,id=%s --mount=type=secret,id=%s \
	export QUILL_SIGN_P12="$(cat /run/secrets/%s)" && \
	export QUILL_SIGN_PASSWORD="$(cat /run/secrets/%s)" && \
	find /%s -type f | while read f; do \
	  if file "$f" | grep -q "Mach-O"; then \
		echo "Signing $f" && \
		quill sign "$f"; \
	  fi; \
	done`,
				cert.Name,
				cert.Name+"_password",
				cert.Name,
				cert.Name+"_password",
				ContainerArtifactsDir,
			))
		}

		addLineFunc("FROM scratch")
		addLineFunc(fmt.Sprintf("COPY --from=signer /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))
	} else {
		addLineFunc("FROM scratch")
		addLineFunc(fmt.Sprintf("COPY --from=builder /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))
	}

	return data
}
