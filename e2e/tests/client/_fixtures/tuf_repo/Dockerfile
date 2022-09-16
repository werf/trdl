FROM golang:1.19 as tuf_repo
RUN go install github.com/theupdateframework/go-tuf/cmd/tuf@v0.5.0
RUN mkdir /workspace
WORKDIR /workspace
COPY staged /workspace/staged
COPY keys /workspace/keys
RUN find staged/targets -type f -print0 | xargs -0 -n1 | sed -e "s|^staged/targets/||" | tuf add \
        && tuf snapshot \
        && tuf timestamp \
        && tuf commit

FROM halverneus/static-file-server:v1.8.8
COPY --from=tuf_repo /workspace/repository /web
