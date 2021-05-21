module github.com/werf/trdl

go 1.16

require (
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/rodaine/table v1.0.1
	github.com/spf13/cobra v1.1.3
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
	github.com/werf/lockgate v0.0.0-20210423043214-fd4df31c9ab0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210521115409-7cfe1d24222d