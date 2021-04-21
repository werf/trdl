package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	stagedFiles []*stagedFileDesc
	stagedMeta  map[string]json.RawMessage
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
		fmt.Printf("%q not staged!\n", name)

		exists, err := store.Filesystem.IsFileExist(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("error checking existance of %q: %s", name, err)
		}

		if exists {
			data, err := store.Filesystem.ReadFile(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("error reading %q: %s", name, err)
			}
			meta[name] = data
		} else {
			fmt.Printf("%q not found in the store filesystem!\n", name)
		}
	}

	fmt.Printf("-- NonAtomicTufStore.GetMeta -> meta[targets]: %s\n", meta["targets.json"])

	return meta, nil
}

func (store *NonAtomicTufStore) SetMeta(name string, meta json.RawMessage) error {
	store.stagedMeta[name] = meta
	return nil
}

func (store *NonAtomicTufStore) WalkStagedTargets(paths []string, targetsFn tuf.TargetsWalkFunc) error {
	if len(paths) == 0 {
		for _, fdesc := range store.stagedFiles {
			if err := targetsFn(fdesc.Path, bytes.NewReader(fdesc.Data)); err != nil {
				return err
			}
		}

		return nil
	}

HandlePaths:
	for _, path := range paths {
		for _, fdesc := range store.stagedFiles {
			if fdesc.Path == path {
				if err := targetsFn(path, bytes.NewReader(fdesc.Data)); err != nil {
					return err
				}

				continue HandlePaths
			}
		}

		return tuf.ErrFileNotFound{Path: path}
	}

	return nil
}

func (store *NonAtomicTufStore) AddStagedFile(path string, data []byte) error {
	store.stagedFiles = append(store.stagedFiles, &stagedFileDesc{
		Path: path,
		Data: data,
	})

	return nil
}

func (store *NonAtomicTufStore) Commit(consistentSnapshot bool, versions map[string]int, hashes map[string]data.Hashes) error {
	if consistentSnapshot {
		panic("not supported")
	}

	ctx := context.Background()

	for _, fdesc := range store.stagedFiles {
		var paths []string

		if strings.HasPrefix(fdesc.Path, "targets/") {
			paths = computeTargetPaths(consistentSnapshot, fdesc.Path, hashes)
		} else {
			paths = computeMetadataPaths(consistentSnapshot, fdesc.Path, versions)
		}

		for _, path := range paths {
			if err := store.Filesystem.WriteFile(ctx, path, fdesc.Data); err != nil {
				return fmt.Errorf("error writing %q into filesystem: %s", path, err)
			}
		}
	}

	for name, data := range store.stagedMeta {
		// TODO: perms 0644
		if err := store.Filesystem.WriteFile(ctx, name, data); err != nil {
			return fmt.Errorf("error writing %q: %s", name, err)
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
