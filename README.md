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
        │   └── <GROUP_NAME>
        │       ├── ...
        │       └── <alpha | beta | ea | stable | rock-solid>
        ├── .meta
        └── releases
            ├── ...
            └── <RELEASE_VERSION>
                └── <OS>_<ARCH>
                    └── ...
```

```shell
~/.trdl$ cat config.yaml
repositories:
  ...
  - name: <REPO>
    url: <URL>
```
