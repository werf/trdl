# trdl

[![maintainability][maintainability-badge]][maintainability-link]
[![coverage][coverage-badge]][coverage-link]
[![github discussions][discussions-badge]][discussions-link]
[![coc][coc-badge]][coc-link]

[maintainability-badge]:    https://api.codeclimate.com/v1/badges/a95ed9e90acae45f40ee/maintainability
[maintainability-link]:     https://codeclimate.com/github/werf/trdl/maintainability
[coverage-badge]:           https://api.codeclimate.com/v1/badges/a95ed9e90acae45f40ee/test_coverage
[coverage-link]:            https://codeclimate.com/github/werf/trdl/test_coverage
[discussions-badge]:        https://img.shields.io/github/discussions/werf/trdl
[discussions-link]:         https://github.com/werf/trdl/discussions
[coc-badge]:                https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg
[coc-link]:                 CODE_OF_CONDUCT.md

trdl *(stands for "true delivery")* is an Open Source solution providing a secure channel for delivering updates from the Git repository to the end user.

The project team releases new versions of the software and switches them in the release channels. Git acts as the single source of truth while [Vault](https://www.vaultproject.io/) is used as a tool to verify operations as well as populate and maintain the [TUF repository](https://github.com/theupdateframework/specification). The user selects a release channel, continuously receives the latest software version from the TUF repository, and uses it.

We already use trdl as an update manager for [werf CI/CD tool](https://github.com/werf/werf).

## Architecture

trdl combines two key components: the server and the client.

**trdl-server**:
* builds and releases software versions;
* publishes the release channels *(here is an [example configuration from werf](https://github.com/werf/werf/blob/multiwerf/trdl_channels.yaml))*;
* ensures the release and the publication security via verifying the minimal number of valid GPG signatures associated with an action;
* ensures the object storage security via saving data signed by keys (no one has access to those keys) and continuously rotating TUF keys and metadata.

**trdl-client**:
* manages software repositories;
* updates software version within the selected release channel;
* provides easy operation with software version artifacts in the shell session;
* ensures safe communication via working with the TUF repository in a reliable fashion.

<img alt="Release" src="https://raw.githubusercontent.com/werf/trdl/master/docs/images/slider/release/6.svg" width="80%">

___

<img alt="Publication" src="https://raw.githubusercontent.com/werf/trdl/master/docs/images/slider/publish/7.svg" width="80%">

## Installation

### trdl-client

Download `trdl` client binaries from the [GitHub Releases page](https://github.com/werf/trdl/releases), optionally verifying the binary with the PGP signature.

## Documentation

Project's website is [now available](https://trdl.dev/) with more information (including developers quickstart) to follow soon.

## Community & support

Please feel free to reach developers/maintainers and users via [GitHub Discussions](https://github.com/werf/trdl/discussions) for any questions regarding trdl.

Your issues are processed carefully if posted to [issues at GitHub](https://github.com/werf/trdl/issues).

## License

Apache License 2.0, see [LICENSE](LICENSE).
