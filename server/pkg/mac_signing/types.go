package mac_signing

type MacSigningCredentials struct {
	Name         string
	Certificate  string
	Password     string
	NotaryKeyID  string
	NotaryKey    string
	NotaryIssuer string
}
