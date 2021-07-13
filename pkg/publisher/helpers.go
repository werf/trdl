package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Masterminds/semver"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
)

type InMemoryFile struct {
	Name string
	Data []byte
}

func NewErrIncorrectTargetPath(path string) error {
	return fmt.Errorf(`got incorrect target path %q: expected path in format <os>-<arch>/... where os can be either "any", "linux", "darwin" or "windows", and arch can be either "any", "amd64" or "arm64"`, path)
}

func NewErrIncorrectChannelName(chnl string) error {
	return fmt.Errorf(`got incorrect channel name %q: expected "alpha", "beta", "ea", "stable" or "rock-solid"`, chnl)
}

func PublishReleaseTarget(ctx context.Context, repository *S3Repository, releaseName, path string, data io.Reader) error {
	_, err := semver.NewVersion(releaseName)
	if err != nil {
		return fmt.Errorf("expected semver release name got %q: %s", releaseName, err)
	}

	pathParts := SplitFilepath(filepath.Clean(path))
	if len(pathParts) == 0 {
		return NewErrIncorrectTargetPath(path)
	}

	osAndArchParts := strings.SplitN(pathParts[0], "-", 2)

	switch osAndArchParts[0] {
	case "any", "linux", "darwin", "windows":
	default:
		return NewErrIncorrectTargetPath(path)
	}

	switch osAndArchParts[1] {
	case "any", "amd64", "arm64":
	default:
		return NewErrIncorrectTargetPath(path)
	}

	return repository.PublishTarget(ctx, filepath.Join("releases", releaseName, path), data)
}

func PublishChannelsConfig(ctx context.Context, repository *S3Repository, trdlChannelsConfig *config.TrdlChannels) error {
	// validate
	for _, grp := range trdlChannelsConfig.Groups {
		if _, err := semver.NewVersion(grp.Name); err != nil {
			return fmt.Errorf("expected semver group got %q: %s", grp.Name, err)
		}

		for _, chnl := range grp.Channels {
			switch chnl.Name {
			case "alpha", "beta", "ea", "stable", "rock-solid":
			default:
				return NewErrIncorrectChannelName(chnl.Name)
			}

			if _, err := semver.NewVersion(chnl.Version); err != nil {
				return fmt.Errorf("expected semver version map for group %q channel %q, got %q: %s", grp.Name, chnl.Name, chnl.Version, err)
			}
		}
	}

	// publish /channels/GROUP/CHANNEL -> VERSION
	for _, grp := range trdlChannelsConfig.Groups {
		for _, chnl := range grp.Channels {
			publishPath := filepath.Join("channels", grp.Name, chnl.Name)

			if err := repository.PublishTarget(ctx, publishPath, bytes.NewBuffer([]byte(chnl.Version+"\n"))); err != nil {
				return fmt.Errorf("error publishing %q: %s", publishPath, err)
			}
		}
	}

	return nil
}

func PublishInMemoryFiles(ctx context.Context, repository *S3Repository, files []*InMemoryFile) error {
	for _, file := range files {
		if err := repository.PublishTarget(ctx, file.Name, bytes.NewReader(file.Data)); err != nil {
			return fmt.Errorf("error publishing %q: %s", file.Name, err)
		}
	}

	return nil
}

// TODO: move this to the separate project in github.com/werf
func SplitFilepath(path string) (result []string) {
	path = filepath.FromSlash(path)
	separator := os.PathSeparator

	idx := 0
	if separator == '\\' {
		// if the separator is '\\', then we can just split...
		result = strings.Split(path, string(separator))
		idx = len(result)
	} else {
		// otherwise, we need to be careful of situations where the separator was escaped
		cnt := strings.Count(path, string(separator))
		if cnt == 0 {
			return []string{path}
		}

		result = make([]string, cnt+1)
		pathlen := len(path)
		separatorLen := utf8.RuneLen(separator)
		emptyEnd := false
		for start := 0; start < pathlen; {
			end := indexRuneWithEscaping(path[start:], separator)
			if end == -1 {
				emptyEnd = false
				end = pathlen
			} else {
				emptyEnd = true
				end += start
			}
			result[idx] = path[start:end]
			start = end + separatorLen
			idx++
		}

		// If the last rune is a path separator, we need to append an empty string to
		// represent the last, empty path component. By default, the strings from
		// make([]string, ...) will be empty, so we just need to increment the count
		if emptyEnd {
			idx++
		}
	}

	return result[:idx]
}

// TODO: move this to the separate project in github.com/werf
// Find the first index of a rune in a string,
// ignoring any times the rune is escaped using "\".
func indexRuneWithEscaping(s string, r rune) int {
	end := strings.IndexRune(s, r)
	if end == -1 {
		return -1
	}
	if end > 0 && s[end-1] == '\\' {
		start := end + utf8.RuneLen(r)
		end = indexRuneWithEscaping(s[start:], r)
		if end != -1 {
			end += start
		}
	}
	return end
}
