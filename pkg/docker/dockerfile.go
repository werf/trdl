package docker

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
)

type DockerfileOpts struct {
	ContainerSourceDir    string
	ContainerArtifactsDir string
	WithArtifacts         bool
	EnvVars               map[string]string
	Labels                map[string]string
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
		return fmt.Errorf("unable to write tar entry %q header: %s", dockerfileTarPath, err)
	}

	if _, err := tw.Write(dockerfileData); err != nil {
		return fmt.Errorf("unable to write tar entry %q data: %s", dockerfileTarPath, err)
	}

	return nil
}

func generateDockerfile(fromImage string, runCommands []string, opts DockerfileOpts) []byte {
	if opts.ContainerSourceDir == "" {
		opts.ContainerSourceDir = "/git"
	}

	if opts.ContainerArtifactsDir == "" {
		opts.ContainerArtifactsDir = "/result"
	}

	var data []byte
	addLineFunc := func(line string) {
		data = append(data, []byte(line+"\n")...)
	}

	addLineFunc(fmt.Sprintf("FROM %s", fromImage))

	for labelName, labelVal := range opts.Labels {
		addLineFunc(fmt.Sprintf("LABEL %s=%q", labelName, labelVal))
	}

	for envVarName, envVarVal := range opts.EnvVars {
		addLineFunc(fmt.Sprintf("ENV %s=%q", envVarName, envVarVal))
	}

	// copy source code and set workdir for the following docker instructions
	addLineFunc(fmt.Sprintf("COPY . %s", opts.ContainerSourceDir))
	addLineFunc(fmt.Sprintf("WORKDIR %s", opts.ContainerSourceDir))

	if opts.WithArtifacts {
		// create empty dir for release artifacts
		addLineFunc(fmt.Sprintf("RUN %s", fmt.Sprintf("mkdir -p %s", opts.ContainerArtifactsDir)))
	}

	// run user's build commands
	for _, command := range runCommands {
		addLineFunc(fmt.Sprintf("RUN %s", command))
	}

	if opts.WithArtifacts {
		// tar result files to stdout (with control messages for a receiver)
		serviceRunCommands := []string{
			fmt.Sprintf("echo -n $(echo -n '%s' | base64 -d)", base64.StdEncoding.EncodeToString(artifactsTarStartReadCode)),
			fmt.Sprintf("tar c -C %s . | base64", opts.ContainerArtifactsDir),
			fmt.Sprintf("echo -n $(echo -n '%s' | base64 -d)", base64.StdEncoding.EncodeToString(artifactsTarStopReadCode)),
		}
		addLineFunc("RUN " + strings.Join(serviceRunCommands, " && "))
	}

	return data
}
