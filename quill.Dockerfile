FROM alpine:3.22.1

RUN apk update && apk add curl file && \
    curl -sSfL https://get.anchore.io/quill | sh -s -- -b /usr/local/bin