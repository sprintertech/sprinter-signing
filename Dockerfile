# Copyright 2020 ChainSafe Systems
# SPDX-License-Identifier: LGPL-3.0-only

FROM alpine:3.6 as alpine
RUN apk add -U --no-cache ca-certificates

FROM  golang:1.23 AS builder
ADD . /src
WORKDIR /src
RUN cd /src && echo $(ls -1 /src)
RUN --mount=type=secret,id=GH_USER_NAME \
    --mount=type=secret,id=GH_USER_TOKEN \
    GH_USER_NAME=$(cat /run/secrets/GH_USER_NAME) && \
    GH_USER_TOKEN=$(cat /run/secrets/GH_USER_TOKEN) && \
    go env -w GOPRIVATE=github.com/sprintertech && \
    git config --global url."https://${GH_USER_NAME}:${GH_USER_TOKEN}@github.com".insteadOf "https://github.com" 
RUN go mod download
RUN go build -ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore -X github.com/sprintertech/sprinter-signing/app.Version=$(sed -n '0,/## \[\([0-9.]*\)\]/s/.*\[\([0-9.]*\)\].*/\1/p' CHANGELOG.md)" -o /signing .

# final stage
FROM debian:stable-slim
COPY --from=builder /signing ./
RUN chmod +x ./signing
RUN mkdir -p /mount
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["./signing"]
