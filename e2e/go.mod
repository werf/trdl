module github.com/werf/trdl/e2e

go 1.15

require (
	github.com/Masterminds/goutils v1.1.1
	github.com/hashicorp/go-hclog v0.16.1 // indirect
	github.com/hashicorp/vault/sdk v0.2.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/otiai10/copy v1.6.0
	github.com/prashantv/gostub v1.0.0
	github.com/werf/trdl/client v0.0.0
	github.com/werf/trdl/server v0.0.0
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210728151427-8674be250fb1
	github.com/werf/trdl/client v0.0.0 => ../client
	github.com/werf/trdl/server v0.0.0 => ../server
)
