#!/bin/bash

set -e

CWD=`pwd`
SOURCE=`dirname ${BASH_SOURCE[0]}`

cd $SOURCE

export GO111MODULE=on
go install github.com/flant/vault-plugin-trdl/cmd/vault-plugin-trdl

cd $CWD
