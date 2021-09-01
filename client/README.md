# trdl

## File structure

```shell
~/.trdl$ tree -a
.
├── config.yaml
├── .locks
│   ├── ...
│   └── repositories
│       ├── ...
│       └── <REPO>
├── logs
│   ├── ...
│   └── repositories
│       ├── ...
│       └── <REPO>
├── .tmp
│   ├── ...
│   └── repositories
│       ├── ...
│       └── <REPO>
└── repositories
    ├── ...
    └── <REPO>
        ├── channels
        │   ├── ...
        │   └── <GROUP>
        │       ├── ...
        │       └── <CHANNEL>
        ├── .meta
        ├── releases
        │   ├── ...
        │   └── <RELEASE_VERSION>
        │       └── <OS>_<ARCH>
        │           └── ...
        └── scripts
            ├── ...
            └── <GROUP>-<CHANNEL>
                ├── ...
                └── source_script[.<ext>]
```

```shell
~/.trdl$ cat config.yaml
repositories:
  ...
  - name: <REPO>
    url: <URL>
    defaultChannel: <CHANNEL>
```
