package pgp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPGPSigningKey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PGP signing key suite")
}

var _ = Describe("PGP signing key", func() {
	BeforeEach(func() {
		rand.Seed(time.Now().Unix())
	})

	It("Should create detached signature of data stream then decode signature using GPG tool", func() {
		key, err := GenerateRSASigningKey()
		Expect(err).NotTo(HaveOccurred())

		fileContent := make([]byte, rand.Uint32()%104857600)
		rand.Read(fileContent)

		pgpSignBuf := bytes.NewBuffer(nil)

		err = SignDataStream(pgpSignBuf, bytes.NewReader(fileContent), key)
		Expect(err).NotTo(HaveOccurred())

		pgpSign := pgpSignBuf.Bytes()

		Expect(len(pgpSignBuf.String()) > 0).To(BeTrue())

		tmpFile, err := ioutil.TempFile("", "pgp-signing-key-test-file-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpFile.Name())
		_, err = io.Copy(tmpFile, bytes.NewReader(fileContent))
		Expect(err).NotTo(HaveOccurred())
		err = tmpFile.Close()
		Expect(err).NotTo(HaveOccurred())

		tmpSigFile, err := ioutil.TempFile("", "pgp-signing-key-test-file-*.sig")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpSigFile.Name())
		_, err = io.Copy(tmpSigFile, bytes.NewReader(pgpSign))
		Expect(err).NotTo(HaveOccurred())
		err = tmpSigFile.Close()
		Expect(err).NotTo(HaveOccurred())

		tmpPubkeyFile, err := ioutil.TempFile("", "pgp-signing-key-test-pubkey-*.gpg")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpPubkeyFile.Name())
		pubkeyData := bytes.NewBuffer(nil)
		err = key.SerializePublicKey(pubkeyData)
		Expect(err).NotTo(HaveOccurred())
		_, err = io.Copy(tmpPubkeyFile, bytes.NewReader(pubkeyData.Bytes()))
		Expect(err).NotTo(HaveOccurred())
		err = tmpPubkeyFile.Close()
		Expect(err).NotTo(HaveOccurred())

		fmt.Printf("Importing gpg key:\n%s\n", pubkeyData)

		cmd := exec.Command("gpg", "--import", tmpPubkeyFile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("gpg", "--verify", tmpSigFile.Name(), tmpFile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should serialize and deserialize PGP signing key into text", func() {
		key, err := GenerateRSASigningKey()
		Expect(err).NotTo(HaveOccurred())

		data := bytes.NewBuffer(nil)
		err = key.SerializeFull(data)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Serialized key:\n%s\n", data.String())

		newKey, err := ParseRSASigningKey(bytes.NewReader(data.Bytes()))
		Expect(err).NotTo(HaveOccurred())

		Expect(key.Entity.PrimaryKey.KeyId).To(Equal(newKey.Entity.PrimaryKey.KeyId))
		Expect(key.Entity.PrimaryKey.Fingerprint).To(Equal(newKey.Entity.PrimaryKey.Fingerprint))

		Expect(key.Entity.PrivateKey.KeyId).To(Equal(newKey.Entity.PrivateKey.KeyId))
		Expect(key.Entity.PrivateKey.Fingerprint).To(Equal(newKey.Entity.PrimaryKey.Fingerprint))

		for k, ident := range key.Entity.Identities {
			newIdent := newKey.Entity.Identities[k]
			Expect(ident.UserId).To(Equal(newIdent.UserId))
			Expect(ident.Name).To(Equal(newIdent.Name))
		}
	})
})
