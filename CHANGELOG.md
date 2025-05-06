# Changelog

## [0.11.0](https://www.github.com/werf/trdl/compare/v0.10.0...v0.11.0) (2025-05-05)


### Features

* **actions:** add install, setup-app actions
* **client, add:** make root version and sha parameters optional ([ba6f254](https://www.github.com/werf/trdl/commit/ba6f25424599c4a3df4cb0f98f04639bf69b60f1))

## [0.10.0](https://www.github.com/werf/trdl/compare/v0.9.0...v0.10.0) (2025-04-14)


### Features

* **server:** add configure/last_published_git_commit endpoint to unlock  publication when Git history rebased ([838aa5b](https://www.github.com/werf/trdl/commit/838aa5b5fb094bd426fb7615434303bf58adda05))

## [0.9.0](https://www.github.com/werf/trdl/compare/v0.8.7...v0.9.0) (2025-04-04)


### Features

* **client:** add debug struct logging ([#304](https://www.github.com/werf/trdl/issues/304)) ([239b401](https://www.github.com/werf/trdl/commit/239b40130aadd3be28916435e4082bec33dc559c))
* **client:** support more envs for `trdl update` ([5f76cb5](https://www.github.com/werf/trdl/commit/5f76cb5646366798a0eace6cd547e32a30637536))
* **release:** add trdl-vault cli for release and publish ([#301](https://www.github.com/werf/trdl/issues/301)) ([5c70b51](https://www.github.com/werf/trdl/commit/5c70b51995cdb848a365f7458a92303e7a403b7e))
* **server:** add a one-time docker builder for build tasks ([#320](https://www.github.com/werf/trdl/issues/320)) ([a055240](https://www.github.com/werf/trdl/commit/a0552406be0073b7cbc03acb7709d7e66b7ad28b))
* **server:** add build log ([#316](https://www.github.com/werf/trdl/issues/316)) ([aee04d8](https://www.github.com/werf/trdl/commit/aee04d82bfc8c7764af00661fff250a6044ba73a))

### [0.8.7](https://www.github.com/werf/trdl/compare/v0.8.6...v0.8.7) (2025-03-04)


### Bug Fixes

* **server:** resolve lock in docker build ([#312](https://www.github.com/werf/trdl/issues/312)) ([c0aba74](https://www.github.com/werf/trdl/commit/c0aba74749f3e43e82169b1f9c213c4ca8640d22))

### [0.8.6](https://www.github.com/werf/trdl/compare/v0.8.5...v0.8.6) (2025-03-03)


### Bug Fixes

* **server:** fix artifact export and build commands leak ([3e2e854](https://www.github.com/werf/trdl/commit/3e2e854ed35a8349bbe8947da2a09cc670f8ae27))

### [0.8.5](https://www.github.com/werf/trdl/compare/v0.8.4...v0.8.5) (2025-02-27)


### Bug Fixes

* **server**: mask sensetive data in debug output ([eef4e2c](https://www.github.com/werf/trdl/commit/eef4e2c61c37045f4fc77be38bf3d4033177eb05))

### [0.8.4](https://www.github.com/werf/trdl/compare/v0.8.3...v0.8.4) (2025-02-26)


### Bug Fixes

* **server**: fix secrets create error ([#305](https://www.github.com/werf/trdl/issues/305)) ([e0ca0de](https://www.github.com/werf/trdl/commit/e0ca0debf3a8f49dd696b550a18e6861f2769704))

### [0.8.3](https://www.github.com/werf/trdl/compare/v0.8.2...v0.8.3) (2025-01-17)


### Miscellaneous Chores

* **release:** fix failed release build ([5697c4d](https://www.github.com/werf/trdl/commit/5697c4d10702145920b9c75690763e06f8df8f33))

### [0.8.2](https://www.github.com/werf/trdl/compare/v0.8.1...v0.8.2) (2025-01-17)


### Miscellaneous Chores

* **server, release:** fix failed release build ([37b6335](https://www.github.com/werf/trdl/commit/37b6335d80e7ca023e8c69f4d5ae21cc9a2a62a7))

### [0.8.1](https://www.github.com/werf/trdl/compare/v0.8.0...v0.8.1) (2025-01-17)


### Miscellaneous Chores

* **server, release:** fix failed release build ([fad75ae](https://www.github.com/werf/trdl/commit/fad75aeae211a2ae1c00885ccbedcb409dee3ba3))

## [0.8.0](https://www.github.com/werf/trdl/compare/v0.7.0...v0.8.0) (2025-01-17)


### Features

* **server:** support build secrets ([#289](https://www.github.com/werf/trdl/issues/289)) ([920c879](https://www.github.com/werf/trdl/commit/920c8791964d9bd7e3072b3fbb613650e54f3666))
* **server:** validate trusted pgp public keys ([#291](https://www.github.com/werf/trdl/issues/291)) ([0293aed](https://www.github.com/werf/trdl/commit/0293aed5281a861856167a2a13bec72013243c8c))


### Bug Fixes

* **client, use:** format repository name for valid environment variable name ([eacedbb](https://www.github.com/werf/trdl/commit/eacedbbf3664cb108da9263c25416252f56056be))
* **server, quorum:** fix case when using annotated service tags ([#286](https://www.github.com/werf/trdl/issues/286)) ([1847b79](https://www.github.com/werf/trdl/commit/1847b79ca69e46c447d0ba76b69c26bfba82aea7))

## [0.7.0](https://www.github.com/werf/trdl/compare/v0.6.5...v0.7.0) (2023-09-13)


### Features

* support dev channel ([fd33299](https://www.github.com/werf/trdl/commit/fd33299ce34a05cb8cfb507e2b8e755529c27bef))

### [0.6.5](https://www.github.com/werf/trdl/compare/v0.6.4...v0.6.5) (2023-03-17)


### Bug Fixes

* **deps:** update logboek and lockgate ([5183b51](https://www.github.com/werf/trdl/commit/5183b516c80607cd2c2a38f2f3c945af5bc719ed))

### [0.6.4](https://www.github.com/werf/trdl/compare/v0.6.3...v0.6.4) (2023-03-17)


### Bug Fixes

* update all dependencies ([777331d](https://www.github.com/werf/trdl/commit/777331db23b5513729ceecd457918d00beac30a1))

### [0.6.3](https://www.github.com/werf/trdl/compare/v0.6.1...v0.6.3) (2022-09-20)


### Build System

* **server:** fix module requires Go 1.18 ([bac9399](https://www.github.com/werf/trdl/commit/bac9399b51cd7abe1a169eb3ef7a12beae7394e1))


### Miscellaneous Chores

* release v0.6.3 ([d44a47c](https://www.github.com/werf/trdl/commit/d44a47cc170862ddef1189d2aa61dd4bb397a46b))

### [0.6.1](https://www.github.com/werf/trdl/compare/v0.6.0...v0.6.1) (2022-09-20)


### Build System

* **client:** fix module requires Go 1.18 ([2a779b3](https://www.github.com/werf/trdl/commit/2a779b3c68d98f6fbe42c4632e7791f8841b430f))

## [0.6.0](https://www.github.com/werf/trdl/compare/v0.5.0...v0.6.0) (2022-09-20)


### Features

* **server:** prolong TUF roles expiration periodically ([7df2cae](https://www.github.com/werf/trdl/commit/7df2cae4004402dfa52eda5b87e2d55329661ac3))
* **server:** update go-tuf library to upstream  
* **client:** update go-tuf library to upstream 

## [0.5.0](https://www.github.com/werf/trdl/compare/v0.4.2...v0.5.0) (2022-06-21)


### Features

* **client:** export environment variable with a used group and channel in the source script ([3c083a7](https://www.github.com/werf/trdl/commit/3c083a7401ef5c1281d4a1e30de0d7050077eb74))

### [0.4.2](https://www.github.com/werf/trdl/compare/v0.4.1...v0.4.2) (2022-05-05)


### Bug Fixes

* **server:** panic when checking signatures ([8956c78](https://www.github.com/werf/trdl/commit/8956c788a39de7f4ae5cf763ef3eaf4d0ce14050))
* **server:** fix signatures ignored when too many git notes exists for a repo ([6c50b6b](https://www.github.com/werf/trdl/commit/6c50b6bd24cd1aaf91a618233c252c05eb21022f))

### [0.4.1](https://www.github.com/werf/trdl/compare/v0.4.0...v0.4.1) (2022-03-03)


### Bug Fixes

* **client:** autoclean ignores releases downloaded by previous trdl versions ([6e7b770](https://www.github.com/werf/trdl/commit/6e7b770b798385e5e15e396ed5da76f8161f6d1a))

## [0.4.0](https://www.github.com/werf/trdl/compare/v0.3.6...v0.4.0) (2022-03-03)


### Features

* **client:** autoclean old downloaded releases ([a2ea508](https://www.github.com/werf/trdl/commit/a2ea508168e87c55f626f13083efcc9f808c7b77))

### [0.3.6](https://www.github.com/werf/trdl/compare/v0.3.5...v0.3.6) (2022-02-09)


### Bug Fixes

* **client:** rebuild previous release, no changes ([59d3822](https://www.github.com/werf/trdl/commit/59d38224f1fde3452254a72070ee7d03ffd7120f))

### [0.3.5](https://www.github.com/werf/trdl/compare/v0.3.4...v0.3.5) (2022-02-09)


### Bug Fixes

* **client:** rebuild previous release, no changes ([ec3dbe4](https://www.github.com/werf/trdl/commit/ec3dbe406ac66f78ada7c6e9d9a6a9ee66ac9c9e))

### [0.3.4](https://www.github.com/werf/trdl/compare/v0.3.3...v0.3.4) (2022-02-09)


### Bug Fixes

* **client:** unknown flag --home-dir ([fcd8e49](https://www.github.com/werf/trdl/commit/fcd8e4949f6f46bce91f57aeae74e85786bc32df))

### [0.3.3](https://www.github.com/werf/trdl/compare/v0.3.2...v0.3.3) (2022-02-04)


### Bug Fixes

* **server:** absence of signatures in git notes not handled properly ([2bea1f4](https://www.github.com/werf/trdl/commit/2bea1f4fcdc163a212052bbf8fa419272d4d6e1d))
* **server:** requiredNumberOfVerifiedSignatures set to 0 incorrectly processed when signatures are present ([02e22de](https://www.github.com/werf/trdl/commit/02e22de923d439c87057c211adfbd49d5959435f))

### [0.3.2](https://www.github.com/werf/trdl/compare/v0.3.1...v0.3.2) (2021-12-26)

### [0.3.1](https://www.github.com/werf/trdl/compare/v0.3.0...v0.3.1) (2021-11-19)


### Bug Fixes

* **client:** unknown command "version" for "trdl" ([702a7f0](https://www.github.com/werf/trdl/commit/702a7f091b5ab8a24ab7c9e1c2ad43a8675fe80d))

## [0.3.0](https://www.github.com/werf/trdl/compare/v0.2.1...v0.3.0) (2021-11-19)


### Features

* **client:** add remove command ([dd32e1a](https://www.github.com/werf/trdl/commit/dd32e1aebfb2363688a7f0fa463813f583221c46))
* **server:** support custom trdl.yaml and trdl_channels.yaml files ([aa5a986](https://www.github.com/werf/trdl/commit/aa5a9860757e2036467af8b958b71ca597d4e134))

### [0.2.1](https://www.github.com/werf/trdl/compare/v0.2.0...v0.2.1) (2021-11-01)


### Bug Fixes

* **server:** fix error in the periodic task when pgp key not generated ([7a29a8e](https://www.github.com/werf/trdl/commit/7a29a8e2f1fa74e308f11c4922790b07ac052b2d))

## [0.2.0](https://www.github.com/werf/trdl/compare/v0.1.7...v0.2.0) (2021-10-28)


### Features

* **server:** auto sign all release targets ([c6221a8](https://www.github.com/werf/trdl/commit/c6221a8aa1cffef3d26049a819972cb680123c32))
* **server:** store trdl-channels.yaml in the default branch by default ([e45ba9b](https://www.github.com/werf/trdl/commit/e45ba9b5a474b1c8b104a3e160bddcf355b3d508))


### Bug Fixes

* **server:** allow custom log file path with VAULT_PLUGIN_SECRETS_TRDL_LOG_FILE=<path> variable ([587697a](https://www.github.com/werf/trdl/commit/587697a174071f289e17713711f014abfbeb1ef9))
* **server:** use default backend logger, stream logs to the vault server ([f787e23](https://www.github.com/werf/trdl/commit/f787e230b516ccd58ea59ef6fbbb5e96c4292831))

### [0.1.7](https://www.github.com/werf/trdl/compare/v0.1.6...v0.1.7) (2021-09-20)


### Bug Fixes

* trigger release ([7d4e1f4](https://www.github.com/werf/trdl/commit/7d4e1f478489b4dbb1fdf5a628996146ad42e654))

### [0.1.6](https://www.github.com/werf/trdl/compare/v0.1.5...v0.1.6) (2021-09-16)


### Bug Fixes

* correction release ([d06a049](https://www.github.com/werf/trdl/commit/d06a049ce2e41cd5ff012d0855cf5442e754043a))

### [0.1.5](https://www.github.com/werf/trdl/compare/v0.1.4...v0.1.5) (2021-09-14)


### Bug Fixes

* **server:** optimize data stream processing, use buffered streams ([393f9b0](https://www.github.com/werf/trdl/commit/393f9b0bfa97cb9bbd2d55587b7b85ec3aaeffd7))

### [0.1.4](https://www.github.com/werf/trdl/compare/v0.1.3...v0.1.4) (2021-09-10)


### Bug Fixes

* **client:** prevent using self-update repository as repo ([1360ae9](https://www.github.com/werf/trdl/commit/1360ae9f4196197b31c6eedcd178d728a0f3a7e3))
* **server:** publish procedure validation ([9ec466b](https://www.github.com/werf/trdl/commit/9ec466b58772a5ae7153f2a8426fd2c5f75acde9))
* **server:** user's release commands are launched in separate RUN instructions ([e1d8468](https://www.github.com/werf/trdl/commit/e1d8468bcda69065758b0ac2611c2bd2037bd626))

### [0.1.3](https://www.github.com/werf/trdl/compare/v0.1.2...v0.1.3) (2021-09-03)


### Bug Fixes

* **client:** trdl release failed ([d0dd5c3](https://www.github.com/werf/trdl/commit/d0dd5c34f7dbe3deb716e532cf1ed7a8845b5ad1))

### [0.1.2](https://www.github.com/werf/trdl/compare/v0.1.1...v0.1.2) (2021-09-03)


### Bug Fixes

* **client:** the use command does not work properly in powershell ([4f0ba99](https://www.github.com/werf/trdl/commit/4f0ba9973bb48ea56ade8acca6a83bc44b680f70))
* **client:** the use command must strictly activate local version ([885880a](https://www.github.com/werf/trdl/commit/885880ab95eb3b3f94b6a90a7beaec99cb0bfb0c))
* new binpath used for background use update in sh ([8c8fb4c](https://www.github.com/werf/trdl/commit/8c8fb4c2f12e498e0d8c134071dc953a027d95d6))
* **server:** the path separator in the repository file names should not be system dependent ([346c851](https://www.github.com/werf/trdl/commit/346c85170e84f013ca622c08091a4a02c122ea31))
* **server:** unable to remove service docker image ([1e9428b](https://www.github.com/werf/trdl/commit/1e9428b2a06346aaadbb7531259645383fec3d93))
