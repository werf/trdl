# Local development environment

This repository provides a Taskfile for setting up a local development environment with MinIO and Vault.

## Prerequisites

- Install [Task](https://taskfile.dev/installation/)
  - Before using Taskfile, set the environment variable:
    ```shell
    export TASK_X_REMOTE_TASKFILES=1
    ```
    (Add this to your shell configuration file, e.g., `.bashrc` or `.zshrc`, for persistence.)
  - To skip confirmation prompts when running tasks, use the `--yes` flag:
    ```shell
    task --yes taskname
    ```

## Default setup

To start the local development environment with default settings, run:

```sh
task setup:dev-environment
```

This command will:
1. Start a MinIO server with a default bucket
2. Start a Vault server in development mode
3. Configure the Vault plugin with default parameters

### Configuration parameters

| Parameter                | Default Value                               | Description                                  |
| ------------------------ | ------------------------------------------- | -------------------------------------------- |
| `project_name`           | `trdl-test-project1`                        | The name of the project/bucket in MinIO      |
| `signatures_count`       | `0`                                         | Number of required verified signatures       |
| `git_repo_url`           | `https://github.com/werf/trdl-test-project` | Git repository URL for TRDL metadata         |
| `git_trdl_path`          | `p1/trdl.yaml`                              | Path to TRDL configuration in the repository |
| `git_trdl_channels_path` | `p1/trdl_channels.yaml`                     | Path to TRDL channels configuration          |
| `vault_addr`             | `http://localhost:8200`                     | Vault server address                         |
| `minio_data_dir`         | `.minio_data`                               | Local directory for MinIO data               |
| `minio_container`        | `trdl_dev_minio`                            | MinIO container name                         |
| `vault_container`        | `trdl_dev_vault`                            | Vault container name                         |
| `s3_access_key`          | `minioadmin`                                | MinIO access key                             |
| `s3_secret_key`          | `minioadmin`                                | MinIO secret key                             |
| `minio_port`             | `9000`                                      | MinIO service port                           |
| `minio_console_port`     | `9001`                                      | MinIO console port                           |
| `vault_port`             | `8200`                                      | Vault service port                           |
| `version`                | `dev`                                       | Version tag for builds                       |

## Custom setup

You can override default values by passing them as variables:

```sh
task setup:dev-environment --project_name=my-custom-project --signatures_count=2 --git_repo_url=https://github.com/example/repo --git_trdl_path=trdl.yaml --git_trdl_channels_path=trdl_channels.yaml
```

This will:
- Use `my-custom-project` as the bucket/project name
- Require `2` verified signatures
- Set custom Git repository URL and paths

## Cleaning up

To remove all containers and clean up data:

```sh
task dev:cleanup
```