#!/bin/bash -e

SOURCE=$(dirname "${BASH_SOURCE[0]}")
tuf --dir "$SOURCE"/bucket_sandbox snapshot
tuf --dir "$SOURCE"/bucket_sandbox timestamp
tuf --dir "$SOURCE"/bucket_sandbox commit
mkdir -p "$SOURCE"/.minio_data/test-project
rsync -avu --delete "$SOURCE/bucket_sandbox/repository/" "$SOURCE/.minio_data/test-project/"
docker-compose --file "$SOURCE"/docker-compose.yaml up --detach