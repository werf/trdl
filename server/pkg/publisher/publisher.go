package publisher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/config"
	"github.com/werf/trdl/server/pkg/pgp"
	"github.com/werf/trdl/server/pkg/util"
)

const (
	storageKeyTufRepositoryKeys = "tuf_repository_keys"
	storageKeyPGPSigningKey     = "pgp_signing_key"
)

var (
	ErrUninitializedRepositoryKeys = errors.New("uninitialized repository keys")
	ErrUninitializedPGPSigningKey  = errors.New("uninitialized pgp signing key")
)

type RepositoryOptions struct {
	S3Endpoint        string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3BucketName      string

	InitializeTUFKeys       bool
	InitializePGPSigningKey bool
}

type InMemoryFile struct {
	Name string
	Data []byte
}

func NewErrIncorrectTargetPath(path string) error {
	return fmt.Errorf(`got incorrect target path %q: expected path in format <os>-<arch>/... where os can be either "any", "linux", "darwin" or "windows", and arch can be either "any", "amd64" or "arm64"`, path)
}

type Publisher struct {
	mu     sync.Mutex
	logger hclog.Logger

	PGPSigningKey *pgp.RSASigningKey
}

func NewPublisher(logger hclog.Logger) *Publisher {
	return &Publisher{logger: logger}
}

func (publisher *Publisher) RotateRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error {
	updated, updatedPrivKeys, err := repository.RotatePrivKeys(ctx)
	if err != nil {
		return fmt.Errorf("unable to rotate TUF repository keys: %w", err)
	}

	if updated {
		entry, err := logical.StorageEntryJSON(storageKeyTufRepositoryKeys, updatedPrivKeys)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %w", storageKeyTufRepositoryKeys, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting private keys json entry by key %q into the storage: %w", storageKeyTufRepositoryKeys, err)
		}

		publisher.logger.Info("Successfully rotated repository private keys")
	}

	return nil
}

func (publisher *Publisher) UpdateTimestamps(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error {
	return repository.UpdateTimestamps(ctx)
}

type setRepositoryKeysOptions struct {
	InitializeKeys bool
}

func (publisher *Publisher) setRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface, opts setRepositoryKeysOptions) error {
	entry, err := storage.Get(ctx, storageKeyTufRepositoryKeys)
	if err != nil {
		return fmt.Errorf("error getting storage private keys json entry by the key %q: %w", storageKeyTufRepositoryKeys, err)
	}

	if entry == nil {
		if !opts.InitializeKeys {
			return ErrUninitializedRepositoryKeys
		}

		publisher.logger.Debug("Will generate new repository private keys")

		if err := repository.GenPrivKeys(); err != nil {
			return fmt.Errorf("error generating repository private keys: %w", err)
		}

		privKeys := repository.GetPrivKeys()

		entry, err := logical.StorageEntryJSON(storageKeyTufRepositoryKeys, privKeys)
		if err != nil {
			return fmt.Errorf("error creating storage json entry by key %q: %w", storageKeyTufRepositoryKeys, err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("error putting private keys json entry by key %q into the storage: %w", storageKeyTufRepositoryKeys, err)
		}

		publisher.logger.Info("Generated new repository private keys")

		return nil
	}

	var privKeys TufRepoPrivKeys
	if err := entry.DecodeJSON(&privKeys); err != nil {
		return fmt.Errorf("unable to decode keys json by the %q storage key:\n%s---\n%w", storageKeyTufRepositoryKeys, entry.Value, err)
	}

	if err := repository.SetPrivKeys(privKeys); err != nil {
		return fmt.Errorf("unable to set private keys into repository: %w", err)
	}

	publisher.logger.Info("Loaded repository private keys from the storage")

	return nil
}

func (publisher *Publisher) deletePGPSigningKey(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, storageKeyPGPSigningKey)
}

