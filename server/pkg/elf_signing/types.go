package elf_signing

type VaultSignerOpts struct {
	Address      string
	TransitPath  string
	AuthPath     string
	AuthRoleID   string
	AuthSecretID string
}
type SignerSettings struct {
	KeyRef           string
	KeyPassword      string
	CertRef          string
	IntermediatesRef string
	VaultOpts        VaultSignerOpts
}
