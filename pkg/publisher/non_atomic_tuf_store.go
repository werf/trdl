package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/sign"
	"github.com/theupdateframework/go-tuf/util"
)

type TufRepoPrivKeys struct {
	Root      []sign.Signer
	Snapshot  []sign.Signer
	Targets   []sign.Signer
	Timestamp []sign.Signer
}

type stagedFileDesc struct {
	Path string
	Data []byte
}

type NonAtomicTufStore struct {
	PrivKeys   TufRepoPrivKeys
	Filesystem Filesystem

	stagedMeta  map[string]json.RawMessage
	stagedFiles []string
}

func NewNonAtomicTufStore(privKeys TufRepoPrivKeys, Filesystem Filesystem) *NonAtomicTufStore {
	return &NonAtomicTufStore{
		PrivKeys:   privKeys,
		Filesystem: Filesystem,
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
		fmt.Printf("-- NonAtomicTufStore.GetMeta %q not found in staged meta!\n", name)

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
			fmt.Printf("-- NonAtomicTufStore.GetMeta %q not found in the store filesystem!\n", name)
		}
	}

	fmt.Printf("-- NonAtomicTufStore.GetMeta -> meta[targets]: %s\n", meta["targets.json"])

	return meta, nil
}

func (store *NonAtomicTufStore) SetMeta(name string, meta json.RawMessage) error {
	fmt.Printf("-- NonAtomicTufStore.SetMeta %q\n", name)
	store.stagedMeta[name] = meta
	return nil
}

func (store *NonAtomicTufStore) WalkStagedTargets(paths []string, targetsFn tuf.TargetsWalkFunc) error {
	fmt.Printf("-- NonAtomicTufStore.WalkStagedTargets %v\n", paths)

	ctx := context.Background()

	runPipedFileReader := func(path string) io.Reader {
		reader, writer := io.Pipe()

		go func() {
			if err := store.Filesystem.ReadFileStream(ctx, path, writer); err != nil {
				if err := writer.CloseWithError(fmt.Errorf("error reading file %q stream: %s", path, err)); err != nil {
					panic(fmt.Sprintf("ERROR: failed to close pipe writer while reading file %q stream: %s\n", path, err))
				}
			}

			if err := writer.Close(); err != nil {
				panic(fmt.Sprintf("ERROR: failed to close pipe writer while reading file %q stream: %s\n", path, err))
			}
		}()

		return reader
	}

	if len(paths) == 0 {
		for _, path := range store.stagedFiles {
			if err := targetsFn(path, runPipedFileReader(path)); err != nil {
				return err
			}
		}

		return nil
	}

FilterStagedPaths:
	for _, path := range paths {
		for _, stagedPath := range store.stagedFiles {
			if stagedPath == path {
				if err := targetsFn(path, runPipedFileReader(path)); err != nil {
					return err
				}

				continue FilterStagedPaths
			}
		}

		return tuf.ErrFileNotFound{Path: path}
	}

	return nil
}

func (store *NonAtomicTufStore) StageTargetFile(ctx context.Context, path string, data io.Reader) error {
	fmt.Printf("-- NonAtomicTufStore.StageTargetFile %q\n", path)

	// NOTE: consistenSnapshot cannot be supported when adding staged files before commit stage

	if err := store.Filesystem.WriteFileStream(ctx, path, data); err != nil {
		return fmt.Errorf("error writing %q into the store filesystem: %s", path, err)
	}

	store.stagedFiles = append(store.stagedFiles, path)

	return nil
}

func (store *NonAtomicTufStore) Commit(consistentSnapshot bool, versions map[string]int, hashes map[string]data.Hashes) error {
	fmt.Printf("-- NonAtomicTufStore.Commit\n")
	if consistentSnapshot {
		panic("not supported")
	}

	ctx := context.Background()

	for name, data := range store.stagedMeta {
		// TODO: perms 0644

		for _, path := range computeMetadataPaths(consistentSnapshot, name, versions) {
			fmt.Printf("-- NonAtomicTufStore.Commit storing metadata path %q into the filesystem\n", path)

			if err := store.Filesystem.WriteFileBytes(ctx, path, data); err != nil {
				return fmt.Errorf("error writing metadata path %q into the filesystem: %s", path, err)
			}
		}
	}

	store.stagedFiles = nil
	store.stagedMeta = make(map[string]json.RawMessage)

	return nil
}

func (store *NonAtomicTufStore) GetSigningKeys(role string) ([]sign.Signer, error) {
	fmt.Printf("-- NonAtomicTufStore.GetSigningKeys(%q)\n", role)

	switch role {
	case "root":
		return store.PrivKeys.Root, nil

	case "snapshot":
		return store.PrivKeys.Snapshot, nil

	case "targets":
		return store.PrivKeys.Targets, nil

	case "timestamp":
		return store.PrivKeys.Timestamp, nil

	default:
		panic(fmt.Sprintf("unknown role %q", role))
	}
}

func (store *NonAtomicTufStore) SavePrivateKey(role string, key *sign.PrivateKey) error {
	panic("not supported")
}

func (m *NonAtomicTufStore) Clean() error {
	panic("not supported")
}

func computeTargetPaths(consistentSnapshot bool, name string, hashes map[string]data.Hashes) []string {
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
