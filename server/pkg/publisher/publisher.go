package publisher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/config"
)

const (
	storageKeyTufRepositoryKeys = "tuf_repository_keys"
)

var ErrUninitializedRepositoryKeys = errors.New("uninitialized repository keys")

type RepositoryOptions struct {
	S3Endpoint        string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3BucketName      string

	InitializeKeys bool
}

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

type Publisher struct {
	mu sync.Mutex
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (publisher *Publisher) RotateRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error {
	updated, updatedPrivKeys, err := repository.RotatePrivKeys(ctx)
	if err != nil {
		return fmt.Errorf("unable to rotate TUF repository keys: %s", err)
	}

	if updated {
		entry, err := logical.StorageEntryJSON(storageKeyTufRepositoryKeys, updatedPrivKeys)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %s", storageKeyTufRepositoryKeys, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting private keys json entry by key %q into the storage: %s", storageKeyTufRepositoryKeys, err)
		}

		hclog.L().Info("Successfully rotated repository private keys")
	}

	return nil
}

func (publisher *Publisher) UpdateTimestamps(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error {
	return repository.UpdateTimestamps(ctx)
}

type initRepositoryKeysOptions struct {
	InitializeKeys bool
}

func (publisher *Publisher) initRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface, opts initRepositoryKeysOptions) error {
	entry, err := storage.Get(ctx, storageKeyTufRepositoryKeys)
	if err != nil {
		return fmt.Errorf("error getting storage private keys json entry by the key %q: %s", storageKeyTufRepositoryKeys, err)
	}

	if entry == nil {
		if !opts.InitializeKeys {
			return ErrUninitializedRepositoryKeys
		}

		hclog.L().Debug("Will generate new repository private keys")

		if err := repository.GenPrivKeys(); err != nil {
			return fmt.Errorf("error generating repository private keys: %s", err)
		}

		privKeys := repository.GetPrivKeys()

		entry, err := logical.StorageEntryJSON(storageKeyTufRepositoryKeys, privKeys)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %s", storageKeyTufRepositoryKeys, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting private keys json entry by key %q into the storage: %s", storageKeyTufRepositoryKeys, err)
		}

		hclog.L().Info("Generated new repository private keys")

		return nil
	}

	var privKeys TufRepoPrivKeys
	if err := entry.DecodeJSON(&privKeys); err != nil {
		return fmt.Errorf("unable to decode keys json by the %q storage key:\n%s---\n%s", storageKeyTufRepositoryKeys, entry.Value, err)
	}

	if err := repository.SetPrivKeys(privKeys); err != nil {
		return fmt.Errorf("unable to set private keys into repository: %s", err)
	}

	hclog.L().Info("Loaded repository private keys from the storage")

	return nil
}

func (publisher *Publisher) GetRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) (RepositoryInterface, error) {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	awsConfig := &aws.Config{
		Endpoint:    aws.String(options.S3Endpoint),
		Region:      aws.String(options.S3Region),
		Credentials: credentials.NewStaticCredentials(options.S3AccessKeyID, options.S3SecretAccessKey, ""),
	}

	repository, err := NewRepositoryWithOptions(
		S3Options{AwsConfig: awsConfig, BucketName: options.S3BucketName},
		TufRepoOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository handle: %s", err)
	}

	if err := repository.Init(); err != nil {
		return nil, fmt.Errorf("error initializing repository: %s", err)
	}

	if err := publisher.initRepositoryKeys(ctx, storage, repository, initRepositoryKeysOptions{InitializeKeys: options.InitializeKeys}); err == ErrUninitializedRepositoryKeys {
		return nil, ErrUninitializedRepositoryKeys
	} else if err != nil {
		return nil, fmt.Errorf("error initializing repository keys: %s", err)
	}

	return repository, nil
}

func (publisher *Publisher) StageReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, path string, data io.Reader) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

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

	return repository.StageTarget(ctx, filepath.Join("releases", releaseName, path), data)
}

func (publisher *Publisher) StageChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

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

			if err := repository.StageTarget(ctx, publishPath, bytes.NewBuffer([]byte(chnl.Version+"\n"))); err != nil {
				return fmt.Errorf("error publishing %q: %s", publishPath, err)
			}
		}
	}

	return nil
}

func (publisher *Publisher) StageInMemoryFiles(ctx context.Context, repository RepositoryInterface, files []*InMemoryFile) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	for _, file := range files {
		if err := repository.StageTarget(ctx, file.Name, bytes.NewReader(file.Data)); err != nil {
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
