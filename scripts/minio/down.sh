#!/bin/bash -e

SOURCE=$(dirname "${BASH_SOURCE[0]}")
docker compose --file "$SOURCE"/docker-compose.yaml down