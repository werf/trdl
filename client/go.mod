module github.com/werf/trdl/client

go 1.18

require (
	bou.ke/monkey v1.0.2
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/gookit/color v1.4.2
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/rodaine/table v1.0.1
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
	github.com/werf/lockgate v0.0.0-20210423043214-fd4df31c9ab0
	github.com/werf/logboek v0.5.4
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	mvdan.cc/xurls v1.1.0
)

require (
	github.com/avelino/slugify v0.0.0-20180501145920-855f152bd774 // indirect
	github.com/gofrs/flock v0.7.1 // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mvdan/xurls v1.1.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975 // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9 // indirect
	golang.org/x/sys v0.0.0-20210330210617-4fbd30eecc44 // indirect
	golang.org/x/text v0.3.2 // indirect
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210728151427-8674be250fb1
