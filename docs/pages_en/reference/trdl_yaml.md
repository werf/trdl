---
title: trdl.yaml
permalink: reference/trdl_yaml.html
toc: false
---

The `trdl.yaml` configuration contains instructions that define the environment and command set needed to build release artifacts.

On release, trdl reads `trdl.yaml` from the Git tag and performs the build:
- Runs a container based on the selected Docker image.
- Mounts the source code of the Git tag into the `/git` directory.
- Executes build instructions in the `/git` directory.
- Saves release artifacts from the `/result` directory.

{% include reference/trdl_yaml/table.html %}

## Release artifacts layout

After completing the build instructions, the release artifacts must reside in the `/result` directory. Artifacts require a strict directory organization to integrate with the trdl client, deliver to different platforms, and efficiently handle executable files.

Each release artifact must be saved to the directory of the platform for which it is designed.
The name of the platform directory depends on the operating system and the `<os>-<arch>` parameter (system architecture).
The reserved name `any` can be used if there is no need to segregate artifacts based on OS and/or system architecture. Below is a list of supported combinations, arranged according to the trdl client preferences.

```
darwin-amd64
darwin-arm64
darwin-any
linux-amd64
linux-amd64
linux-any
windows-amd64
windows-any
any-any
```

To use the basic functions of the trdl client (e.g., the [trdl use](/documentation/reference/cli/trdl_use.html) command), you need to save the executables in the `bin` subdirectory.

As a result, for most projects, the `/result` directory after the build should have the following structure:
```
result
├── ...
└── <os>-<arch>
    ├── bin
    │   ├── ...
    │   └── <release artifact>
    ├── ...
    └── <release artifact>
```

Here:

- `os` — operating system (`darwin`, `linux`, `windows`, or `any` if the release artifacts are system-independent);
- `arch` — architecture (`amd64`, `arm64`, or `any` if the release artifacts are platform-independent);
- `release artifact` — an arbitrary file.

## Example

### trdl.yaml

{% include reference/trdl_yaml/example_trdl_yaml.md.liquid %}

### build.sh

{% include reference/trdl_yaml/example_build_sh.md.liquid %}

### Below is the structure of the /result directory after running assembly instructions

{% include reference/trdl_yaml/example_result.md.liquid %}
