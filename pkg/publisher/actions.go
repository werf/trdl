package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/sign"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
)

const (
	storageKeyTufRepositoryRootKey      = "tuf_repository_root_key"
	storageKeyTufRepositoryTargetsKey   = "tuf_repository_targets_key"
	storageKeyTufRepositorySnapshotKey  = "tuf_repository_snapshot_key"
	storageKeyTufRepositoryTimestampKey = "tuf_repository_timestamp_key"
)

type RepositoryOptions struct {
	S3Endpoint        string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3BucketName      string
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

func (m *Publisher) InitRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	awsConfig := &aws.Config{
		Endpoint:    aws.String(options.S3Endpoint),
		Region:      aws.String(options.S3Region),
		Credentials: credentials.NewStaticCredentials(options.S3AccessKeyID, options.S3SecretAccessKey, ""),
	}

	publisherRepository, err := NewRepositoryWithOptions(
		S3Options{AwsConfig: awsConfig, BucketName: options.S3BucketName},
		TufRepoOptions{},
	)
	if err != nil {
		return fmt.Errorf("error initializing publisher repository: %s", err)
	}

	if err := publisherRepository.TufRepo.Init(false); err == tuf.ErrInitNotAllowed {
		return fmt.Errorf("found existing targets in the tuf repository in the s3 storage, cannot reinitialize already initialized repository. Please use new s3 bucket or remove existing targets")
	} else if err != nil {
		return fmt.Errorf("unable to init tuf repository: %s", err)
	}

	_, err = publisherRepository.TufRepo.GenKey("root")
	if err != nil {
		return fmt.Errorf("error generating tuf repository root key: %s", err)
	}

	_, err = publisherRepository.TufRepo.GenKey("targets")
	if err != nil {
		return fmt.Errorf("error generating tuf repository targets key: %s", err)
	}

	_, err = publisherRepository.TufRepo.GenKey("snapshot")
	if err != nil {
		return fmt.Errorf("error generating tuf repository snapshot key: %s", err)
	}

	_, err = publisherRepository.TufRepo.GenKey("timestamp")
	if err != nil {
		return fmt.Errorf("error generating tuf repository timestamp key: %s", err)
	}

	for _, storeKey := range []struct {
		Key        *sign.PrivateKey
		StorageKey string
	}{
		{publisherRepository.TufStore.PrivKeys.Root, storageKeyTufRepositoryRootKey},
		{publisherRepository.TufStore.PrivKeys.Targets, storageKeyTufRepositoryTargetsKey},
		{publisherRepository.TufStore.PrivKeys.Snapshot, storageKeyTufRepositorySnapshotKey},
		{publisherRepository.TufStore.PrivKeys.Timestamp, storageKeyTufRepositoryTimestampKey},
	} {
		entry, err := logical.StorageEntryJSON(storeKey.StorageKey, storeKey.Key)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %s", storeKey.StorageKey, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting json entry by key %q into the storage: %s", storeKey.StorageKey, err)
		}
	}

	if err := publisherRepository.PublishTarget(ctx, "initialized_at", bytes.NewBuffer([]byte(time.Now().UTC().String()+"\n"))); err != nil {
		return fmt.Errorf("unable to publish initialization timestamp: %s", err)
	}

	if err := publisherRepository.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit initialized tuf repository: %s", err)
	}

	return nil
}

func (m *Publisher) GetRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) (RepositoryInterface, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	awsConfig := &aws.Config{
		Endpoint:    aws.String(options.S3Endpoint),
		Region:      aws.String(options.S3Region),
		Credentials: credentials.NewStaticCredentials(options.S3AccessKeyID, options.S3SecretAccessKey, ""),
	}

	publisherRepository, err := NewRepositoryWithOptions(
		S3Options{AwsConfig: awsConfig, BucketName: options.S3BucketName},
		TufRepoOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository handle: %s", err)
	}

	for _, desc := range []struct {
		storageKey    string
		targetPrivKey **sign.PrivateKey
	}{
		{storageKeyTufRepositoryRootKey, &publisherRepository.TufStore.PrivKeys.Root},
		{storageKeyTufRepositoryTargetsKey, &publisherRepository.TufStore.PrivKeys.Targets},
		{storageKeyTufRepositorySnapshotKey, &publisherRepository.TufStore.PrivKeys.Snapshot},
		{storageKeyTufRepositoryTimestampKey, &publisherRepository.TufStore.PrivKeys.Timestamp},
	} {
		entry, err := storage.Get(ctx, desc.storageKey)
		if err != nil {
			return nil, fmt.Errorf("error getting storage json entry by the key %q: %s", desc.storageKey, err)
		}

		if entry == nil {
			return nil, fmt.Errorf("%q storage key not found", desc.storageKey)
		}

		privKey := &sign.PrivateKey{}

		if err := entry.DecodeJSON(privKey); err != nil {
			return nil, fmt.Errorf("unable to decode json by the %q storage key:\n%s---\n%s", desc.storageKey, entry.Value, err)
		}

		*desc.targetPrivKey = privKey
	}

	return publisherRepository, nil
}

func (m *Publisher) PublishReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, path string, data io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *Publisher) PublishChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *Publisher) PublishInMemoryFiles(ctx context.Context, repository RepositoryInterface, files []*InMemoryFile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
