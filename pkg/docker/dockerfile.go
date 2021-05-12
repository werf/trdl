package docker

import (
	"archive/tar"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
)

const (
	containerSourceDir    = "/git"
	containerArtifactsDir = "/result"
)

var (
	ImageNameWithoutRequiredDigestError = errors.New("the image name must contain an digest \"REPO[:TAG]@DIGEST\" (e.g. \"ubuntu:18.04@sha256:538529c9d229fb55f50e6746b119e899775205d62c0fc1b7e679b30d02ecb6e8\")")
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

func ValidateImageNameWithDigest(imageName string) error {
	if !reference.ReferenceRegexp.MatchString(imageName) {
		return ImageNameWithoutRequiredDigestError
	}

	res := reference.ReferenceRegexp.FindStringSubmatch(imageName)

	// res[0] full match
	// res[1] repository
	// res[2] tag
	// res[3] digest
	if len(res) != 4 {
		panic(fmt.Sprintf("unexpected regexp find submatch result %v (%d)", res, len(res)))
	} else if res[3] == "" {
		return ImageNameWithoutRequiredDigestError
	}

	return nil
}
