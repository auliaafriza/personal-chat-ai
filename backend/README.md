# PersonalChatAI-Aulia — Backend (Go)

Go backend untuk PersonalChatAI-Aulia. Stack: **chi + pgx + pgvector + golang-migrate + golang-jwt**, Groq (chat) + Voyage AI (embeddings) via raw `net/http` + SSE.

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
| `POST` | `/chat` | Yes | **Streaming endpoint** — Vercel AI SDK data stream protocol + auto-RAG (Minggu 5) |
| `GET`  | `/documents` | Yes | List user's documents (Minggu 4) |
| `POST` | `/documents` | Yes | Upload file or paste text → parse + chunk + embed |
| `GET`  | `/documents/{id}` | Yes | Document detail + all chunks |
| `DELETE`| `/documents/{id}` | Yes | Cascade delete document + chunks |
| `POST` | `/documents/search` | Yes | Cosine similarity search across all user's chunks |

## Setup

```bash
cd backend
cp .env.example .env
# Edit .env — isi GROQ_API_KEY + VOYAGE_API_KEY + DATABASE_URL + AUTH_SECRET
# AUTH_SECRET harus identik dengan FE .env.local (generate: openssl rand -hex 32)

go mod download
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Migrate (001 + 002 + 003)
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
│   │   ├── models.go             # User, Conversation, Message, Document, DocumentChunk
│   │   ├── repo_user.go          # Upsert by google_sub + settings CRUD
│   │   ├── repo_conversation.go  # User-scoped CRUD
│   │   ├── repo_message.go
│   │   ├── repo_document.go      # CreateWithChunks (tx) + SearchSimilar (cosine)
│   │   └── migrations/
│   │       ├── 001_initial.{up,down}.sql
│   │       ├── 002_add_users.{up,down}.sql
│   │       ├── 003_add_documents.{up,down}.sql      # pgvector + HNSW index
│   │       ├── 004_add_message_sources.{up,down}.sql # JSONB sources (citation persist)
│   │       └── 005_add_tsvector.{up,down}.sql        # tsvector + GIN index (Minggu 6 hybrid)
│   ├── handler/                  # HTTP handlers (thin — call repo/service)
│   │   ├── chat.go               # SSE streaming (user-scoped)
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── title.go
│   │   ├── me.go                 # GET /me + PUT /me/settings
│   │   ├── document.go           # Upload + List + Search (Minggu 4)
│   │   └── errors.go
│   ├── service/
│   │   ├── anthropic.go          # Groq Chat Completions client (SSE + tool_calls, Minggu 7)
│   │   ├── embeddings.go         # Voyage AI client (voyage-3-lite, 512 dim)
│   │   ├── rerank.go             # Voyage rerank-2 client (cross-encoder, Minggu 6)
│   │   ├── retriever.go          # Orchestrator: embed → hybrid → rerank (Minggu 6)
│   │   ├── parser.go             # txt/md/pdf/docx → plain text
│   │   └── chunker.go            # Heading-aware + fallback fixed-size chunking
│   ├── tools/                    # Minggu 7 — tool calling
│   │   ├── types.go              # Tool interface + Schema (OpenAI format)
│   │   ├── registry.go           # Tool registry (Register/Run by name)
│   │   ├── web_search.go         # Tavily AI search
│   │   ├── fetch_url.go          # HTML fetch + markdown conversion
│   │   ├── calculator.go         # Math expression eval (expr-lang/expr)
│   │   └── current_time.go       # Real-time clock + timezone
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

## RAG (Minggu 5)

Auto-RAG di `POST /chat` (lihat `internal/handler/chat.go`). Aktif kalau user punya ≥1 chunk:

1. Ambil pesan user terakhir → embed via Voyage (`input_type=query`).
2. `SearchSimilar` top-5 (cosine, HNSW index).
3. Filter `similarity ≥ 0.30` (anti-noise untuk chit-chat).
4. Build context block `[n] (Judul — Heading)\n{chunk}` → append ke system prompt + instruksi citation.
5. Kirim `sources` via AI SDK annotation frame `8:[{type:"sources",sources:[...]}]` SEBELUM text.
6. Stream response LLM (Groq) dengan inline `[n]` markers.
7. Persist assistant message + `sources` (JSONB) → citation survive reload.

Source shape (annotation + `messages.sources`):

```json
{
  "index": 1,
  "documentId": "01HXXX...",
  "documentTitle": "Knowledge Base: Embeddings",
  "heading": "2. Cara Kerja pgvector",
  "snippet": "pgvector adalah extension Postgres…",
  "similarity": 0.72
}
```

Graceful degradation: kalau embedding/search gagal di mana pun, chat tetap jalan tanpa RAG (sources kosong, no citation).

## Hybrid Search + Rerank (Minggu 6)

Pipeline `service.Retriever` (lihat `internal/service/retriever.go`):

1. **Embed query** — Voyage `voyage-3-lite`, `input_type=query`
2. **Hybrid search** (`db.SearchHybrid`) — single SQL:
   - Vector top-20 via `<=>` (cosine, HNSW index `chunks_embedding_idx`)
   - BM25 top-20 via `ts_rank(content_tsv, websearch_to_tsquery('simple', $query))` (GIN index `chunks_content_tsv_idx`)
   - RRF combine: `score = Σ 1 / (60 + rank_i)`
   - FULL OUTER JOIN → preserve chunks yang cuma muncul di salah satu retriever
3. **Rerank** — Voyage `rerank-2` cross-encoder, top-K final
4. **Fallback** — kalau rerank gagal, return RRF results as-is (graceful degradation)

Dipakai oleh:
- `POST /chat` (chat handler RAG, threshold `rerankScore >= 0.10`)
- `POST /documents/search` (search UI; user bisa skip rerank dengan `noRerank: true` di body)

Tuning (`internal/handler/chat.go`):
```go
ragCandidateLimit      = 20   // per-retriever top-N untuk hybrid stage
ragTopK                = 5    // final top-K setelah rerank
ragSimilarityThreshold = 0.10 // rerank score di bawah ini = nggak relevan
```

## Tool Calling (Minggu 7)

`internal/tools` package implements OpenAI-compatible function calling buat Groq. 4 tools registered di main.go (Tavily optional):

| Tool | Args | Provider |
|---|---|---|
| `web_search` | `{ query, max_results? }` | Tavily AI |
| `fetch_url` | `{ url }` | net/http + html-to-markdown |
| `calculator` | `{ expression }` | expr-lang/expr |
| `get_current_time` | `{ timezone? }` | stdlib `time` |

Pipeline di `handler/chat.go`:
1. Stream service kirim request ke Groq dengan `tools: [...]`
2. Groq emit tool_call deltas via SSE (per-index, partial args)
3. Service accumulate ke `ToolCallRequest` list, emit `9:` frame (AI SDK protocol)
4. Handler execute via registry, emit `a:` frame untuk tiap result
5. Append assistant-with-tool_calls + tool turns ke message history
6. Loop sampai finish_reason == "stop" atau max 5 iterasi

Graceful: tool fail → return `{error: "..."}` ke model, model bisa adapt atau apologize. Registry empty → tools nggak di-send, chat behaves seperti sebelum Minggu 7.

## Coding Tools (Minggu 8)

Package `internal/workspace` provides per-user sandboxed filesystem area di `<WORKSPACE_ROOT>/<user_id>/`. Strict path validation: no abs, no `..`, verify via `filepath.Rel` defense-in-depth.

5 tools baru di `internal/tools/`:
| Tool | File |
|---|---|
| `read_file` | `read_file.go` — line range support |
| `write_file` | `write_file.go` — 1 MB cap |
| `list_directory` | `list_directory.go` — sorted, dirs first |
| `search_code` | `search_code.go` — regex walk, skip node_modules/.git/etc |
| `run_shell` | `run_shell.go` — allowlist + tokenizer rejecting shell meta |

Tool reads user ID from ctx via `workspace.UserFromContext(ctx)`. Chat handler inject via `ctx = workspace.WithUser(r.Context(), user.ID)` sebelum tool execution.

Allowlist (`run_shell`): `ls cat find grep wc head tail file du tree`, plus `git` dengan sub-allowlist `log status diff show branch ls-files blame`. NO destructive (rm/mv/install/curl/etc). NO shell expansion (`exec.Command` bukan `sh -c`).

Production deployment: Railway needs Volume mount di `/data` supaya workspace files persist. Tanpa volume, files hilang tiap deploy.

## Productivity Tools (Minggu 9)

11 new tools split jadi 3 groups:
- **Tasks** (internal DB, table `tasks` migration 006): create_task, list_tasks, complete_task, delete_task, remind_me. Plus REST `/tasks` CRUD untuk FE page.
- **Calendar** (Google Calendar API): list, create, update, delete events di primary calendar.
- **Gmail** (read-only): search_gmail, read_gmail_message.

Google access token forwarded dari FE Auth.js via JWT claim `google_access_token`. BE middleware extract via `appmw.GoogleTokenFromCtx(ctx)`. Helper `internal/tools/google_common.go` punya `googleTokenOrError`, `googleGET/POST/PATCH/DELETE` dengan Bearer auth + standard error parsing.

FE auth.ts handle access_token refresh via Google OAuth2 token endpoint (`https://oauth2.googleapis.com/token` with `grant_type=refresh_token`). Required: `access_type=offline + prompt=consent` di authorization params supaya refresh_token guaranteed di-issued.

