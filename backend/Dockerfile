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

# ca-certificates for TLS (Groq API, Neon Postgres over SSL)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/server ./server

EXPOSE 8080

ENTRYPOINT ["./server"]
