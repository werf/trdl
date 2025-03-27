FROM --platform=linux/amd64 golang:1.23-bookworm@sha256:3149bc5043fa58cf127fd8db1fdd4e533b6aed5a40d663d4f4ae43d20386665f

RUN apt-get -y update && \
    apt-get -y install file && \
    curl -sSLO https://github.com/go-task/task/releases/download/v3.33.1/task_linux_amd64.deb && \
    apt-get -y install ./task_linux_amd64.deb && \
    rm -rf ./task_linux_amd64.deb /var/cache/apt/* /var/lib/apt/lists/* /var/log/*

ADD server /.trdl-deps/server
ADD client /.trdl-deps/client
ADD e2e /.trdl-deps/e2e
ADD docs /.trdl-deps/docs
ADD Taskfile.dist.yaml /.trdl-deps

RUN cd /.trdl-deps && \
    task build:dist:all version=base && \
    task client:verify:dist:binaries version=base && \
    task server:verify:dist:binaries version=base && \
    rm -rf /.trdl-deps

RUN git config --global --add safe.directory /git