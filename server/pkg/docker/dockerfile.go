package docker

import (
	"archive/tar"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	ContainerSourceDir    = "git"
	ContainerArtifactsDir = "result"
)

type DockerfileOpts struct {
	EnvVars map[string]string
	Labels  map[string]string
	Secrets []secrets.Secret
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
	addLineFunc(fmt.Sprintf("FROM %s as builder", fromImage))

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

	// since we need only the artifacts from the build stage
	// we use scratch as the final image containing ONLY artifacts
	addLineFunc("FROM scratch")
	addLineFunc(fmt.Sprintf("COPY --from=builder /%s /%s/", ContainerArtifactsDir, ContainerArtifactsDir))

	return data
}
