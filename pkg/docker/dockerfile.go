package docker

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	containerSourceDir    = "/git"
	containerArtifactsDir = "/result"
)

func GenerateAndAddDockerfileToTar(tw *tar.Writer, dockerfileTarPath, serviceDirInContext string, fromImage string, runCommands []string, withArtifacts bool) error {
	dockerfileData := generateDockerfile(fromImage, runCommands, serviceDirInContext, withArtifacts)
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

func generateDockerfile(fromImage string, runCommands []string, serviceDirInContext string, withArtifacts bool) []byte {
	var data []byte
	addLineFunc := func(line string) {
		data = append(data, []byte(line+"\n")...)
	}

	addLineFunc(fmt.Sprintf("FROM %s", fromImage))

	// copy source code and set workdir for the following docker instructions
	addLineFunc(fmt.Sprintf("COPY . %s", containerSourceDir))
	addLineFunc(fmt.Sprintf("WORKDIR %s", containerSourceDir))

	// remove service data from user's context
	addLineFunc(fmt.Sprintf("RUN %s", fmt.Sprintf("rm -rf %s", serviceDirInContext)))

	if withArtifacts {
		// create empty dir for release artifacts
		addLineFunc(fmt.Sprintf("RUN %s", fmt.Sprintf("mkdir -p %s", containerArtifactsDir)))
	}

	// run user's build commands
	for _, command := range runCommands {
		addLineFunc(fmt.Sprintf("RUN %s", command))
	}

	if withArtifacts {
		// tar result files to stdout (with control messages for a receiver)
		serviceRunCommands := []string{
			fmt.Sprintf("echo -n $(echo -n '%s' | base64 -d)", base64.StdEncoding.EncodeToString(artifactsTarStartReadCode)),
			fmt.Sprintf("tar c -C %s . | base64", containerArtifactsDir),
			fmt.Sprintf("echo -n $(echo -n '%s' | base64 -d)", base64.StdEncoding.EncodeToString(artifactsTarStopReadCode)),
		}
		addLineFunc("RUN " + strings.Join(serviceRunCommands, " && "))
	}

	return data
}
