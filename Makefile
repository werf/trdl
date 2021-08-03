GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

GOSRC = $(shell find . -type f -name '*.go')
.DEFAULT_GOAL := all

.PHONY: fmt lint clean tail

all: fmt lint .run tail

fmt:
	go fmt $$(go list ./...)

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run ./... --config .golangci.yaml

vault/plugins/vault-plugin-secrets-trdl: $(GOSRC)
	CGO_ENABLED=0 GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-secrets-trdl cmd/vault-plugin-secrets-trdl/main.go

build: vault/plugins/vault-plugin-secrets-trdl

.run: vault/plugins/vault-plugin-secrets-trdl
	# Run minio, create bucket
	docker rm -f trdl_dev_minio || true
	sudo rm -rf .minio_data
	mkdir .minio_data
	docker run --name trdl_dev_minio --detach --rm -p 9000:9000 -p 9001:9001 --volume $$(pwd)/.minio_data:/data minio/minio server /data --console-address ":9001"
	( \
		while ! docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc ls main ; \
		do \
			sleep 1 ; \
		done ; \
	)
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/trdl-test-project

	# Run vault dev server
	docker rm -f trdl_dev_vault || true
	rm -f trdl.log
	touch trdl.log
	docker run --workdir /app --privileged --name trdl_dev_vault --detach --volume /var/run/docker.sock:/var/run/docker.sock --volume $$(pwd):/app -p 8200:8200 vault:latest server -dev -dev-root-token-id=root -dev-plugin-dir=/app/vault/plugins -log-level trace
	( \
		while ! VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault status ; \
		do \
			sleep 1 ; \
		done ; \
	)

	# Enable and configure plugin
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl-test-project/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project

	touch .run

tail:
	tail -f trdl.log

clean:
	rm -f ./vault/plugins/vault-plugin-secrets-trdl
	docker rm -f trdl_dev_minio || true
	docker rm -f trdl_dev_vault || true
	sudo rm -rf .minio_data
	rm -f trdl.log
