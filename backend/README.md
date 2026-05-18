# PersonalGPT — Backend (Go)

Go backend untuk PersonalGPT. Stack: **chi + pgx + golang-migrate**, Anthropic API via raw `net/http` + SSE (no SDK — alpha SDK API-nya masih sering berubah).

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET`  | `/healthz` | Healthcheck |
| `GET`  | `/conversations` | List conversations (sort by `updated_at` desc) |
| `POST` | `/conversations` | Create new conversation |
| `GET`  | `/conversations/{id}` | Get conversation detail |
| `PATCH`| `/conversations/{id}` | Update (rename / change settings) |
| `DELETE`| `/conversations/{id}` | Delete (cascade messages) |
| `GET`  | `/conversations/{id}/messages` | List messages (sort by `created_at` asc) |
| `POST` | `/conversations/{id}/title` | Auto-generate title pakai Claude Haiku |
| `POST` | `/chat` | **Streaming endpoint** — Vercel AI SDK data stream protocol |

## Setup

```bash
cd backend
cp .env.example .env
# Edit .env — isi ANTHROPIC_API_KEY + DATABASE_URL (Neon connection string)

go mod download
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Migrate
make migrate-up

# Run
make run
# atau: go run ./cmd/server
```

Server listen di `:8080`. Test:

```bash
curl http://localhost:8080/healthz
# → {"ok":true}

curl http://localhost:8080/conversations
# → []

curl -X POST http://localhost:8080/conversations
# → {"id":"01HXXX...","title":"New chat",...}
```

## Folder Structure

```
backend/
├── cmd/server/main.go            # Entry point: chi router + graceful shutdown
├── internal/
│   ├── config/config.go          # Env loading + validation
│   ├── db/
│   │   ├── pool.go               # pgx connection pool
│   │   ├── models.go             # Conversation, Message structs
│   │   ├── repo_conversation.go  # CRUD operations
│   │   ├── repo_message.go
│   │   └── migrations/
│   │       └── 001_initial.{up,down}.sql
│   ├── handler/                  # HTTP handlers (thin — call repo/service)
│   │   ├── chat.go               # SSE streaming
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── title.go
│   │   └── errors.go
│   ├── service/
│   │   └── anthropic.go          # Anthropic API client (net/http + SSE parsing)
│   ├── middleware/logger.go
│   └── stream/ai_sdk.go          # Vercel AI SDK data stream protocol writer
├── Makefile
└── .env.example
```

## Streaming Protocol

`POST /chat` returns **Vercel AI SDK data stream protocol v1**, so FE's `useChat()` works without modification.

Format: newline-delimited frames `<type>:<json>\n`

```
f:{"messageId":"msg-01HXX..."}
0:"Hello"
0:", how can I help?"
e:{"finishReason":"stop","usage":{"promptTokens":10,"completionTokens":7}}
d:{"finishReason":"stop","usage":{"promptTokens":10,"completionTokens":7}}
```

Headers: `Content-Type: text/plain; charset=utf-8` + `X-Vercel-Ai-Data-Stream: v1`.

Implementation: `internal/stream/ai_sdk.go`.

## Migrations

```bash
make migrate-create name=add_users     # bikin file baru
make migrate-up                        # apply pending
make migrate-down                      # rollback 1
```

SQL files di `internal/db/migrations/`. Format: `<seq>_<name>.{up,down}.sql`.

## Development

Dev server tanpa hot reload:
```bash
make run        # atau: go run ./cmd/server
```

Dengan hot reload (install [air](https://github.com/air-verse/air) dulu):
```bash
go install github.com/air-verse/air@latest
air
```

## Conventions

- Handlers thin — semua logic di `service/` atau `db/`
- Errors: log internally, return clean message ke client (jangan leak detail)
- Context: pakai `r.Context()` di handlers, propagate ke DB/service
- IDs: ULID (lexicographic-sortable, time-ordered) — `oklog/ulid/v2`
- Migrations forward-only di production. Rollback hanya untuk dev.

## Troubleshooting

### `migrate: command not found`

`go install` taruh binary di `$(go env GOPATH)/bin` (default: `~/go/bin`) yang biasanya nggak otomatis di-add ke `PATH`.

**Cepat (cuma untuk session ini):** Makefile-nya sekarang resolve binary pakai `$(go env GOPATH)/bin/migrate` otomatis, jadi `make migrate-up` jalan tanpa PATH change.

**Permanen (recommended)** — tambahkan ke `~/.zshrc` (macOS default) atau `~/.bashrc`:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
# Verify:
which migrate
```

### `connection refused` ke Postgres

Cek `.env` — pastikan `DATABASE_URL` punya `?sslmode=require` di akhir (Neon wajib SSL).

### `pq: SSL is not enabled on the server` (lokal Postgres)

Kalau pakai Postgres lokal (bukan Neon), ganti `sslmode=require` jadi `sslmode=disable`.

---

Part of [PersonalGPT](../README.md) — Roadmap AI Engineer Minggu 2.
