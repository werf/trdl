package trdl

import (
	"bytes"
	"fmt"
	"os"

	log "github.com/hashicorp/go-hclog"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/keyhelper"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
)

func GetAwsAccessKeyID() (string, error) {
	if v := os.Getenv("TRDL_AWS_ACCESS_KEY_ID"); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("required env var TRDL_AWS_ACCESS_KEY_ID")
}

func GetAwsSecretAccessKey() (string, error) {
	if v := os.Getenv("TRDL_AWS_SECRET_ACCESS_KEY"); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("required env var TRDL_AWS_SECRET_ACCESS_KEY")
}

func LoadFixturePublisherKeys() (publisher.TufRepoPrivKeys, error) {
	privKeys := publisher.TufRepoPrivKeys{}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherRootKey)), []byte(fixturePublisherRootPassphrase)); err != nil {
			return publisher.TufRepoPrivKeys{}, fmt.Errorf("error loading fixture root key: %s", err)
		} else {
			for _, key := range keys {
				privKeys.Root = append(privKeys.Root, key.Signer())
			}
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherSnapshotKey)), []byte(fixturePublisherSnapshotPassphrase)); err != nil {
			return publisher.TufRepoPrivKeys{}, fmt.Errorf("error loading fixture snapshot key: %s", err)
		} else {
			for _, key := range keys {
				privKeys.Snapshot = append(privKeys.Snapshot, key.Signer())
			}
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherTargetsKey)), []byte(fixturePublisherTargetsPassphrase)); err != nil {
			return publisher.TufRepoPrivKeys{}, fmt.Errorf("error loading fixture targets key: %s", err)
		} else {
			for _, key := range keys {
				privKeys.Targets = append(privKeys.Targets, key.Signer())
			}
		}
	}

	{
		if keys, err := keyhelper.LoadKeys(bytes.NewReader([]byte(fixturePublisherTimestampKey)), []byte(fixturePublisherTimestampPassphrase)); err != nil {
			return publisher.TufRepoPrivKeys{}, fmt.Errorf("error loading fixture timestamp key: %s", err)
		} else {
			for _, key := range keys {
				privKeys.Timestamp = append(privKeys.Timestamp, key.Signer())
			}
		}
	}

	log.L().Debug("privKeys: %#v\n", privKeys)

	return privKeys, nil
}

