package publisher

import (
	"fmt"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/pkg/keys"
)

type TufRepoPrivKeys struct {
	Root      *data.PrivateKey `json:"root"`
	Snapshot  *data.PrivateKey `json:"snapshot"`
	Targets   *data.PrivateKey `json:"targets"`
	Timestamp *data.PrivateKey `json:"timestamp"`
}

func (keys *TufRepoPrivKeys) SetKeyFromSigner(role string, signer keys.Signer) error {
	pk, err := signer.MarshalPrivateKey()
	if err != nil {
		return fmt.Errorf("unable to marshal signer private key: %w", err)
	}

	switch role {
	case "root":
		keys.Root = pk

	case "targets":
		keys.Targets = pk

	case "snapshot":
		keys.Snapshot = pk

	case "timestamp":
		keys.Timestamp = pk

	default:
		panic(fmt.Sprintf("unknown role %q", role))
	}

	return nil
}

func (privKeys TufRepoPrivKeys) SetupStoreSigners(store tuf.LocalStore) error {
	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		signer, err := privKeys.GetSigner(role)
		if err != nil {
			return fmt.Errorf("unable to get key signer for role %q: %w", role, err)
		}
		if signer != nil {
			if err := store.SaveSigner(role, signer); err != nil {
				return fmt.Errorf("unable to save key signer for role %q into tuf store: %w", role, err)
			}
		}
	}

	return nil
}

func (privKeys TufRepoPrivKeys) SetupTufRepoSigners(tufRepo *tuf.Repo) error {
	for _, desc := range []struct {
		role string
		key  *data.PrivateKey
	}{
		{"root", privKeys.Root},
		{"targets", privKeys.Targets},
		{"snapshot", privKeys.Snapshot},
		{"timestamp", privKeys.Timestamp},
	} {
		signer, err := keys.GetSigner(desc.key)
		if err != nil {
			return fmt.Errorf("unable to get key signer for role %s: %w", desc.role, err)
		}

		if err := tufRepo.AddPrivateKeyWithExpires(desc.role, signer, data.DefaultExpires("root")); err != nil {
			return fmt.Errorf("unable to add tuf repository private key for role %s: %w", desc.role, err)
		}
	}

	return nil
}

func (privKeys TufRepoPrivKeys) GetSigner(role string) (keys.Signer, error) {
	switch role {
	case "root":
		return toSigner(privKeys.Root)

	case "targets":
		return toSigner(privKeys.Targets)

	case "snapshot":
		return toSigner(privKeys.Snapshot)

	case "timestamp":
		return toSigner(privKeys.Timestamp)

	default:
		panic(fmt.Sprintf("unknown role %q", role))
	}
}

func toSigner(key *data.PrivateKey) (keys.Signer, error) {
	if key == nil {
		return nil, nil
	}
	return keys.GetSigner(key)
}
