#!/bin/bash -e

SOURCE=$(dirname "${BASH_SOURCE[0]}")

mkdir -p "$SOURCE"/bucket_sandbox/staged/targets
date > "$SOURCE"/bucket_sandbox/staged/targets/date
tuf --dir "$SOURCE"/bucket_sandbox add date
tuf --dir "$SOURCE"/bucket_sandbox snapshot
tuf --dir "$SOURCE"/bucket_sandbox timestamp
tuf --dir "$SOURCE"/bucket_sandbox commit
mkdir -p "$SOURCE"/.minio_data/test-project
rsync -avu --delete "$SOURCE/bucket_sandbox/repository/" "$SOURCE/.minio_data/test-project/"
docker-compose --file "$SOURCE"/docker-compose.yaml up --detach
until $(curl --output /dev/null --silent --head --fail http://localhost:9000/test-project); do printf '.'; sleep 1; done
