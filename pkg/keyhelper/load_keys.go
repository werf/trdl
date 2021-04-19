package keyhelper

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/theupdateframework/go-tuf/encrypted"
	"github.com/theupdateframework/go-tuf/sign"
)

type PersistedKeys struct {
	Encrypted bool            `json:"encrypted"`
	Data      json.RawMessage `json:"data"`
}

func LoadKeys(r io.Reader, passphrase []byte) ([]*sign.PrivateKey, error) {
	pk := &PersistedKeys{}

	if err := json.NewDecoder(r).Decode(pk); err != nil {
		return nil, fmt.Errorf("error unmarshalling keys json data: %s", err)
	}

	var keys []*sign.PrivateKey

	if !pk.Encrypted {
		if err := json.Unmarshal(pk.Data, &keys); err != nil {
			return nil, fmt.Errorf("error unmarshalling private key json data: %s", err)
		}

		return keys, nil
	}

	if err := encrypted.Unmarshal(pk.Data, &keys, passphrase); err != nil {
		return nil, fmt.Errorf("unable to decrypt data: %s", err)
	}

	return keys, nil
}