const (
	fixturePublisherRootKey = `
	{
		"encrypted": true,
		"data": {
			"kdf": {
				"name": "scrypt",
				"params": {
					"N": 32768,
					"r": 8,
					"p": 1
				},
				"salt": "fELZZAk3cI44uEgVcbjEkfoJUhehXCcSOP0kcos3y8g="
			},
			"cipher": {
				"name": "nacl/secretbox",
				"nonce": "4MksbI4zpr6KWArHUo7QoUQkjGml7GVC"
			},
			"ciphertext": "ZqnMMTOyr/Eunyh6QqVZSrD3CShNGsEc8xhLvk3Vl6Ff32H2wYw6jN6XhQT7vyLaFBNs7wnkRdyk0ANn9HwYSxXQpCA+byHQNCJumDu/x3KTxGwuyIst7Nbkg7alk46+NCMONtceSZrGZ3wBhS/7Vs4gcjaht6U/iffOOrOxKmLZ0HAl3MNLYw+zoZruHu4eA0TUvxSy7Sy+nz4aNJbSyAxHeATtjf2Hfgyf21E6dMKBjtDqvzgf3rp2BD7y4JY+7kw/K8H17vHCra9C4+N891Z3/tUYsI4fGythihK/7Kzu+OPodkh9xd7w5YStcst0qHyeV09ybBQ0B69DtgaLgHcw45GhJd8O7BXmcjjrQ1uv/xhrDAiant76DFE1QCdTc5RenQq+QOR+Q0gnWb7z773tbrGgsfffGZStvb0BcN3OPUZyC8rdIW1GjdHINYLX4kuVQ3CyigFcApzNgo0JAoGC3l/QJTY21R2SR12E9Fqhk9359qHf"
		}
	}
	`
	fixturePublisherRootPassphrase = `gfhjkm`

	fixturePublisherSnapshotKey = `
	{
		"encrypted": true,
		"data": {
			"kdf": {
				"name": "scrypt",
				"params": {
					"N": 32768,
					"r": 8,
					"p": 1
				},
				"salt": "IgaOsvm1zH+G62yk7ZDBmp7pIrLRKjAXjbmw/wJqSoA="
			},
			"cipher": {
				"name": "nacl/secretbox",
				"nonce": "dVGb4F5oQmqR4NN15Iyiu+mWX4CINRFn"
			},
			"ciphertext": "05NskumVUKV2I3UunEzP7yh7nPbeo06D67b6FMVhsxC51GI9H4H7IInGSX08F5ZHJUI8JpD7rMykNKgbMrQSRG05LkPoKXom1OhG58WDJ8nQAv7S5gIiG35V7iwfXK2XL+7oKOWwJfQyz9pY+XxtH6DqgHVvRzleZSXJTc8yEHXtWcl+TOxRb+EGv1veHHFmyW37RahRJ4vnQ8ILFhdUNr2THokzLc1T8d8zJPS77h5mWGEfraQxGq/To5YLW1pwPC2dlCwuoeYn2oFxU5f6lKOAR6a/5v+LRu60xnnmXlgS3YQbEW1P38AU7BiTJQHLSsNt7lzGDrYsqiXP/H48NNFgzMrvFetJcEQBeSsgg28ARqQQkpaAhAyVG2q/Ipk8b0KCmrpn0CDbdw94qgigiwI14rSOPZg4gYJ8HqgGu+hPvi8UcRaMIDorpU59MaluRIfRrXtdWvjqi5PIHe5J4ahsBldmF/pR0HiM7IR+VCEzrUK3TMBr"
		}
	}
	`
	fixturePublisherSnapshotPassphrase = `gfhjkm`

	fixturePublisherTargetsKey = `
	{
		"encrypted": true,
		"data": {
			"kdf": {
				"name": "scrypt",
				"params": {
					"N": 32768,
					"r": 8,
					"p": 1
				},
				"salt": "HD5D0XPfILWdmbeQSN+z7cu2wbqGp8QD5bUAP/EjROs="
			},
			"cipher": {
				"name": "nacl/secretbox",
				"nonce": "ADR36MN9Rqq+9nOGfbXMwPE4RqAmEpJ5"
			},
			"ciphertext": "orQJ5o0qen+HtXGqHevjhJHWDJj31OJStz3/0RU1fa/4GDnxP9UT7h0qzlPNCcVNjZJo7ArrYbljGJsx9q/hPcRSlmZm/sqVyUJyxARLkzTEqstb5W72CKFNvFyp4Eze8QtZNlnIT8dXJUhlLk7b9Y2BrYITIzNAO+uj4S3SFHjQux0pcPCxIRbhM3fTcMovDZ/PSAQ0m8g5Fn46T7OpLvYNgBJ3Ff6d6P0oqoN6hMgggqr8giKvhBvmP5miO9yEhd22SHXpApcvZqORbkUlbTYLGAYBoSrqg7DvClPncpWUWjyDnASg5V7PC7yerrtnZHWwumFGgO0sd/ecP4ad263rTD2EPlAnDuezK3lUuWKh4pwR5cYRVOfZ0sTNSc4wUE2E0hpuRaL71DAPQ2mzTaLsLJzhO+oYhRiLhT7pwk2lcjkmOvs330D3yK+7+usBiYzrhZDhsSi70BCbKOZIqGQmMzKwwbvI7knIsp+QwQ5khYsyhfoA"
		}
	}
	`
	fixturePublisherTargetsPassphrase = `gfhjkm`

	fixturePublisherTimestampKey = `
	{
		"encrypted": true,
		"data": {
			"kdf": {
				"name": "scrypt",
				"params": {
					"N": 32768,
					"r": 8,
					"p": 1
				},
				"salt": "/chCChlDJwerROVgwmCQSU+XieoVoks/rY/052yh9pQ="
			},
			"cipher": {
				"name": "nacl/secretbox",
				"nonce": "OLJYqD61knlmMTnt8SPhG5yzvTc+3PZq"
			},
			"ciphertext": "7KXIcao2M4uUCj/XMzEej44G5d27h7UTjhePvp7afriJwOwm109HulZ0o8JRtuEwZmpeE5Zs7kKtTQFJAkSXzUYLwGD3mk5r0KCkU0MrscWIjVMDF26FRwlNWyl7cATOG5u6MdaCAbzpBmBVuyhw71UZp61g5dI1wHU8lcjkDKnCWW6lh09T4jaaIvP0ZWsQT9Ktr4WvV/uU2Wp2sFtnjDRLpouDmXl3FFwGxvMVQakOfnmL0uAT1YMFZgWqNDYRrdRv276AljPpwYXPjXMJ7+i+8trilNzNylb26cf+fiB2DJH+E9/0NZgnvOHJZOMsKG95gLiG9L7sifZgMUYDDmwOa4GcCKQXRmUptIiNpgg5bShYIUsRj+dkVfgwkFiM9MvDXTP6aWMnjqfzN+z+8h9k1ctBCiGJHhEtbh5BTO14SES4XWJ9iDB/NECUeK+fl+F4YKFvTFDU2ftfBC8zt7TYRZhrj9Mnl4SfesstFn6CMNoqY+37"
		}
	}
	`
	fixturePublisherTimestampPassphrase = `gfhjkm`
)
