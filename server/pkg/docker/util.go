package docker

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

	list, err := cli.ImageList(ctx, types.ImageListOptions{Filters: filterSet})
	if err != nil {
		return fmt.Errorf("unable to list images: %s", err)
	}

	for _, img := range list {
		options := types.ImageRemoveOptions{PruneChildren: true, Force: true}
		if _, err := cli.ImageRemove(ctx, img.ID, options); err != nil {
			return fmt.Errorf("unable to remove image %q: %s", img.ID, err)
		}
	}

	return nil
}
