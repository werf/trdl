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

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run ./... --config .golangci.yaml

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-secrets-trdl cmd/vault-plugin-secrets-trdl/main.go

start:
	export TRDL_DEV=1 ; vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins -log-level trace

dev: minio
	VAULT_ADDR='http://127.0.0.1:8200' vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
	VAULT_ADDR='http://127.0.0.1:8200' vault write trdl-test-project/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project

	tail -f trdl.log

MINIO_RUNNING = $(shell docker inspect trdl_minio 2>&1 >/dev/null && echo true || echo false)
minio:
ifneq ($(MINIO_RUNNING), true)
	docker run --name trdl_minio --detach --rm -p 9000:9000 -p 9001:9001 --volume $$(pwd)/minio_data:/data minio/minio server /data --console-address ":9001"
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/trdl-test-project
endif

clean:
	docker rm -f trdl_minio || true
	rm -f ./vault/plugins/vault-plugin-secrets-trdl
	sudo rm -rf ./minio_data

fmt:
	go fmt $$(go list ./...)

.PHONY: lint build clean fmt start enable
