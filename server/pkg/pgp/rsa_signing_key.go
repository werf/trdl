package pgp

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

type RSASigningKey struct {
	Entity *openpgp.Entity
}

func (key *RSASigningKey) SerializePublicKey(out io.Writer) error {
	armoredOut, err := armor.Encode(out, openpgp.PublicKeyType, nil)
	if err != nil {
		return fmt.Errorf("unable to prepare armored writer: %w", err)
	}

	if err := key.Entity.Serialize(armoredOut); err != nil {
		return err
	}

	if err := armoredOut.Close(); err != nil {
		return fmt.Errorf("unable to close armored writer: %w", err)
	}

	return nil
}

func (key *RSASigningKey) SerializeFull(out io.Writer) error {
	return key.SerializePrivateKey(out)
}

func (key *RSASigningKey) SerializePrivateKey(out io.Writer) error {
	armoredOut, err := armor.Encode(out, openpgp.PrivateKeyType, nil)
	if err != nil {
		return fmt.Errorf("unable to prepare armored writer: %w", err)
	}

	if err := key.Entity.SerializePrivate(armoredOut, nil); err != nil {
		return err
	}

	if err := armoredOut.Close(); err != nil {
		return fmt.Errorf("unable to close armored writer: %w", err)
	}

	return nil
}

func GenerateRSASigningKey() (*RSASigningKey, error) {
	entity, err := openpgp.NewEntity("trdl", "trdl server auto signer", "", &packet.Config{
		Time:          time.Now,
		Rand:          rand.Reader,
		DefaultHash:   crypto.SHA256,
		DefaultCipher: packet.CipherAES128,
		RSABits:       4096,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to generate openpgp entity: %w", err)
	}

	return &RSASigningKey{Entity: entity}, nil
}

func ParseRSASigningKey(in io.Reader) (*RSASigningKey, error) {
	el, err := openpgp.ReadArmoredKeyRing(in)
	if err != nil {
		return nil, err
	}

	if len(el) == 0 {
		return nil, fmt.Errorf("no private PGP signing key entities found")
	}

	return &RSASigningKey{Entity: el[0]}, nil
}

func SignDataStream(detachedSignatureOut io.Writer, dataStream io.Reader, key *RSASigningKey) error {
	return openpgp.DetachSign(detachedSignatureOut, key.Entity, dataStream, nil)
}
