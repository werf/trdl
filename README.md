# trdl

[![maintainability][maintainability-badge]][maintainability-link]
[![coverage][coverage-badge]][coverage-link]
[![github discussions][discussions-badge]][discussions-link]
[![coc][coc-badge]][coc-link]

[maintainability-badge]:    https://qlty.sh/gh/werf/projects/trdl/maintainability.svg
[maintainability-link]:     https://qlty.sh/gh/werf/projects/trdl
[coverage-badge]:           https://qlty.sh/gh/werf/projects/trdl/coverage.svg
[coverage-link]:            https://qlty.sh/gh/werf/projects/trdl
[discussions-badge]:        https://img.shields.io/github/discussions/werf/trdl
[discussions-link]:         https://github.com/werf/trdl/discussions
[coc-badge]:                https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg
[coc-link]:                 CODE_OF_CONDUCT.md

trdl *(stands for "true delivery")* is an Open Source solution providing a secure channel for delivering updates from the Git repository to the end user.

The project team releases new versions of the software and switches them in the release channels. Git acts as the single source of truth while [Vault](https://www.vaultproject.io/) is used as a tool to verify operations as well as populate and maintain the [TUF repository](https://github.com/theupdateframework/specification). The user selects a release channel, continuously receives the latest software version from the TUF repository, and uses it.

<p align="center">
  <img alt="Scheme" src="https://raw.githubusercontent.com/werf/trdl/main/docs/images/intro-scheme_en.svg" width="50%">
</p>

We have been successfully using trdl to continuously deliver our [werf CI/CD tool](https://github.com/werf/werf) to CI runners and user hosts.

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

## How it works

### Releasing

<p align="center">
  <img alt="Release" src="https://raw.githubusercontent.com/werf/trdl/master/docs/images/slider/release/6.svg" width="80%">
</p>
  
### Publishing the channels

<p align="center">
  <img alt="Publication" src="https://raw.githubusercontent.com/werf/trdl/master/docs/images/slider/publish/7.svg" width="80%">
</p>

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
