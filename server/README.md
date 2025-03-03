# Local development environment

This repository provides a Makefile for setting up a local development environment with MinIO and Vault.

## Default setup

To start the local development environment with default settings, simply run:

```sh
make .run
```

This command will:

1. Start a MinIO server with a default bucket.
2. Start a Vault server in development mode.
3. Configure the Vault plugin with default parameters.

### Default parameters

| Parameter                | Default Value                               | Description                                   |
| ------------------------ | ------------------------------------------- | --------------------------------------------- |
| `PROJECT_NAME`           | `trdl-test-project1`                        | The name of the project/bucket in MinIO.      |
| `SIGNATURES_COUNT`       | `0`                                         | Number of required verified signatures.       |
| `GIT_REPO_URL`           | `https://github.com/werf/trdl-test-project` | Git repository URL for TRDL metadata.         |
| `GIT_TRDL_PATH`          | `p1/trdl.yaml`                              | Path to TRDL configuration in the repository. |
| `GIT_TRDL_CHANNELS_PATH` | `p1/trdl_channels.yaml`                     | Path to TRDL channels configuration.          |

## Custom Setup

You can override default values by passing them as arguments:

```sh
make .run PROJECT_NAME=my-custom-project SIGNATURES_COUNT=2 GIT_REPO_URL=https://github.com/example/repo GIT_TRDL_PATH=trdl.yaml GIT_TRDL_CHANNELS_PATH=trdl_channels.yaml
```

This will:

- Use `my-custom-project` as the bucket/project name.
- Require `2` verified signatures.
- Set `GIT_REPO_URL` to `https://github.com/example/repo`.
- Set `GIT_TRDL_PATH` to the path to `trdl.yaml` in your `GIT_REPO_URL`
- Set `GIT_TRDL_CHANNELS_PATH` to the path to `trdl_channels.yaml` in your `GIT_REPO_URL`

## Cleaning Up

To remove all containers and clean up data, run:

```sh
make clean
```
