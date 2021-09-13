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
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/sign"
	"github.com/theupdateframework/go-tuf/util"
)

type TufRepoPrivKeys struct {
	Root      *sign.PrivateKey `json:"root"`
	Snapshot  *sign.PrivateKey `json:"snapshot"`
	Targets   *sign.PrivateKey `json:"targets"`
	Timestamp *sign.PrivateKey `json:"timestamp"`
}

type NonAtomicTufStore struct {
	PrivKeys   TufRepoPrivKeys
	Filesystem Filesystem

	stagedMeta  map[string]json.RawMessage
	stagedFiles []string
}

func NewNonAtomicTufStore(privKeys TufRepoPrivKeys, filesystem Filesystem) *NonAtomicTufStore {
	return &NonAtomicTufStore{
		PrivKeys:   privKeys,
		Filesystem: filesystem,
		stagedMeta: make(map[string]json.RawMessage),
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
		hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta %q not found in staged meta!", name))

		exists, err := store.Filesystem.IsFileExist(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("error checking existance of %q: %s", name, err)
		}

		if exists {
			data, err := store.Filesystem.ReadFileBytes(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("error reading %q: %s", name, err)
			}
			meta[name] = data
		} else {
			hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta %q not found in the store filesystem!", name))
		}
	}

	hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.GetMeta -> meta[targets]: %s", meta["targets.json"]))

	return meta, nil
}

func (store *NonAtomicTufStore) SetMeta(name string, meta json.RawMessage) error {
	hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.SetMeta %q", name))
	store.stagedMeta[name] = meta
	return nil
}

func (store *NonAtomicTufStore) WalkStagedTargets(targetPathList []string, targetsFn tuf.TargetsWalkFunc) error {
	hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets %v", targetPathList))

	ctx := context.Background()

	runPipedFileReader := func(path string) io.Reader {
		buf := buffer.New(64 * 1024 * 1024)
		reader, writer := nio.Pipe(buf)

		go func() {
			hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets before ReadFileStream %q", path))

			if err := store.Filesystem.ReadFileStream(ctx, path, writer); err != nil {
				if err := writer.CloseWithError(fmt.Errorf("error reading file %q stream: %s", path, err)); err != nil {
					panic(fmt.Sprintf("ERROR: failed to close pipe writer while reading file %q stream: %s\n", path, err))
				}
			}

			hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.WalkStagedTargets after ReadFileStream %q", path))
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
	hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.StageTargetFile %q", targetPath))

	// NOTE: consistenSnapshot cannot be supported when adding staged files before commit stage

	if err := store.Filesystem.WriteFileStream(ctx, path.Join("targets", targetPath), data); err != nil {
		return fmt.Errorf("error writing %q into the store filesystem: %s", targetPath, err)
	}

	store.stagedFiles = append(store.stagedFiles, targetPath)

	return nil
}

func (store *NonAtomicTufStore) Commit(consistentSnapshot bool, versions map[string]int, _ map[string]data.Hashes) error {
	hclog.L().Debug("-- NonAtomicTufStore.Commit")
	if consistentSnapshot {
		panic("not supported")
	}

	ctx := context.Background()

	for name, data := range store.stagedMeta {
		// TODO: perms 0644

		for _, metadataPath := range computeMetadataPaths(consistentSnapshot, name, versions) {
			hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.Commit storing metadata path %q into the filesystem", metadataPath))

			if err := store.Filesystem.WriteFileBytes(ctx, metadataPath, data); err != nil {
				return fmt.Errorf("error writing metadata path %q into the filesystem: %s", metadataPath, err)
			}
		}
	}

	store.stagedFiles = nil
	store.stagedMeta = make(map[string]json.RawMessage)

	return nil
}

func (store *NonAtomicTufStore) GetSigningKeys(role string) ([]sign.Signer, error) {
	hclog.L().Debug(fmt.Sprintf("-- NonAtomicTufStore.GetSigningKeys(%q) store.PrivKeys=%#v", role, store.PrivKeys))

	toSigners := func(key *sign.PrivateKey) []sign.Signer {
		if key != nil {
			return []sign.Signer{key.Signer()}
		}
		return nil
	}

	switch role {
	case "root":
		return toSigners(store.PrivKeys.Root), nil

	case "targets":
		return toSigners(store.PrivKeys.Targets), nil

	case "snapshot":
		return toSigners(store.PrivKeys.Snapshot), nil

	case "timestamp":
		return toSigners(store.PrivKeys.Timestamp), nil

	default:
		panic(fmt.Sprintf("unknown role %q", role))
	}
}

func (store *NonAtomicTufStore) SavePrivateKey(role string, key *sign.PrivateKey) error {
	switch role {
	case "root":
		store.PrivKeys.Root = key

	case "targets":
		store.PrivKeys.Targets = key

	case "snapshot":
		store.PrivKeys.Snapshot = key

	case "timestamp":
		store.PrivKeys.Timestamp = key

	default:
		panic(fmt.Sprintf("unknown role %q", role))
	}

	return nil
}

func (m *NonAtomicTufStore) Clean() error {
	panic("not supported")
}

func computeTargetPaths(consistentSnapshot bool, name string, _ map[string]data.Hashes) []string {
	if consistentSnapshot {
		panic("not supported")
	}

	return []string{name}
}

func computeMetadataPaths(consistentSnapshot bool, name string, versions map[string]int) []string {
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
