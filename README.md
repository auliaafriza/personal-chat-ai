# Personal Chat AI by Aulia

Chat assistant pribadi untuk dokumen, kode, dan produktivitas.
Bagian dari [Roadmap AI Engineer](../) — proyek yang bertumbuh tiap minggu.

**Status: Minggu 9 — Productivity tools (Calendar + Gmail + Tasks)**

Architecture: **Next.js FE (Auth.js v5)** ←→ **Go BE (JWT-protected, RAG)** ←→ **Neon Postgres (pgvector)**

```
portofolio-ai-aulia/
├── (frontend - Next.js 15 + TypeScript + Auth.js v5 + Google OAuth)
└── backend/    (Go service — chi + pgx + pgvector + golang-jwt, Groq + Voyage AI via net/http)
```

---

## Tech Stack

### Frontend (this folder)
- **Next.js 15** (App Router) + **TypeScript** strict
- **Vercel AI SDK** (`@ai-sdk/react` untuk `useChat` hook + streaming)
- **Auth.js v5** (NextAuth) — Google OAuth + HS256 JWT mint untuk BE
- **TanStack Query v5** (singleton, di root)
- **axios** (satu instance per backend service, Bearer token via interceptor)
- **Tailwind CSS** + **shadcn/ui** (Radix primitives)
- **react-hook-form** + **zod** (schema-first forms)
- **sonner** (toast)
- **next-themes** (dark mode)
- **Husky** + **commitlint** (Conventional Commits)

### Backend (`backend/`)
- **Go 1.23** + **chi** router
- **pgx** (native Postgres driver, no ORM magic)
- **pgvector** (Neon extension) — vector(512) + HNSW cosine index
- **golang-migrate** (SQL migrations)
- **golang-jwt/jwt/v5** — HS256 validation (shared `AUTH_SECRET` dengan FE)
- **Groq Chat Completions API** (OpenAI-compatible) via raw `net/http` + SSE — chat
- **Voyage AI** (voyage-3-lite, 512 dim) via raw `net/http` — embeddings (free 200M tokens/bulan)
- **ledongthuc/pdf** — PDF text extraction (pure Go, no CGO)
- DOCX parsing via stdlib `archive/zip` + `encoding/xml` (no extra dep)
- Implements **Vercel AI SDK data stream protocol** sehingga FE pakai `useChat` tanpa perubahan

Mengikuti yang diadaptasi (shadcn/ui menggantikan `@edot/sdk-ui-react`, BE Go terpisah biar "one axios instance per backend service" beneran).

## Quick Start

Butuh **2 terminal** — satu untuk BE, satu untuk FE. Plus setup OAuth + shared secret sekali aja.

### 1. Generate `AUTH_SECRET` (sekali aja)

```bash
openssl rand -hex 32
# Copy output — paste ke FE .env.local DAN backend/.env (HARUS sama).
```

### 2. Google OAuth client (sekali aja)

1. Buka [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials)
2. **Create credentials → OAuth client ID**
3. Application type: **Web application**
4. Authorized redirect URI: `http://localhost:3000/api/auth/callback/google`
5. Copy **Client ID** + **Client Secret** → paste ke FE `.env.local`

### 3. Backend setup (sekali aja)

```bash
cd backend
cp .env.example .env
# Edit .env — isi:
#   GROQ_API_KEY     (https://console.groq.com/keys — free)
#   VOYAGE_API_KEY   (https://www.voyageai.com — free 200M tokens/bulan)
#   DATABASE_URL     (Neon connection string — lihat "Database Setup" di bawah)
#   AUTH_SECRET      (paste hasil openssl di atas)

go mod download
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

make migrate-up   # apply 001 + 002 + 003 (pgvector + documents) ke Neon
make run          # listen di :8080
```

### 4. Frontend setup (di folder root, terminal lain)

