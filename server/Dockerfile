FROM debian:bookworm

ENV DEBIAN_FRONTEND=noninteractive
ENV VAULT_ADDR=http://localhost:8200
ENV VAULT_TOKEN=root
ENV VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200

RUN apt-get update && \
    apt-get install -y ca-certificates curl gnupg binutils-multiarch && \
    install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://apt.releases.hashicorp.com/gpg \
      | gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com bookworm main" \
      > /etc/apt/sources.list.d/hashicorp.list && \
    curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable" \
      > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y vault docker-ce-cli docker-buildx-plugin && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["vault"]