## Long-term Memory (Minggu 10)

Per-user persistent facts disimpan di `user_memories` (migration 007) dengan vector(512) embedding pakai Voyage. 3 tools (`remember_this`, `update_memory`, `forget_memory`) + REST CRUD `/memories`.

Chat handler inject top-3 memory di system prompt sebelum RAG context — order: base prompt → memory → docs → tool rules. Threshold `0.20` (lebih rendah dari RAG karena memori lebih personal/varied). Skip embedding call kalau `CountByUser == 0` untuk hemat token.

Memory retrieval lebih sederhana dari RAG: cuma cosine vector (no BM25, no rerank). Memori pendek (1-2 kalimat per fact), sehingga vector cukup akurat.

## Observability + Evals (Minggu 11)

**Package baru**:
- `internal/tracing/` — Tracer struct dengan spans + counters. Thread-safe. Snapshot() untuk async persist.
- `internal/eval/retrieval.go` — RetrievalEvaluator: run golden queries → compute recall@k + MRR + per-query breakdown.
- `internal/eval/judge.go` — JudgeEvaluator: Groq LLM scores assistant response (faithfulness + helpfulness 1-5) dengan structured JSON output.

**Migration 008** — 3 tables: `chat_traces` (spans JSONB inline), `eval_sets` (queries JSONB), `eval_runs` (results JSONB).

**Chat handler** — di-instrument setiap stage dengan `tr.Start("stage").SetMeta(...).Finish()`. Async persist ke DB pas request selesai (goroutine + 5s timeout supaya nggak leak).

**Endpoints**:
- `GET /observability/traces?limit=` — recent traces per user
- `GET /observability/metrics?sample=` — computed metrics dari N recent traces
- `GET /eval-sets`, `POST /eval-sets`, `DELETE /eval-sets/{id}` — CRUD golden query sets
- `GET /eval-runs?kind=&evalSetId=` — list runs
- `POST /eval-runs/retrieval` — trigger retrieval eval (butuh evalSetId)
- `POST /eval-runs/judge` — trigger judge eval (butuh messageId)

Part of [PersonalChatAI-Aulia](../README.md) — Roadmap AI Engineer Minggu 11.
