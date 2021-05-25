# trdl

## File structure

```shell
~/.trdl$ tree -a
.
├── config.yaml
├── .locks
│   ├── ...
│   └── <PROJECT_NAME>
├── .tmp
│   ├── ...
│   └── <PROJECT_NAME>
└── projects
    ├── ...
    └── <PROJECT_NAME>
        ├── channels
        │   ├── ...
        │   └── <GROUP_NAME>
        │       ├── ...
        │       └── <alpha | beta | ea | stable | rc>
        ├── .meta
        └── releases
            ├── ...
            └── <RELEASE_VERSION>
                └── <OS>_<ARCH>
                    └── ...
```

```shell
~/.trdl$ cat config.yaml
projects:
  ...
  - name: <PROJECT_NAME>
    repourl: <REPO_URL>
```
