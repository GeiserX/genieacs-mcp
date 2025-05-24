# ────────────────────────────────────────────────────────────────
# Stage 1 – build the Go binary
# ────────────────────────────────────────────────────────────────
FROM golang:1.24 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary for Alpine/glibc-less images
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/genieacs-mcp ./cmd/server

# ────────────────────────────────────────────────────────────────
# Stage 2 – tiny runtime image
# ────────────────────────────────────────────────────────────────
FROM busybox:glibc

COPY --from=builder /out/genieacs-mcp /usr/local/bin/genieacs-mcp
USER 65532:65532 # non-root “nobody” user
EXPOSE 8080

ENV ACS_URL=http://genieacs:7557
ENV ACS_USER=admin
ENV ACS_PASS=admin

ENTRYPOINT ["/usr/local/bin/genieacs-mcp"]