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
	gci write -s Standard -s Default -s 'Prefix(github.com/werf)' pkg/ cmd/
	gofumpt -extra -w cmd/ pkg/

lint:
	GOOS=$(OS) GOARCH="$(GOARCH)" golangci-lint run ./... --config ../.golangci.yaml

all: fmt lint
