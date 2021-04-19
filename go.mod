module github.com/werf/vault-plugin-secrets-trdl

go 1.16

require (
	github.com/aws/aws-sdk-go v1.30.27 // indirect
	github.com/hashicorp/go-hclog v0.14.1 // indirect
	github.com/hashicorp/vault/api v1.1.0 // indirect
	github.com/hashicorp/vault/sdk v0.2.0 // indirect
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55 // indirect
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210420212757-8e2932fb01f2
