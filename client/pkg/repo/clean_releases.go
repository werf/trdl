package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/werf/trdl/client/pkg/util"
)

var releaseMetafileExpirationPeriod = time.Hour * 24

func (c Client) CleanReleases() error {
	actualLocalReleases, err := c.getActualLocalReleases()
	if err != nil {
		return fmt.Errorf("unable to get actual local releases: %w", err)
	}

	allReleasesGlob := filepath.Join(c.dir, releasesDir, "*")
	releaseDirList, err := filepath.Glob(allReleasesGlob)
	if err != nil {
		return fmt.Errorf("unable to glob files: %w", err)
	}

	for _, releaseDir := range releaseDirList {
		_, releaseName := filepath.Split(releaseDir)

		// skip actual channel release
		if actualLocalReleases[releaseName] {
			continue
		}

		// skip recently used release
		{
			metafile := c.releaseMetafile(releaseName)
			isRecentlyUsed, err := metafile.HasBeenModifiedWithinPeriod(c.locker, releaseMetafileExpirationPeriod)
			if err != nil {
				return err
			}

			if isRecentlyUsed {
				continue
			}

			if err := metafile.Delete(c.locker); err != nil {
				return fmt.Errorf("unable to remove release %q metafile: %w", releaseName, err)
			}
		}

		if err := os.RemoveAll(releaseDir); err != nil {
			return fmt.Errorf("unable to remove %q: %w", releaseDir, err)
		}
	}

	return nil
}

func (c Client) getActualLocalReleases() (map[string]bool, error) {
	actualLocalReleases := map[string]bool{}

	allChannelsGlob := filepath.Join(c.dir, channelsDir, "*", "*")
	filePathList, err := filepath.Glob(allChannelsGlob)
	if err != nil {
		return nil, fmt.Errorf("unable to glob files: %w", err)
	}

	for _, filePath := range filePathList {
		exist, err := util.IsRegularFileExist(filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to check existence of file %q: %w", filePath, err)
		}

		if !exist {
			continue
		}

		release, err := readChannelRelease(filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to get channel release: %w", err)
		}

		actualLocalReleases[release] = true
	}

	return actualLocalReleases, nil
}
