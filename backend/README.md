# PersonalChatAI-Aulia — Backend (Go)

Go backend untuk PersonalChatAI-Aulia. Stack: **chi + pgx + golang-migrate + golang-jwt**, Groq Chat Completions API via raw `net/http` + SSE.

## Endpoints

Semua endpoint kecuali `/healthz` butuh `Authorization: Bearer <jwt>` header. JWT di-issue oleh FE Auth.js (`GET /api/token`) dan ditandatangani pakai HS256 menggunakan `AUTH_SECRET` yang sama dengan backend.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET`  | `/healthz` | No | Healthcheck |
| `GET`  | `/me` | Yes | Current user + settings |
| `PUT`  | `/me/settings` | Yes | Update default model / temperature / system prompt |
| `GET`  | `/conversations` | Yes | List conversations (scoped per user, sort by `updated_at` desc) |
| `POST` | `/conversations` | Yes | Create new conversation (pakai user settings sebagai default) |
| `GET`  | `/conversations/{id}` | Yes | Get conversation detail |
| `PATCH`| `/conversations/{id}` | Yes | Update (rename / change settings) |
| `DELETE`| `/conversations/{id}` | Yes | Delete (cascade messages) |
| `GET`  | `/conversations/{id}/messages` | Yes | List messages |
| `POST` | `/conversations/{id}/title` | Yes | Auto-generate title pakai Llama 8B instant |
| `POST` | `/chat` | Yes | **Streaming endpoint** — Vercel AI SDK data stream protocol |

## Setup

```bash
cd backend
cp .env.example .env
# Edit .env — isi GROQ_API_KEY + DATABASE_URL + AUTH_SECRET
# AUTH_SECRET harus identik dengan FE .env.local (generate: openssl rand -hex 32)

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
├── cmd/server/main.go            # Entry point: chi router + auth middleware + graceful shutdown
├── internal/
│   ├── config/config.go          # Env loading + validation (AUTH_SECRET ≥ 32 chars)
│   ├── db/
│   │   ├── pool.go               # pgx connection pool
│   │   ├── models.go             # User, Conversation, Message structs
│   │   ├── repo_user.go          # Upsert by google_sub + settings CRUD
│   │   ├── repo_conversation.go  # User-scoped CRUD (ListByUser, GetByUser, ...)
│   │   ├── repo_message.go
│   │   └── migrations/
│   │       ├── 001_initial.{up,down}.sql
│   │       └── 002_add_users.{up,down}.sql
│   ├── handler/                  # HTTP handlers (thin — call repo/service)
│   │   ├── chat.go               # SSE streaming (user-scoped)
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── title.go
│   │   ├── me.go                 # GET /me + PUT /me/settings
│   │   └── errors.go
│   ├── service/
│   │   └── anthropic.go          # Groq Chat Completions client (OpenAI-compatible SSE)
│   ├── middleware/
│   │   ├── logger.go
│   │   └── auth.go               # HS256 JWT validate + user upsert + ctx injection
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

## Auth (Minggu 3)

Middleware `internal/middleware/auth.go` validates a Bearer JWT (HS256) using `AUTH_SECRET`, upserts the user (keyed by Google `sub`), and stores `*db.User` di `r.Context()` via `appmw.UserFromCtx(ctx)`.

Token shape (issued by FE `/api/token`):

```json
{
  "sub":     "117823498023480923",   // Google's stable user id
  "email":   "user@gmail.com",
  "name":    "Aulia Afriza",
  "picture": "https://lh3.googleusercontent.com/...",
  "iat":     1715990400,
  "exp":     1715992200               // 30 min lifetime
}
```

Failure modes (all return 401):
- Missing/malformed Authorization header
- Invalid signature (mismatched AUTH_SECRET)
- Expired token (FE auto-refreshes via `/api/token`)
- Missing `sub` or `email` claims

Part of [PersonalChatAI-Aulia](../README.md) — Roadmap AI Engineer Minggu 3.
