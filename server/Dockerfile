FROM alpine:latest AS builder

ARG BUILDX_VERSION="v0.10.5"

RUN apk add --no-cache wget && \
    if [ "$(arch)" = "x86_64" ]; then ARCH="amd64"; \
    elif [ "$(arch)" = "aarch64" ]; then ARCH="arm64"; \
    else echo "Unsupported architecture: $(arch)" && exit 1; fi && \
    wget -O /docker-buildx \
    "https://github.com/docker/buildx/releases/download/${BUILDX_VERSION}/buildx-${BUILDX_VERSION}.linux-${ARCH}" && \
    chmod +x /docker-buildx

FROM ghcr.io/werf/trdl-dev-vault:latest
RUN addgroup vault ping
ENV VAULT_ADDR=http://localhost:8200
ENV VAULT_TOKEN=root
COPY --from=builder /docker-buildx /usr/lib/docker/cli-plugins/docker-buildx