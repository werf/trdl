---
title: TUF repository layout
permalink: reference/tuf_repository_layout.html
---


Please, refer to the [TUF documentation](https://theupdateframework.github.io/specification/latest/#the-repository) to learn more about the TUF repository, its purpose, and the standard set of files. This article will focus on the [_Target files_](https://theupdateframework.github.io/specification/latest/#target-files) organization, how the release is stored, GPG signatures of the release artifacts, and release channels. 

```
targets
├── channels/
├── releases/
└── signatures/
```

## Storing the release

### Storing release artifacts

When releasing, trdl uses the path corresponding to the release version `targets/releases/<semver>/` and saves the build result unchanged.

```
targets
└── releases
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>
```

**where:**

- semver — release version in the [semver](https://semver.org/) format;
- os — operating system (`darwin`, `linux`, `windows`, or `any`, if the release artifacts are system-independent);
- arch — architecture (`amd64`, `arm64`, or `any`, if the release artifacts are platform-independent);
- release artifact — an arbitrary file. 

#### Example

````
targets
└── releases
    ├── ...
    └── 1.2.20
        ├── darwin-amd64
        │   └── bin
        │       └── werf
        ├── darwin-arm64
        │   └── bin
        │       └── werf
        ├── linux-amd64
        │   └── bin
        │       └── werf
        ├── linux-amd64
        │   └── bin
        │       └── werf
        └── windows-amd64
            └── bin
                └── werf.exe
````

### Storing GPG signatures of the release artifacts

When releasing, trdl signs all release artifacts (`targets/releases/<semver>/<os>-<arch>/<release artifact>`) and saves all signatures in `targets/signatures/` to the same path but with the `.sig` extension (`targets/signatures/<semver>/<os>-<arch>/<release artifact>.sig`).

```
targets
└── signatures
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>.sig
```

**where:**

- semver — release version in the [semver](https://semver.org/) format;
- os — operating system (`darwin`, `linux`, `windows`, or `any`, if the release artifacts are system-independent);
- arch — architecture (`amd64`, `arm64`, or `any`, if the release artifacts are platform-independent);
- release artifact — an arbitrary file.

#### Example

````
targets
└── signatures
    ├── ...
    └── 1.2.20
        ├── darwin-amd64
        │   └── bin
        │       └── werf.sig
        ├── darwin-arm64
        │   └── bin
        │       └── werf.sig
        ├── linux-amd64
        │   └── bin
        │       └── werf.sig
        ├── linux-amd64
        │   └── bin
        │       └── werf.sig
        └── windows-amd64
            └── bin
                └── werf.exe.sig
````

## Storing release channels

When publishing, trdl stores release channels according to the trdl_channels.yaml configuration file.

```
targets
└── channels
    ├── ...
    └── <semver part>
        ├── ...
        └── <channel>
```

**where:**

- semver part — the [semver](https://semver.org/) part;
- channel — `alpha`, `beta`, `ea`, `stable`, or `rock-solid` release channel. 

### Example

````
targets
└── channels
    ├── ...
    └── 1.2
        ├── alpha
        ├── beta
        ├── ea
        ├── stable
        └── rock-solid
````
