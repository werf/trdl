module github.com/werf/trdl

go 1.16

require (
	bou.ke/monkey v1.0.2
	github.com/Masterminds/goutils v1.1.1
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/gookit/color v1.4.2
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/prashantv/gostub v1.0.0
	github.com/rodaine/table v1.0.1
	github.com/spf13/cobra v1.1.3
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
	github.com/werf/lockgate v0.0.0-20210423043214-fd4df31c9ab0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210728151427-8674be250fb1