```bash
yarn install
cp .env.example .env.local
# Edit .env.local — isi:
#   NEXT_PUBLIC_API_BASE_URL  (default http://localhost:8080)
#   AUTH_GOOGLE_ID            (dari Google Cloud Console)
#   AUTH_GOOGLE_SECRET        (dari Google Cloud Console)
#   AUTH_SECRET               (paste hasil openssl — sama dengan backend/.env)
yarn setup:hooks  # init git + husky (sekali aja)
yarn dev          # listen di :3000
```

Buka [http://localhost:3000](http://localhost:3000) → redirect ke `/signin` → sign in dengan Google → masuk `/chat`.

## Database Setup (Neon)

1. Daftar di [console.neon.tech](https://console.neon.tech) (free, no credit card).
2. Buat project baru → copy **connection string** (format `postgresql://...?sslmode=require`).
3. Paste ke `backend/.env` sebagai `DATABASE_URL=...`.
4. Run migrations:
   ```bash
   cd backend
   make migrate-up
   ```

**Update schema:** edit SQL di `backend/internal/db/migrations/` atau bikin migration baru:
```bash
cd backend
make migrate-create name=add_users      # bikin file *.up.sql + *.down.sql
# edit kedua file
make migrate-up
```

## Scripts (Frontend)

| Command | Action |
|---|---|
| `yarn dev` | Run dev server (port 3000) |
| `yarn build` | Production build (`output: standalone`) |
| `yarn start` | Run production build |
| `yarn lint` | ESLint |
| `yarn type-check` | TypeScript only (no emit) |
| `yarn format` | Prettier write |
| `yarn setup:hooks` | Init git + husky hooks (run once) |

## Scripts (Backend)

Run dari folder `backend/`:

| Command | Action |
|---|---|
| `make run` | Run dev server (port 8080) |
| `make build` | Build binary ke `bin/server` |
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create name=foo` | Create new migration file pair |
| `make help` | Lihat semua target |

## Folder Structure

### Frontend (`src/`)
```
src/
├── api/apiApp.ts                              # axios instance → Go BE
├── app/                                       # App Router (thin shells only)
│   ├── (chat)/
│   │   ├── layout.tsx                         # Sidebar + main panel
│   │   ├── chat/page.tsx                      # /chat (empty / new chat state)
│   │   └── chat/[conversationId]/page.tsx     # /chat/abc — load history
│   ├── layout.tsx                             # providers + Toaster
│   ├── globals.css                            # Tailwind + design tokens
│   ├── page.tsx                               # redirect → /chat
│   ├── not-found.tsx
│   └── global-error.tsx
├── components/layout/
│   ├── Sidebar.tsx                            # list + new chat + user menu
│   ├── MobileSidebar.tsx                      # Radix Dialog drawer (mobile only)
│   ├── UserMenu.tsx                           # avatar dropdown — settings, theme, signout
│   ├── ThemeToggle.tsx                        # next-themes light/dark button
│   └── ConversationItem.tsx                   # rename/delete dropdown
├── features/chat/
│   ├── pages/ChatPage.tsx                     # streaming + history + title gen
│   ├── components/                            # ChatBubble, MessageList, ChatInput
│   ├── hooks/useChatShortcuts.ts              # Cmd+K / Cmd+/ keyboard shortcuts
│   ├── services/
│   │   ├── query-keys.ts                      # typed const, never inline
│   │   ├── conversation/
│   │   │   ├── list/get.ts                    # useGetConversations
│   │   │   ├── detail/get.ts                  # useGetConversation
│   │   │   ├── post.ts                        # useMutationCreateConversation
│   │   │   ├── patch.ts                       # useMutationUpdateConversation
│   │   │   └── delete.ts                      # useMutationDeleteConversation
│   │   ├── message/list/get.ts                # useGetMessages
│   │   └── title/post.ts                      # useMutationGenerateTitle
│   ├── types.ts                               # Zod schemas (form validation)
│   ├── types/api.ts                           # type defs matching Go BE
│   └── constants.ts                           # models, defaults
├── features/settings/                         # Settings page (Minggu 3)
│   ├── pages/SettingsPage.tsx
│   ├── services/me/{get,put}.ts               # GET /me, PUT /me/settings
│   ├── types.ts                               # Zod form schema
│   ├── types/api.ts                           # User type
│   └── constants.ts                           # AVAILABLE_MODELS, temperature bounds
├── features/documents/                        # Documents page (Minggu 4)
│   ├── pages/DocumentsPage.tsx
│   ├── components/{UploadCard,DocumentList,SearchTool}.tsx
│   ├── services/{list/get,detail/get,post,delete,search/post}.ts
│   ├── types.ts                               # Zod form schemas
│   ├── types/api.ts                           # Document, DocumentChunk, SearchResult
│   └── constants.ts                           # Accepted formats, max size, topK bounds
├── app/
│   ├── signin/page.tsx                        # Google sign-in
│   ├── api/auth/[...nextauth]/route.ts        # Auth.js handlers
│   └── api/token/route.ts                     # Mint HS256 JWT untuk Go BE
├── lib/
│   ├── utils.ts                               # cn() helper
│   └── types.ts                               # ApiResponse envelope
└── providers/
    ├── QueryProvider.tsx                      # TanStack Query (singleton)
    ├── SessionProvider.tsx                    # next-auth/react SessionProvider
    └── ThemeProvider.tsx                      # next-themes
```

Top-level (di luar `src/`):
- `auth.ts` — Auth.js v5 NextAuth config (Google provider, jwt/session callbacks)
- `middleware.ts` — protect routes; redirect ke `/signin` kalau no session

### Backend (`backend/`)

Lihat [backend/README.md](./backend/README.md) untuk struktur lengkap.

## Roadmap (12 Minggu)

- [x] **Minggu 1** — Setup + streaming chat
- [x] **Minggu 2** — Persistence (Go BE + Neon Postgres + pgx) + multi-conversation + auto-title
- [x] **Minggu 3** — Auth (Auth.js v5 + Google OAuth + JWT shared secret) + Settings page + dark mode + Cmd+K/Cmd+/ + mobile responsive
- [x] **Minggu 4** — Embeddings (Voyage AI voyage-3-lite, 512 dim) + pgvector + document upload (txt/md/pdf/docx + paste) + similarity search UI
- [x] **Minggu 5** — RAG end-to-end (auto-retrieve global) + inline citation [n] + Sources footer + persisted citations
- [x] **Minggu 6** — Hybrid search (vector + BM25 RRF) + Voyage rerank-2 cross-encoder
- [x] **Minggu 7** — Tool calling — web_search (Tavily), fetch_url (html→markdown), calculator (expr), get_current_time + multi-turn loop
- [x] **Minggu 8** — Coding assistant tools — read_file, write_file, list_directory, search_code, run_shell (allowlist) + per-user workspace sandbox + syntax highlight + diff viewer
- [x] **Minggu 9** — Productivity tools — Google Calendar (list/create/update/delete) + Gmail (search/read) + Tasks CRUD + remind_me + /tasks page ← **kamu di sini**
- [ ] **Minggu 10** — Long-term memory
- [ ] **Minggu 11** — Evals + observability
- [ ] **Minggu 12** — Polish, security, showcase

## Conventions

**Commits** ikut [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(chat): add streaming response with useChat
fix(api): handle 401 from Anthropic with retry
chore: bump pgx to v5.7.2
```

**Code rules:**

- Business logic di `features/<feature>/pages/` atau `services/` — never di `app/**/page.tsx` (thin shell only)
- Query keys dari typed `queryKeys` object — never inline string
- Forms: schema-first dengan zod, `mode: "onChange"`
- Toast on both branches: `toast.success` & `toast.error` — never silent
- `cn()` untuk class composition — never `+` concat
- No `any`, no `console.log` (pakai `console.warn` / `console.error`)
- Provider dengan error guard di hook (throw kalau dipakai di luar provider)

## Auth Flow (Minggu 3)

```
Browser ──1. Sign in──▶ Auth.js v5 (FE) ──2. OAuth──▶ Google
                              │
                              │ 3. Profile (sub, email, name, picture)
                              ▼
                       FE /api/token
                       (sign HS256 JWT pakai AUTH_SECRET)
                              │
                              │ 4. Bearer JWT
                              ▼
                       Go BE: middleware/auth.go
                       (validate HS256 + upsert user + inject ke ctx)
                              │
                              ▼
                       Handler (user-scoped queries)
```

Token lifetime 30 menit; FE auto-refresh on 401 (lihat `src/api/apiToken.ts`).

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Cmd/Ctrl + K` | New chat (atau focus input kalau lagi di `/chat`) |
| `Cmd/Ctrl + /` | Focus input |
| `Enter` | Kirim pesan |
| `Shift + Enter` | Baris baru |

## Embeddings & RAG Setup (Minggu 4)

### Voyage AI

1. Daftar di [voyageai.com](https://www.voyageai.com) → no credit card required.
2. Dashboard → **API Keys** → create key → paste ke `backend/.env` sebagai `VOYAGE_API_KEY=pa-...`
3. Free tier: 200M tokens/bulan untuk `voyage-3-lite`. Setiap chunk ~250-500 tokens, jadi 200M tokens ≈ 500k+ chunks. Praktis nggak akan habis untuk personal use.

### pgvector

Neon free tier auto-include pgvector extension. Migration `003_add_documents.up.sql` jalanin:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

Kalau pakai Postgres lokal, install dulu via [pgvector repo](https://github.com/pgvector/pgvector). Mac:
```bash
brew install pgvector
```

### Document flow

```
File upload (.txt|.md|.pdf|.docx) atau paste text
  ↓ service.Parse  (dispatch by extension)
Plain text
  ↓ service.Chunk  (heading-aware + fallback fixed-size 1500 chars, 100 overlap)
[]Chunk
  ↓ Voyage AI POST /v1/embeddings  (input_type=document, batched 128)
[][]float32  (512-dim per chunk)
  ↓ tx: INSERT documents + batch INSERT document_chunks (embedding::vector)
Neon pgvector
```

Search:
```
Query text
  ↓ Voyage AI  (input_type=query)
[]float32
  ↓ ORDER BY embedding <=> $query_vec ASC LIMIT topK   (HNSW index, cosine distance)
[]SearchResult  (chunk + documentTitle + similarity)
```

## RAG Flow (Minggu 5)

Auto-RAG aktif kalau user punya ≥1 dokumen. Setiap pesan:

```
User message
  ↓ chat handler: CountChunksByUser > 0 ?
  ↓ ya → embed query (Voyage, input_type=query)
  ↓ SearchSimilar top-5 (cosine, HNSW)
  ↓ filter similarity ≥ 0.30
  ↓ build context block + citation instruction → augment system prompt
  ↓ kirim sources via AI SDK annotation frame (8:)   ──→ FE: "Membaca N dokumen…" + Sources footer
  ↓ stream LLM response (Groq) dengan inline [n] markers
  ↓ save assistant message + sources (JSONB) → citations survive reload
```

Tuning ada di `backend/internal/handler/chat.go`:
- `ragTopK = 5` — jumlah chunk yang di-retrieve
- `ragSimilarityThreshold = 0.30` — di bawah ini chunk diabaikan (anti-noise untuk chit-chat)
- `ragSnippetMaxChars = 300` — panjang snippet di Sources footer

Citation di-render dua jalur yang sama (lihat `src/features/chat/lib/sources.ts`):
- **Live stream**: BE kirim `8:[{type:"sources",...}]` → AI SDK `message.annotations`
- **History**: sources di-load dari DB → di-inject ke `annotations` saat hydrate

## Retrieval Pipeline (Minggu 6)

Setiap chat message (kalau user punya dokumen) + setiap `/documents/search` masuk pipeline yang sama:

```
Query
  ↓ embed via Voyage voyage-3-lite (input_type=query)
  ↓ ┌─────────────────────────────────────────────────────────┐
    │ Vector top-20    │  BM25 top-20                         │
    │ pgvector <=>     │  Postgres ts_rank + GIN index        │
    │ (HNSW, cosine)   │  (config 'simple', websearch_to_tsquery) │
    └────────┬─────────┴────────┬────────────────────────────┘
             ↓ RRF combine (k=60) — single SQL dengan FULL OUTER JOIN
  ↓ top-20 unique candidates (sorted by RRF)
  ↓ Voyage rerank-2 (cross-encoder, more accurate but slower)
  ↓ top-5 final + relevance scores
  ↓ inject ke prompt (RAG) atau return ke search UI
```

**Kenapa pipeline ini**:
- **Vector** = semantic similarity (paham sinonim, paraphrase)
- **BM25** = exact term matching (akurat untuk nama, ID, kode, term unik)
- **RRF** = combine ranking tanpa perlu normalize skor — parameter-free, robust
- **Rerank** = cross-encoder lihat query+chunk bareng, jauh lebih akurat dari bi-encoder (embedding) untuk decide relevance

**Graceful degradation**: Rerank gagal? Fallback ke RRF results. Search gagal? Chat tetap jalan tanpa RAG.

## Tool Calling (Minggu 7)

Tools yang tersedia:

| Tool | Description | Provider / lib |
|---|---|---|
| `web_search` | Cari di web, return top-N results dengan title/url/snippet | Tavily AI (free 1k/bulan) |
| `fetch_url` | Download HTML, convert ke markdown | `JohannesKaufmann/html-to-markdown` |
| `calculator` | Math expression eval (sqrt, pow, sin, dll) | `expr-lang/expr` (sandboxed) |
| `get_current_time` | Real-time clock + timezone | `time.LoadLocation` |

Pipeline multi-turn (max 5 iter):

```
User message
  ↓ chat handler: build initial turns + Tools=registry.Schemas()
  ↓ loop:
    │ Groq stream → text deltas + tool_call deltas
    │ kalau finish_reason == "tool_calls":
    │    for tc in tool_calls:
    │       result = registry.Run(tc.name, tc.args)
    │       emit ToolResult frame (a:)
    │       append assistant-with-tool_calls turn + tool result turn
    │    continue
    │ else: break (finish_reason == "stop")
  ↓ persist final assistant message
```

**Frames yang dikirim** (Vercel AI SDK protocol):
- `9:` tool_call — `{toolCallId, toolName, args}` — saat model decide call tool
- `a:` tool_result — `{toolCallId, result}` — setelah tool execution selesai

FE pakai `useChat` yang otomatis populate `message.toolInvocations` dari frames itu, lalu `ToolInvocationCard` render dengan icon + collapsible args/result.

**Tavily setup**: signup gratis di [tavily.com](https://tavily.com) (no credit card) → API key ke `TAVILY_API_KEY` di `backend/.env`. Kalau kosong, web_search tool nggak di-register (3 tools lain tetap jalan).

## Coding Tools (Minggu 8)

5 tools baru sandboxed ke `<WORKSPACE_ROOT>/<user_id>/`:

| Tool | Description |
|---|---|
| `read_file` | Baca file (max 200 KB) dengan optional `line_start`/`line_end` |
| `write_file` | Create/overwrite (max 1 MB), auto-create parent dirs |
| `list_directory` | List entries (1-level, sorted dirs first) |
| `search_code` | Regex match across files; skip node_modules/.git/dist/etc; max 200 hits |
| `run_shell` | Allowlist read-only: ls/cat/find/grep/wc/head/tail/file/du/tree + git (log/status/diff/show/branch/ls-files/blame). NO shell expansion |

**Security**:
- Path validation: reject absolute, reject `..`, verify resolved path stays inside user dir via `filepath.Rel` defense-in-depth.
- Shell: pakai `exec.Command` (NOT `sh -c`) — no shell expansion. Tokenizer reject `& | ; < > $ ( ) { } \``. Timeout 15s, output cap 50 KB.
- File ops: read 200 KB cap, write 1 MB cap.
- User ID di-inject via `workspace.WithUser(ctx, user.ID)` di chat handler — tools cek `workspace.UserFromContext(ctx)` sebelum touch filesystem.

**FE rendering**:
- Code blocks dapat syntax highlight (Prism via `react-syntax-highlighter`) + copy button.
- `ToolInvocationCard` render per-tool: `read_file` jadi CodeBlock, `list_directory` jadi tree icons, `search_code` jadi `path:line` hits, `run_shell` jadi terminal stdout/stderr.
- `DiffViewer` component tersedia (lazy-loaded `react-diff-viewer-continued`) untuk render diff.

**Workspace location**:
- Local dev: `./tmp/workspaces/` (relative ke `backend/`)
- Production (Railway): `/data/workspaces` — attach Volume di Railway service supaya file persist across deploy.

## Productivity Tools (Minggu 9)

**11 tools baru**:

| Group | Tool | Description |
|---|---|---|
| Tasks (internal DB) | `create_task` | Title + optional due_date |
| | `list_tasks` | Filter by status (pending/completed) atau due (overdue/today/upcoming/no_due) |
| | `complete_task` | Mark done |
| | `delete_task` | Permanent delete |
| | `remind_me` | Shortcut: create task dengan is_reminder=true + required due_date |
| Calendar (Google) | `list_calendar_events` | Range default 7 hari ke depan |
| | `create_calendar_event` | summary + start + end + optional location/description |
| | `update_calendar_event` | Partial update |
| | `delete_calendar_event` | Permanent |
| Gmail (Google, read-only) | `search_gmail` | Gmail search syntax (e.g. `from:bob@x.com is:unread`) |
| | `read_gmail_message` | Full body extraction (text/plain preferred, fallback strip-HTML) |

**OAuth scopes baru**:
- `https://www.googleapis.com/auth/calendar` (Calendar read+write)
- `https://www.googleapis.com/auth/gmail.readonly` (Gmail read-only)

User HARUS sign out + sign in again setelah upgrade ini (Google OAuth re-consent dialog akan muncul).

**Access token flow**:
1. Auth.js v5 `jwt` callback simpan access_token + refresh_token + expires_at
2. Auto-refresh kalau token expiring (1 min buffer) — Google OAuth2 token endpoint
3. `/api/token` (FE) include `google_access_token` claim di HS256 JWT untuk BE
4. BE `middleware/auth.go` extract token, inject ke ctx via `GoogleTokenFromCtx`
5. Tools Calendar/Gmail pakai `googleTokenOrError(ctx, "calendar")` — error explicit kalau missing

**/tasks page**:
- Filter status + due (chips toggle)
- Inline form: title + datetime-local for due_date
- Complete checkbox per item
- Delete with confirm
- Auto-invalidate query setelah mutate

## What's Next (Minggu 10 — Long-term memory)

1. Conversation summarization saat conversation panjang
2. User profile facts extraction (preferences, recurring themes)
3. Memory retrieval di chat context — selevel di atas RAG (memory dulu, baru documents)
4. Memory UI untuk lihat/edit/delete

Detail lengkap di [Roadmap doc](https://docs.google.com/document/d/1yNJwtVLvIDWOd37nubd3-IQaeSPgBmbin-lANCMnh28/edit).

---

Built by Aulia Afriza · 2026
