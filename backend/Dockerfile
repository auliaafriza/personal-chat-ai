# --- Build stage ---
# Pin Go 1.25 — indirect dep rogpeppe/go-internal@v1.15.0 requires it.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download deps first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/server ./cmd/server


# --- Runtime stage ---
FROM alpine:3.21

# ca-certificates for TLS (Groq API, Neon Postgres over SSL).
# tzdata untuk get_current_time tool (time.LoadLocation untuk Asia/Jakarta dll).
# git untuk run_shell tool (read-only allowlist: log/status/diff/show/branch/ls-files/blame).
RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

COPY --from=builder /app/bin/server ./server

# Workspace volume mount point — Railway: attach Volume di /data.
# Coding tools (Minggu 8) bikin per-user folder di sini.
RUN mkdir -p /data/workspaces
ENV WORKSPACE_ROOT=/data/workspaces

EXPOSE 8080

ENTRYPOINT ["./server"]
