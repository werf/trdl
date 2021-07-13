package publisher

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/keyhelper"
)

func LoadDevPublisherKeys() (TufRepoPrivKeys, error) {
	privKeys := TufRepoPrivKeys{}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherRootKey)), []byte(fixturePublisherRootPassphrase)); err != nil {
			return TufRepoPrivKeys{}, fmt.Errorf("error loading fixture root key: %s", err)
		} else {
			privKeys.Root = keys[0]
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherTargetsKey)), []byte(fixturePublisherTargetsPassphrase)); err != nil {
			return TufRepoPrivKeys{}, fmt.Errorf("error loading fixture targets key: %s", err)
		} else {
			privKeys.Targets = keys[0]
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherSnapshotKey)), []byte(fixturePublisherSnapshotPassphrase)); err != nil {
			return TufRepoPrivKeys{}, fmt.Errorf("error loading fixture snapshot key: %s", err)
		} else {
			privKeys.Snapshot = keys[0]
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherTimestampKey)), []byte(fixturePublisherTimestampPassphrase)); err != nil {
			return TufRepoPrivKeys{}, fmt.Errorf("error loading fixture timestamp key: %s", err)
		} else {
			privKeys.Timestamp = keys[0]
		}
	}

	hclog.L().Debug(fmt.Sprintf("privKeys: %#v", privKeys))

	return privKeys, nil
}

const (
	fixturePublisherRootKey = `
	{
		"encrypted": false,
		"data": [
			{
				"keytype": "ed25519",
				"scheme": "ed25519",
				"keyid_hash_algorithms": [
					"sha256",
					"sha512"
				],
				"keyval": {
					"public": "ee6467c8962e7dbc2789962e5c4b05ed5ecde332c31d92bde46d3547bf3242b3",
					"private": "5a8a41d8df82c7ccf025c9a038ec76988d23e052cd72306f1ff62f2df4c07b49ee6467c8962e7dbc2789962e5c4b05ed5ecde332c31d92bde46d3547bf3242b3"
				}
			}
		]
	}
	`
	fixturePublisherRootPassphrase = ``

	fixturePublisherTargetsKey = `
	{
		"encrypted": false,
		"data": [
			{
				"keytype": "ed25519",
				"scheme": "ed25519",
				"keyid_hash_algorithms": [
					"sha256",
					"sha512"
				],
				"keyval": {
					"public": "ea353ec57ea696f56ea374614ffd8d75dff8424043a4f34666f402db0eab175d",
					"private": "7d260c0474f60b006e9c2b39f965be2b5527eb134ce6f658bff62411a87ee066ea353ec57ea696f56ea374614ffd8d75dff8424043a4f34666f402db0eab175d"
				}
			}
		]
	}
	`
	fixturePublisherTargetsPassphrase = ``

	fixturePublisherSnapshotKey = `
	{
		"encrypted": false,
		"data": [
			{
				"keytype": "ed25519",
				"scheme": "ed25519",
				"keyid_hash_algorithms": [
					"sha256",
					"sha512"
				],
				"keyval": {
					"public": "89a4842e3463d43cdb7e3e393033fda84a69792eb54fd16797ece96dfccbf859",
					"private": "af0bc3298ac94b9ab1e3704ddd4458ef674368166379425ed3bb71f00d82506689a4842e3463d43cdb7e3e393033fda84a69792eb54fd16797ece96dfccbf859"
				}
			}
		]
	}
	`
	fixturePublisherSnapshotPassphrase = ``

	fixturePublisherTimestampKey = `
	{
		"encrypted": false,
		"data": [
			{
				"keytype": "ed25519",
				"scheme": "ed25519",
				"keyid_hash_algorithms": [
					"sha256",
					"sha512"
				],
				"keyval": {
					"public": "a941dbf4a7fe2a77ed68b74172de76d25bdb9b21d102dc5f3ba44b58cc8ab474",
					"private": "28974590481139f4dd27de03b248a6351064be4758ff16e32a8cff3d301b4a5aa941dbf4a7fe2a77ed68b74172de76d25bdb9b21d102dc5f3ba44b58cc8ab474"
				}
			}
		]
	}
	`
	fixturePublisherTimestampPassphrase = ``
)
