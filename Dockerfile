############################################################################
# docker buildx build --platform linux/amd64,linux/arm64 \                 #
#   -t drumsergio/genieacs-mcp:version -t drumsergio/genieacs-mcp:latest \ #
#   --push .                                                               #
############################################################################
# ───────────────────────────────────────────────
# Stage 1 – build the Go binary
# ───────────────────────────────────────────────
# Bases are pinned by digest (Dependabot keeps them current). The builder is
# the exact Go release go.mod names, so go1.27.4 (eMudhra-rejecting, and fixed
# for GO-2026-6859) is the one that builds the image.
FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS builder
LABEL maintainer="acsdesk@protonmail.com"

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The release passes these from the tag (goreleaser build_flag_templates), so the
# image logs and reports the version it was built from instead of "dev".
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w \
      -X github.com/geiserx/genieacs-mcp/version.Version=${VERSION} \
      -X github.com/geiserx/genieacs-mcp/version.Commit=${COMMIT} \
      -X github.com/geiserx/genieacs-mcp/version.Date=${DATE}" \
    -o /out/genieacs-mcp ./cmd/server

# ───────────────────────────────────────────────
# Stage 2 – tiny runtime image
# ───────────────────────────────────────────────
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
# The base image can lag the CA bundle; take the current one, which drops
# the compromised eMudhra (emSign) intermediates.
RUN apk upgrade --no-cache ca-certificates-bundle
LABEL io.modelcontextprotocol.server.name="io.github.GeiserX/genieacs-mcp"

COPY --from=builder /out/genieacs-mcp /usr/local/bin/genieacs-mcp
EXPOSE 8080

ENV ACS_URL=http://genieacs:7557
ENV TRANSPORT=stdio

# Run as an unprivileged user: the server needs no root, and it holds the
# ACS credentials.
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/genieacs-mcp"]
