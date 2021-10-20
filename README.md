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

trdl *(stands for "true delivery")* is an Open Source tool for implementing automatic updates. It is a universal package manager delivering new versions of your application from a trusted [TUF repository](https://github.com/theupdateframework/specification). Your application might be distributed in any form of programming code, e.g., a binary file, a shell script, and even an Ansible playbook.

trdl is already used as an update manager for [werf CI/CD tool](https://github.com/werf/werf).

# Architecture

trdl combines two key components: the server and the client.

**trdl-server**:
* builds and publishes software releases;
* monitors for consistency between release channels and releases *(here is an [example from werf](https://github.com/werf/werf/blob/multiwerf/trdl_channels.yaml))*;
* ensures repo security via saving data signed by keys to the TUF repository (no one has access to those keys) and continuously rotating keys and metadata.

**trdl-client**:
* processes application files within the release channels;
* processes files in the TUF repository in a reliable fashion.

# Documentation

Project's website is [now available](https://trdl.dev/) with more information (including developers quickstart) to follow soon.

# Community & support

Please feel free to reach developers/maintainers and users via [GitHub Discussions](https://github.com/werf/trdl/discussions) for any questions regarding trdl.

Your issues are processed carefully if posted to [issues at GitHub](https://github.com/werf/trdl/issues).

# License

Apache License 2.0, see [LICENSE](LICENSE).
