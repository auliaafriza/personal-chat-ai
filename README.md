# Personal Chat AI by Aulia

Chat assistant pribadi untuk dokumen, kode, dan produktivitas.
Bagian dari [Roadmap AI Engineer](../) — proyek yang bertumbuh tiap minggu.

Architecture: **Next.js FE (Auth.js v5)** ←→ **Go BE (JWT-protected)** ←→ **Neon Postgres**

```
portofolio-ai-aulia/
├── (frontend - Next.js 15 + TypeScript + Auth.js v5 + Google OAuth)
└── backend/    (Go service — chi + pgx + golang-migrate + golang-jwt, Groq via net/http)
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
- **golang-migrate** (SQL migrations)
- **golang-jwt/jwt/v5** — HS256 validation (shared `AUTH_SECRET` dengan FE)
- **Groq Chat Completions API** (OpenAI-compatible) via raw `net/http` + SSE
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
#   GROQ_API_KEY    (https://console.groq.com/keys — free tier OK)
#   DATABASE_URL    (Neon connection string — lihat "Database Setup" di bawah)
#   AUTH_SECRET     (paste hasil openssl di atas)

go mod download
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

make migrate-up   # apply 001_initial + 002_add_users ke Neon
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
- [x] **Minggu 3** — Auth (Auth.js v5 + Google OAuth + JWT shared secret) + Settings page + dark mode + Cmd+K/Cmd+/ + mobile responsive ← **kamu di sini**
- [ ] **Minggu 4** — Embeddings + pgvector
- [ ] **Minggu 5** — RAG end-to-end + citation
- [ ] **Minggu 6** — Hybrid search + reranking
- [ ] **Minggu 7** — Tool calling (web search, fetch URL)
- [ ] **Minggu 8** — Coding assistant tools
- [ ] **Minggu 9** — Productivity tools (Calendar, Task)
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

## What's Next (Minggu 4 — Embeddings & pgvector)

1. Tambah `pgvector` extension di Neon
2. Migrate `documents`, `document_chunks` tables (chunk + embedding kolom)
3. Document upload endpoint di Go BE
4. Generate embeddings (Voyage / OpenAI / Cohere) saat upload
5. Vector search via cosine similarity untuk RAG

Detail lengkap di [Roadmap doc](https://docs.google.com/document/d/1yNJwtVLvIDWOd37nubd3-IQaeSPgBmbin-lANCMnh28/edit).

---

Built by Aulia Afriza · 2026