func (publisher *Publisher) fetchPGPSigningKey(ctx context.Context, storage logical.Storage, initializeKey bool) (*pgp.RSASigningKey, error) {
	entry, err := storage.Get(ctx, storageKeyPGPSigningKey)
	if err != nil {
		return nil, fmt.Errorf("error getting storage pgp signing key json entry by storage key %q: %w", storageKeyPGPSigningKey, err)
	}

	if entry == nil {
		if !initializeKey {
			return nil, ErrUninitializedPGPSigningKey
		}

		hclog.L().Debug("Will generate a new pgp signing key")

		key, err := pgp.GenerateRSASigningKey()
		if err != nil {
			return nil, fmt.Errorf("unable to generate new rsa pgp signing key: %w", err)
		}

		serializedKey := bytes.NewBuffer(nil)
		if err := key.SerializeFull(serializedKey); err != nil {
			return nil, fmt.Errorf("unable to serialize pgp signing key: %w", err)
		}

		entry := &logical.StorageEntry{
			Key:   storageKeyPGPSigningKey,
			Value: serializedKey.Bytes(),
		}

		if err := storage.Put(ctx, entry); err != nil {
			return nil, fmt.Errorf("error putting pgp signing key by storage key %q: %w", storageKeyPGPSigningKey, err)
		}

		hclog.L().Info("Generated new PGP signing key")

		return key, nil
	}

	key, err := pgp.ParseRSASigningKey(bytes.NewReader(entry.Value))
	if err != nil {
		return nil, fmt.Errorf("unable to parse pgp signing key by the %q storage key:\n%s\n---%w", storageKeyPGPSigningKey, entry.Value, err)
	}
	return key, nil
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
		publisher.logger,
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher repository handle: %w", err)
	}

	if err := repository.Init(); err != nil {
		return nil, fmt.Errorf("error initializing repository: %w", err)
	}

	if err := publisher.setRepositoryKeys(ctx, storage, repository, setRepositoryKeysOptions{InitializeKeys: options.InitializeTUFKeys}); err == ErrUninitializedRepositoryKeys {
		return nil, ErrUninitializedRepositoryKeys
	} else if err != nil {
		return nil, fmt.Errorf("error initializing repository keys: %w", err)
	}

	pgpSigningKey, err := publisher.fetchPGPSigningKey(ctx, storage, options.InitializePGPSigningKey)
	if err != nil {
		return nil, fmt.Errorf("error fetching pgp signing key: %w", err)
	}
	publisher.PGPSigningKey = pgpSigningKey

	return repository, nil
}

func (publisher *Publisher) StageReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, releaseFilePath string, data io.Reader) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	pathParts := SplitFilepath(filepath.Clean(releaseFilePath))
	if len(pathParts) == 0 {
		return NewErrIncorrectTargetPath(releaseFilePath)
	}

	osAndArchParts := strings.SplitN(pathParts[0], "-", 2)

	switch osAndArchParts[0] {
	case "any", "linux", "darwin", "windows":
	default:
		return NewErrIncorrectTargetPath(releaseFilePath)
	}

	switch osAndArchParts[1] {
	case "any", "amd64", "arm64":
	default:
		return NewErrIncorrectTargetPath(releaseFilePath)
	}

	gpgSignErrCh := make(chan error)
	gpgSignDoneCh := make(chan struct{})
	gpgSignBuf := bytes.NewBuffer(nil)

	r := util.BufferedPipedWriterProcess(func(w io.WriteCloser) {
		signDataReader := io.TeeReader(data, w)

		if err := pgp.SignDataStream(gpgSignBuf, signDataReader, publisher.PGPSigningKey); err != nil {
			gpgSignErrCh <- fmt.Errorf("unable to sign %q: %w", releaseFilePath, err)
			return
		}

		if err := w.Close(); err != nil {
			gpgSignErrCh <- fmt.Errorf("unable to close sign data reader stream: %w", err)
			return
		}

		close(gpgSignDoneCh)
	})

	pathToReleaseTarget := path.Join("releases", releaseName, releaseFilePath)
	hclog.L().Debug(fmt.Sprintf("Stage release target %q ...\n", pathToReleaseTarget))
	if err := repository.StageTarget(ctx, pathToReleaseTarget, r); err != nil {
		return fmt.Errorf("unable to stage release target %q into the repository: %w", pathToReleaseTarget, err)
	}

	select {
	case <-gpgSignDoneCh:
	case err := <-gpgSignErrCh:
		return err
	}

	pathToReleaseTargetSignature := path.Join("signatures", releaseName, fmt.Sprintf("%s.sig", releaseFilePath))
	hclog.L().Debug(fmt.Sprintf("Stage release target signature %q ...\n", pathToReleaseTargetSignature))
	if err := repository.StageTarget(ctx, pathToReleaseTargetSignature, bytes.NewBufferString(gpgSignBuf.String())); err != nil {
		return fmt.Errorf("unable to stage release target signature %q into the repository: %w", pathToReleaseTargetSignature, err)
	}

	return nil
}

func (publisher *Publisher) StageChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	// publish /channels/GROUP/CHANNEL -> VERSION
	for _, grp := range trdlChannelsConfig.Groups {
		for _, chnl := range grp.Channels {
			publishPath := path.Join("channels", grp.Name, chnl.Name)

			if err := repository.StageTarget(ctx, publishPath, bytes.NewBuffer([]byte(chnl.Version+"\n"))); err != nil {
				return fmt.Errorf("error publishing %q: %w", publishPath, err)
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
			return fmt.Errorf("error publishing %q: %w", file.Name, err)
		}
	}

	return nil
}

func (publisher *Publisher) GetExistingReleases(ctx context.Context, repository RepositoryInterface) ([]string, error) {
	existingTargets, err := repository.GetTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting existing targets: %w", err)
	}

	var releases []string

ScanTargets:
	for _, target := range existingTargets {
		if strings.HasPrefix(target, "releases/") {
			pathParts := strings.SplitN(strings.TrimPrefix(target, "releases/"), "/", 2)
			releaseName := strings.TrimPrefix(pathParts[0], "v")

			for _, r := range releases {
				if r == releaseName {
					continue ScanTargets
				}
			}

			releases = append(releases, releaseName)
		}
	}

	return releases, nil
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
