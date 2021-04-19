#!/bin/bash

set -e

CWD=`pwd`
SOURCE=`dirname ${BASH_SOURCE[0]}`

cd $SOURCE

export GO111MODULE=on

mkdir -p vault/plugins
go build -o vault/plugins/vault-plugin-secrets-trdl github.com/werf/vault-plugin-secrets-trdl/cmd/vault-plugin-secrets-trdl

cd $CWD
