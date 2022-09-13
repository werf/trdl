package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/hashicorp/go-hclog"
	"github.com/samber/lo"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/pkg/keys"
	"github.com/theupdateframework/go-tuf/util"
)

type NonAtomicTufStore struct {
	Filesystem Filesystem
	PrivKeys   TufRepoPrivKeys

	stagedMeta  map[string]json.RawMessage
	stagedFiles []string
	logger      hclog.Logger

	signerForKeyID map[string]keys.Signer
	keyIDsForRole  map[string][]string
}

func NewNonAtomicTufStore(privKeys TufRepoPrivKeys, filesystem Filesystem, logger hclog.Logger) *NonAtomicTufStore {
	return &NonAtomicTufStore{
		Filesystem:     filesystem,
		PrivKeys:       privKeys,
		stagedMeta:     make(map[string]json.RawMessage),
		logger:         logger,
		signerForKeyID: make(map[string]keys.Signer),
		keyIDsForRole:  make(map[string][]string),
	}
}

var topLevelManifests = []string{
	"root.json",
	"targets.json",
	"snapshot.json",
	"timestamp.json",
}

func (store *NonAtomicTufStore) GetMeta() (map[string]json.RawMessage, error) {
	ctx := context.Background()

	meta := make(map[string]json.RawMessage)

	for _, name := range topLevelManifests {
		stagedData, hasKey := store.stagedMeta[name]
		if hasKey {
			meta[name] = stagedData
			continue
		}
		store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta %q not found in staged meta!", name))

		exists, err := store.Filesystem.IsFileExist(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("error checking existence of %q: %w", name, err)
		}

		if exists {
			data, err := store.Filesystem.ReadFileBytes(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("error reading %q: %w", name, err)
			}
			meta[name] = data
		} else {
			store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta %q not found in the store filesystem!", name))
		}
	}

	store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta -> meta[targets]: %s", meta["targets.json"]))

	return meta, nil
}

func (store *NonAtomicTufStore) SetMeta(name string, meta json.RawMessage) error {
	store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.SetMeta %q", name))
	store.stagedMeta[name] = meta
	return nil
}

func (store *NonAtomicTufStore) WalkStagedTargets(targetPathList []string, targetsFn tuf.TargetsWalkFunc) error {
	store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets %v", targetPathList))

	ctx := context.Background()

	runPipedFileReader := func(path string) io.Reader {
		buf := buffer.New(64 * 1024 * 1024)
		reader, writer := nio.Pipe(buf)

		go func() {
			store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets before ReadFileStream %q", path))

			if err := store.Filesystem.ReadFileStream(ctx, path, writer); err != nil {
				if err := writer.CloseWithError(fmt.Errorf("error reading file %q stream: %w", path, err)); err != nil {
					panic(fmt.Sprintf("ERROR: failed to close pipe writer while reading file %q stream: %s\n", path, err))
				}
			}

			store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets after ReadFileStream %q", path))
			if err := writer.Close(); err != nil {
				panic(fmt.Sprintf("ERROR: failed to close pipe writer while reading file %q stream: %s\n", path, err))
			}
		}()

		return reader
	}

	if len(targetPathList) == 0 {
		for _, filePath := range store.stagedFiles {
			if err := targetsFn(filePath, runPipedFileReader(path.Join("targets", filePath))); err != nil {
				return err
			}
		}

		return nil
	}

FilterStagedPaths:
	for _, targetPath := range targetPathList {
		for _, stagedPath := range store.stagedFiles {
			if stagedPath == targetPath {
				if err := targetsFn(targetPath, runPipedFileReader(path.Join("targets", targetPath))); err != nil {
					return err
				}

				continue FilterStagedPaths
			}
		}

		return tuf.ErrFileNotFound{Path: targetPath}
	}

	return nil
}

func (store *NonAtomicTufStore) StageTargetFile(ctx context.Context, targetPath string, data io.Reader) error {
	store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.StageTargetFile %q", targetPath))

	// NOTE: consistenSnapshot cannot be supported when adding staged files before commit stage

	if err := store.Filesystem.WriteFileStream(ctx, path.Join("targets", targetPath), data); err != nil {
		return fmt.Errorf("error writing %q into the store filesystem: %w", targetPath, err)
	}

	store.stagedFiles = append(store.stagedFiles, targetPath)

	return nil
}

func (store *NonAtomicTufStore) Commit(consistentSnapshot bool, versions map[string]int64, _ map[string]data.Hashes) error {
	store.logger.Debug("-- NonAtomicTufStore.Commit")
	if consistentSnapshot {
		panic("not supported")
	}

	ctx := context.Background()

	for name, data := range store.stagedMeta {
		// TODO: perms 0644

		for _, metadataPath := range computeMetadataPaths(consistentSnapshot, name, versions) {
			store.logger.Debug(fmt.Sprintf("-- NonAtomicTufStore.Commit storing metadata path %q into the filesystem", metadataPath))

			if err := store.Filesystem.WriteFileBytes(ctx, metadataPath, data); err != nil {
				return fmt.Errorf("error writing metadata path %q into the filesystem: %w", metadataPath, err)
			}
		}
	}

	store.stagedFiles = nil
	store.stagedMeta = make(map[string]json.RawMessage)

	return nil
}

func (store *NonAtomicTufStore) FileIsStaged(filename string) bool {
	_, ok := store.stagedMeta[filename]
	return ok
}

func (store *NonAtomicTufStore) GetSigners(role string) ([]keys.Signer, error) {
	keyIDs, ok := store.keyIDsForRole[role]
	if ok {
		return store.SignersForKeyIDs(keyIDs), nil
	}
	return nil, nil
}

func (store *NonAtomicTufStore) SignersForKeyIDs(keyIDs []string) []keys.Signer {
	signers := []keys.Signer{}
	keyIDsSeen := map[string]struct{}{}

	for _, keyID := range keyIDs {
		signer, ok := store.signerForKeyID[keyID]
		if !ok {
			continue
		}
		addSigner := false

		for _, skid := range signer.PublicData().IDs() {
			if _, seen := keyIDsSeen[skid]; !seen {
				addSigner = true
			}

			keyIDsSeen[skid] = struct{}{}
		}

		if addSigner {
			signers = append(signers, signer)
		}
	}

	return signers
}

func (store *NonAtomicTufStore) SaveSigner(role string, signer keys.Signer) error {
	keyIDs := signer.PublicData().IDs()

	for _, keyID := range keyIDs {
		store.signerForKeyID[keyID] = signer
	}

	mergedKeyIDs := lo.Uniq[string](append(store.keyIDsForRole[role], keyIDs...))
	store.keyIDsForRole[role] = mergedKeyIDs

	if err := store.PrivKeys.SetKeyFromSigner(role, signer); err != nil {
		return fmt.Errorf("unable to set private key for role %q: %w", role, err)
	}

	return nil
}

func (m *NonAtomicTufStore) Clean() error {
	panic("not supported")
}

func computeMetadataPaths(consistentSnapshot bool, name string, versions map[string]int64) []string {
	if consistentSnapshot {
		panic("not supported")
	}

	copyVersion := false

	switch name {
	case "root.json":
		copyVersion = true
	case "timestamp.json":
		copyVersion = false
	default:
		copyVersion = false
	}

	paths := []string{name}
	if copyVersion {
		paths = append(paths, util.VersionedPath(name, versions[name]))
	}

	return paths
}
