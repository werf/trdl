<p align="center">
  <img src="https://trdl.dev/images/logo.svg" alt="trdl" style="max-height:100%;" height="30">
</p>
___

This repository provides actions for seamless integration of trdl into your GitHub Workflows.

## Table of contents

- [Table of contents](#table-of-contents)
- [Workflows](#workflows)
  - [Set up your application executable files with `werf/trdl/actions/setup-app` action](#set-up-your-application-executable-files-with-werftrdlactionssetup-app-action)
    - [Easy set up via presets](#easy-set-up-via-presets)
      - [werf](#werf)
      - [nelm](#nelm)
      - [kubedog](#kubedog)
    - [Set up with preset group and channel](#set-up-with-preset-group-and-channel)
    - [Manual set up](#manual-set-up)
  - [Installation of `trdl` with `werf/trdl/actions/install` action](#installation-of-trdl-with-werftrdlactionsinstall-action)
- [License](#license)

## Workflows

### Set up your application executable files with `werf/trdl/actions/setup-app` action

#### Easy set up via presets

##### werf

```yaml
- name: Setup werf
  uses: werf/trdl/actions/setup-app@main
  with:
    preset: werf

- name: Use werf binary
  run: werf version
```

##### nelm

```yaml
- name: Setup nelm
  uses: werf/trdl/actions/setup-app@main
  with:
    preset: nelm

- name: Use nelm binary
  run: nelm version
```

##### kubedog

```yaml
- name: Setup kubedog
  uses: werf/trdl/actions/setup-app@main
  with:
    preset: kubedog

- name: Use kubedog binary
  run: kubedog version
```

#### Set up with preset group and channel

```yaml
- name: Setup werf
  uses: werf/trdl/actions/setup-app@main
  with:
    preset: werf
    group: 2
    channel: alpha

- name: Use werf binary
  run: werf version
```

#### Manual set up

```yaml
- name: Setup example application
  uses: werf/trdl/actions/setup-app@main
  with:
    repo: app
    url: https://s3.example.com
    root-version: 12
    root-sha512: e1d3c7bcfdf473fe1466c5e9d9030bea0fed857d0563db1407754d2795256e4d063b099156807346cdcdc21d747326cc43f96fa2cacda5f1c67c8349fe09894d
    group: 2
    channel: stable

- name: Use application binaries
  run: app version
```

### Installation of `trdl` with `werf/trdl/actions/install` action

```yaml
- name: Install trdl
  uses: werf/trdl/actions/install@main

- name: Use trdl binary
  run: |
    . $(trdl add app https://s3.example.com 12 e1d3c7bcfdf473fe1466c5e9d9030bea0fed857d0563db1407754d2795256e4d063b099156807346cdcdc21d747326cc43f96fa2cacda5f1c67c8349fe09894d)
    . $(trdl use app 2 stable)

    app version
```

## License

Apache License 2.0, see [LICENSE](LICENSE)
