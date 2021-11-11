---
title: trdl.yaml
permalink: reference/trdl_yaml.html
toc: false
---

The `trdl.yaml` configuration contains instructions that define the environment and command set needed to build release artifacts.

On release, trdl reads `trdl.yaml` from the git tag and performs the build:

- Runs a container based on the selected docker image.
- Mounts the source code of the git tag into the `/git` directory.
- Executes build instructions in the `/git` directory.
- Saves release artifacts from the `/result` directory.

{% include reference/trdl_yaml/table.html %}

## Example

```yaml
dockerImage: golang:1.17-alpine@sha256:13919fb9091f6667cb375d5fdf016ecd6d3a5d5995603000d422b04583de4ef9
commands:
  - ./scripts/build.sh {{ .Tag }} 
  - cp -a release/* /result
```