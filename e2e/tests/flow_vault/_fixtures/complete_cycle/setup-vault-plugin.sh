#!/bin/bash

VAULT_CONTAINER="${VAULT_CONTAINER:-trdl-vault-dev}"
MINIO_CONTAINER="${MINIO_CONTAINER:-minio-dev}"

wait_for_healthy() {
  local container=$1
  local timeout=$2
  local elapsed=0

  echo "Waiting for container '$container' to become healthy (timeout: ${timeout}s)..."

  until [ "$(docker inspect -f '{{.State.Health.Status}}' "$container")" == "healthy" ]; do
    if ! docker inspect -f '{{.State.Running}}' "$container" &>/dev/null; then
      echo "Error: container '$container' not running or not found"
      return 1
    fi

    sleep 1
    elapsed=$((elapsed + 1))

    if [ "$elapsed" -ge "$timeout" ]; then
      echo "Error: timeout waiting for container '$container' to become healthy"
      return 1
    fi
  done

  echo "Container '$container' is healthy"
  return 0
}

wait_for_healthy "$MINIO_CONTAINER" 300 || exit 1

echo "Minio is healthy, proceeding with setup..."

MINIO_CLIENT_EXEC="docker compose exec mc"

$MINIO_CLIENT_EXEC mc alias set main http://$MINIO_CONTAINER:9000 minioadmin minioadmin
$MINIO_CLIENT_EXEC mc mb main/repo
$MINIO_CLIENT_EXEC mc anonymous set download main/repo


wait_for_healthy "$VAULT_CONTAINER" 300 || exit 1

echo "Vault is healthy, proceeding with setup..."

VAULT_EXEC="docker compose exec vault"
DOCKER_SOCK_GID=$(stat -c '%g' /var/run/docker.sock)

$VAULT_EXEC apk add --no-cache git
$VAULT_EXEC addgroup -g $DOCKER_SOCK_GID docker3
$VAULT_EXEC addgroup vault docker3

echo "Restart Vault..."
docker compose restart vault
wait_for_healthy "$VAULT_CONTAINER" 300 || exit 1
echo "Vault is ready"