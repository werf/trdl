# Changelog

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
