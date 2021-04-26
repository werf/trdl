module github.com/werf/vault-plugin-secrets-trdl

go 1.16

require (
	github.com/aws/aws-sdk-go v1.30.27
	github.com/docker/docker v1.4.2-0.20200319182547-c7ad2b866182
	github.com/go-git/go-billy/v5 v5.1.0
	github.com/go-git/go-git/v5 v5.3.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210420212757-8e2932fb01f2
