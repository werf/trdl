package elf_signing

type VaultSignerOpts struct {
	Address      string `json:"address"`
	TransitPath  string `json:"transit_path"`
	AuthPath     string `json:"auth_path"`
	AuthRoleID   string `json:"auth_role_id"`
	AuthSecretID string `json:"auth_secret_id"`
}
type SignerSettings struct {
	KeyRef           string          `json:"key_ref"`
	KeyPassword      string          `json:"key_password"`
	CertRef          string          `json:"cert_ref"`
	IntermediatesRef string          `json:"intermediates_ref"`
	VaultOpts        VaultSignerOpts `json:"vault_opts"`
}
