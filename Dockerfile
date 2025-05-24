# syntax=docker/dockerfile:1
############################################################################
# docker buildx build --platform linux/amd64,linux/arm64 \                 #
#   -t drumsergio/genieacs-mcp:version -t drumsergio/genieacs-mcp:latest \ #
#   --push .                                                               #
############################################################################
# ───────────────────────────────────────────────
# Stage 1 – build the Go binary
# ───────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder
LABEL maintainer="acsdesk@protonmail.com"

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# cross-compile for the platform we are ultimately building *for*
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOARM=${TARGETVARIANT#v} \
    go build -ldflags "-s -w" -o /out/genieacs-mcp ./cmd/server

# ───────────────────────────────────────────────
# Stage 2 – tiny runtime image
# ───────────────────────────────────────────────
FROM --platform=$TARGETPLATFORM busybox:glibc

COPY --from=builder /out/genieacs-mcp /usr/local/bin/genieacs-mcp
USER nobody
EXPOSE 8080

ENV ACS_URL=http://genieacs:7557
ENV ACS_USER=admin
ENV ACS_PASS=admin

ENTRYPOINT ["/usr/local/bin/genieacs-mcp"]