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

all: fmt lint restart tail

fmt:
	gci write -s Standard -s Default -s 'Prefix(github.com/werf)' pkg/ cmd/ *.go
	gofumpt -extra -w cmd/ pkg/ *.go

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run ./... --config ../.golangci.yaml

vault/plugins/vault-plugin-secrets-trdl: $(GOSRC)
	CGO_ENABLED=0 GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-secrets-trdl cmd/vault-plugin-secrets-trdl/main.go

build: vault/plugins/vault-plugin-secrets-trdl

restart:
	docker rm -f trdl_dev_minio || true
	docker run --name trdl_dev_minio --detach --rm -p 9000:9000 -p 9001:9001 --volume $$(pwd)/.minio_data:/data minio/minio server /data --console-address ":9001"
	( \
		while ! docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc ls main ; \
		do \
			sleep 1 ; \
		done ; \
	)

	# Run vault dev server
	docker rm -f trdl_dev_vault || true
	docker run --workdir /app --privileged --name trdl_dev_vault -e VAULT_PLUGIN_SECRETS_TRDL_PPROF_ENABLE=1 -e VAULT_PLUGIN_SECRETS_TRDL_DEBUG=1 --detach --volume /var/run/docker.sock:/var/run/docker.sock --volume $$(pwd):/app -p 8200:8200 ghcr.io/werf/trdl-dev-vault:latest server -dev -dev-root-token-id=root -dev-plugin-dir=/app/vault/plugins -log-level trace
	( \
		while ! VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault status ; \
		do \
			sleep 1 ; \
		done ; \
	)

	# Enable and configure plugin
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl-test-project1 vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl-test-project1/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project1 s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project git_trdl_path=p1/trdl.yaml git_trdl_channels_path=p1/trdl_channels.yaml

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl-test-project2 vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl-test-project2/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project2 s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project git_trdl_path=p2/trdl.yaml git_trdl_channels_path=p2/trdl_channels.yaml

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=kubedog vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write kubedog/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=kubedog s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/kubedog

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=werf vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write werf/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=werf s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/werf git_trdl_channels_branch=multiwerf


.run: vault/plugins/vault-plugin-secrets-trdl
	# Run minio, create bucket
	docker rm -f trdl_dev_minio || true
	docker run --rm --volume $$(pwd):/wrk alpine rm -rf /wrk/.minio_data
	mkdir .minio_data
	docker run --name trdl_dev_minio --detach --rm -p 9000:9000 -p 9001:9001 --volume $$(pwd)/.minio_data:/data minio/minio server /data --console-address ":9001"
	( \
		while ! docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc ls main ; \
		do \
			sleep 1 ; \
		done ; \
	)
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/trdl-test-project1
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc policy set public main/trdl-test-project1
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/trdl-test-project2
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc policy set public main/trdl-test-project2
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/werf
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc policy set public main/werf
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/trdl
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc policy set public main/trdl
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc mb main/kubedog
	docker run -ti --rm -e MC_HOST_main=http://minioadmin:minioadmin@$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 minio/mc policy set public main/kubedog

	# Run vault dev server
	docker rm -f trdl_dev_vault || true
	docker run --workdir /app --privileged --name trdl_dev_vault -e VAULT_PLUGIN_SECRETS_TRDL_PPROF_ENABLE=1 -e VAULT_PLUGIN_SECRETS_TRDL_DEBUG=1 --detach --volume /var/run/docker.sock:/var/run/docker.sock --volume $$(pwd):/app -p 8200:8200 ghcr.io/werf/trdl-dev-vault:latest server -dev -dev-root-token-id=root -dev-plugin-dir=/app/vault/plugins -log-level trace
	( \
		while ! VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault status ; \
		do \
			sleep 1 ; \
		done ; \
	)

	# Enable and configure plugin
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl-test-project1 vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl-test-project1/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project1 s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project git_trdl_path=p1/trdl.yaml git_trdl_channels_path=p1/trdl_channels.yaml

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl-test-project2 vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl-test-project2/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl-test-project2 s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project git_trdl_path=p2/trdl.yaml git_trdl_channels_path=p2/trdl_channels.yaml

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=trdl vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write trdl/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=trdl s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=kubedog vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write kubedog/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=kubedog s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/kubedog

	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault secrets enable -path=werf vault-plugin-secrets-trdl
	VAULT_TOKEN=root VAULT_ADDR=http://$$(docker inspect trdl_dev_vault --format "{{ .NetworkSettings.IPAddress }}"):8200 vault write werf/configure s3_secret_access_key=minioadmin s3_access_key_id=minioadmin s3_bucket_name=werf s3_region=ru-central1 s3_endpoint=http://$$(docker inspect trdl_dev_minio --format "{{ .NetworkSettings.IPAddress }}"):9000 required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/werf git_trdl_channels_branch=multiwerf

	touch .run

tail:
	docker logs -f trdl_dev_vault

clean:
	rm -f ./vault/plugins/vault-plugin-secrets-trdl
	docker rm -f trdl_dev_minio || true
	docker rm -f trdl_dev_vault || true
	docker run --rm --volume $$(pwd):/wrk alpine rm -rf /wrk/.minio_data

install-to-dev: build
	: "$${TRDL_DEV_SSH_HOST:?not set}"
	scp vault/plugins/vault-plugin-secrets-trdl ubuntu@$$TRDL_DEV_SSH_HOST:/home/ubuntu/
	ssh -tt ubuntu@$$TRDL_DEV_SSH_HOST sudo /opt/update-vault-plugin-secrets-trdl.sh
