FROM golang:1.19 as tuf_repo

RUN go install github.com/theupdateframework/go-tuf/cmd/tuf@v0.5.0

RUN mkdir /workspace
WORKDIR /workspace

ENV TUF_ROOT_PASSPHRASE="foobar" \
    TUF_TARGETS_PASSPHRASE="foobar" \
    TUF_SNAPSHOT_PASSPHRASE="foobar" \
    TUF_TIMESTAMP_PASSPHRASE="foobar"
RUN tuf gen-key root \
        && tuf gen-key targets \
        && tuf gen-key snapshot \
        && tuf gen-key timestamp \
        && tuf sign root.json

COPY staged /workspace/staged
RUN find staged/targets -type f -print0 | xargs -0 -n1 | sed -e "s|^staged/targets/||" | tuf add \
        && tuf snapshot \
        && tuf timestamp \
        && tuf commit

FROM halverneus/static-file-server:v1.8.8
COPY --from=tuf_repo /workspace/repository /web
