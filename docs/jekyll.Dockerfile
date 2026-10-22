FROM alpine:3.22 AS certificates

RUN apk add --no-cache ca-certificates

FROM jekyll/builder:3

COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY Gemfile Gemfile.lock ./

RUN gem install bundler -v 2.2.28 && bundle install
