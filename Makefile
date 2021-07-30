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
.PHONY: fmt lint minio_up minio_down

fmt:
	gofumpt -l -w .

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run ./... --config .golangci.yaml

minio_up:
	./scripts/minio/up.sh

minio_down:
	./scripts/minio/down.sh

all: fmt lint