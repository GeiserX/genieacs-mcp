############################################################################
# docker buildx build --platform linux/amd64,linux/arm64 \                 #
#   -t drumsergio/genieacs-mcp:version -t drumsergio/genieacs-mcp:latest \ #
#   --push .                                                               #
############################################################################
# ───────────────────────────────────────────────
# Stage 1 – build the Go binary
# ───────────────────────────────────────────────
FROM golang:1.27 AS builder
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
RUN CGO_ENABLED=0 go build -ldflags "-s -w \
      -X github.com/geiserx/genieacs-mcp/version.Version=${VERSION} \
      -X github.com/geiserx/genieacs-mcp/version.Commit=${COMMIT} \
      -X github.com/geiserx/genieacs-mcp/version.Date=${DATE}" \
    -o /out/genieacs-mcp ./cmd/server

# ───────────────────────────────────────────────
# Stage 2 – tiny runtime image
# ───────────────────────────────────────────────
FROM alpine:3.24
LABEL io.modelcontextprotocol.server.name="io.github.GeiserX/genieacs-mcp"

COPY --from=builder /out/genieacs-mcp /usr/local/bin/genieacs-mcp
EXPOSE 8080

ENV ACS_URL=http://genieacs:7557
ENV TRANSPORT=stdio

ENTRYPOINT ["/usr/local/bin/genieacs-mcp"]
