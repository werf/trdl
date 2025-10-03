package mac_signing

type Credentials struct {
	Certificate  string
	Password     string
	NotaryKeyID  string
	NotaryKey    string
	NotaryIssuer string
}

const MacSigningCertificateName = "certificate"
