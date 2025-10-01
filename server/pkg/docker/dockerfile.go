package docker

import (
	"archive/tar"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	ContainerSourceDir    = "git"
	ContainerArtifactsDir = "result"
	DefaultQuillImage     = "registry.werf.io/trdl/quill:028f446b1b76be918781b24e7f77a6b4c0c74972"
)

type DockerfileOpts struct {
	EnvVars               map[string]string
	Labels                map[string]string
	Secrets               []secrets.Secret
	MacSigningCredentials *mac_signing.Credentials
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

	if opts.MacSigningCredentials != nil {
		creds := opts.MacSigningCredentials
		quillImage := GetQuillImage()
		addLineFunc(fmt.Sprintf("FROM %s AS signer", quillImage))
		addLineFunc(fmt.Sprintf("COPY --from=builder /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))

		addLineFunc(fmt.Sprintf(
			`RUN --mount=type=secret,id=%[1]s_cert \
	--mount=type=secret,id=%[1]s_password \
	--mount=type=secret,id=%[1]s_notary_key_id \
	--mount=type=secret,id=%[1]s_notary_key \
	--mount=type=secret,id=%[1]s_notary_issuer \
	export QUILL_SIGN_P12="$(cat /run/secrets/%[1]s_cert)" && \
	export QUILL_SIGN_PASSWORD="$(cat /run/secrets/%[1]s_password)" && \
	export QUILL_NOTARY_KEY_ID="$(cat /run/secrets/%[1]s_notary_key_id)" && \
	export QUILL_NOTARY_KEY="$(cat /run/secrets/%[1]s_notary_key)" && \
	export QUILL_NOTARY_ISSUER="$(cat /run/secrets/%[1]s_notary_issuer)" && \
	find /%[2]s -type f | while read f; do \
	  if file "$f" | grep -q "Mach-O"; then \
		echo "Signing $f" && \
		quill sign-and-notarize "$f"; \
	  fi; \
	done`,
			creds.Name,
			ContainerArtifactsDir,
		))

		addLineFunc("FROM scratch")
		addLineFunc(fmt.Sprintf("COPY --from=signer /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))
	} else {
		addLineFunc("FROM scratch")
		addLineFunc(fmt.Sprintf("COPY --from=builder /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))
	}

	return data
}

func GetQuillImage() string {
	if image := os.Getenv("TRDL_QUILL_IMAGE"); image != "" {
		return image
	}
	return DefaultQuillImage
}
