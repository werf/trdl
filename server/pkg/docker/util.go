package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

var ErrImageNameWithoutRequiredDigest = errors.New("the image name must contain an digest \"REPO[:TAG]@DIGEST\" (e.g. \"ubuntu:18.04@sha256:538529c9d229fb55f50e6746b119e899775205d62c0fc1b7e679b30d02ecb6e8\")")

func ValidateImageNameWithDigest(imageName string) error {
	if !reference.ReferenceRegexp.MatchString(imageName) {
		return ErrImageNameWithoutRequiredDigest
	}

	res := reference.ReferenceRegexp.FindStringSubmatch(imageName)

	// res[0] full match
	// res[1] repository
	// res[2] tag
	// res[3] digest
	if len(res) != 4 {
		panic(fmt.Sprintf("unexpected regexp find submatch result %v (%d)", res, len(res)))
	} else if res[3] == "" {
		return ErrImageNameWithoutRequiredDigest
	}

	return nil
}

func RemoveImagesByLabels(ctx context.Context, cli *client.Client, labels map[string]string) error {
	filterSet := filters.NewArgs()
	for key, value := range labels {
		filterSet.Add("label", fmt.Sprintf("%s=%s", key, value))
	}

	list, err := cli.ImageList(ctx, image.ListOptions{Filters: filterSet})
	if err != nil {
		return fmt.Errorf("unable to list images: %w", err)
	}

	for _, img := range list {
		options := image.RemoveOptions{PruneChildren: true, Force: true}
		if _, err := cli.ImageRemove(ctx, img.ID, options); err != nil {
			return fmt.Errorf("unable to remove image %q: %w", img.ID, err)
		}
	}

	return nil
}

func getRecommendation(line string) string {
	switch {
	case strings.Contains(line, "unable to decode p12 file"):
		return "P12 certificate corrupted - verify with: base64 --decode cert.b64 > cert.p12 && openssl pkcs12 -info -in cert.p12"

	case strings.Contains(line, "unable to parse EC private key"):
		return "Invalid EC private key format - ensure PEM encoding starts with '-----BEGIN PRIVATE KEY-----'"

	case strings.Contains(line, "401 Unauthorized"):
		return "Invalid Apple Notary credentials - check API key in App Store Connect"

	default:
		return ""
	}
}
