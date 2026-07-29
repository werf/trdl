FROM --platform=linux/amd64 golang:1.25.12-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58

RUN apt-get -y update && \
    apt-get -y install file && \
    curl -sSLO https://github.com/go-task/task/releases/download/v3.33.1/task_linux_amd64.deb && \
    apt-get -y install ./task_linux_amd64.deb && \
    rm -rf ./task_linux_amd64.deb /var/cache/apt/* /var/lib/apt/lists/* /var/log/*

ENV TASK_X_REMOTE_TASKFILES=1

ADD server /.trdl-deps/server
ADD client /.trdl-deps/client
ADD release /.trdl-deps/release
ADD e2e /.trdl-deps/e2e
ADD docs /.trdl-deps/docs
ADD actions /.trdl-deps/actions
ADD Taskfile.dist.yaml /.trdl-deps

RUN cd /.trdl-deps && \
    task --yes server:deps:install:c && \
    task --yes build:dist:all version=base && \
    task --yes client:verify:dist:binaries version=base && \
    task --yes server:verify:dist:binaries version=base && \
    rm -rf /.trdl-deps

RUN git config --global --add safe.directory /git