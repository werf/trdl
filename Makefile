GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-secrets-trdl cmd/vault-plugin-secrets-trdl/main.go

start:
	export TRDL_DEV=1 ; vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins -log-level trace

enable:
ifndef TRDL_S3_SECRET_ACCESS_KEY
	$(error TRDL_S3_SECRET_ACCESS_KEY variable required)
endif
ifndef TRDL_S3_ACCESS_KEY_ID
	$(error TRDL_S3_ACCESS_KEY_ID variable required)
endif

	VAULT_ADDR='http://127.0.0.1:8200' vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
	vault write trdl-test-project/configure s3_secret_access_key="${TRDL_S3_SECRET_ACCESS_KEY}" s3_access_key_id="${TRDL_S3_ACCESS_KEY_ID}" s3_region=ru-central1 s3_bucket_name=trdl-test-project s3_endpoint=https://storage.yandexcloud.net required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project

	VAULT_ADDR='http://127.0.0.1:8200' vault secrets enable -path=trdl-test-werf vault-plugin-secrets-trdl
	vault write trdl-test-werf/configure s3_secret_access_key="${TRDL_S3_SECRET_ACCESS_KEY}" s3_access_key_id="${TRDL_S3_ACCESS_KEY_ID}" s3_region=ru-central1 s3_bucket_name=trdl-test-werf s3_endpoint=https://storage.yandexcloud.net required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/werf

	tail -f trdl.log

clean:
	rm -f ./vault/plugins/vault-plugin-secrets-trdl

fmt:
	go fmt $$(go list ./...)

.PHONY: build clean fmt start enable
