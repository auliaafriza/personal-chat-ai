# PersonalGPT

Chat assistant pribadi untuk dokumen, kode, dan produktivitas.

Architecture: **Next.js FE** ←→ **Go BE** ←→ **Neon Postgres**

```
portofolio-ai-aulia/
├── (frontend - Next.js 15 + TypeScript)
└── backend/    (Go service — chi + pgx + golang-migrate, Anthropic via net/http)
```

---

## Tech Stack

### Frontend (this folder)
- **Next.js 15** (App Router) + **TypeScript** strict
- **Vercel AI SDK** (`@ai-sdk/react` untuk `useChat` hook + streaming)
- **TanStack Query v5** (singleton, di root)
- **axios** (satu instance per backend service)
- **Tailwind CSS** + **shadcn/ui** (Radix primitives)
- **react-hook-form** + **zod** (schema-first forms)
- **sonner** (toast)
- **Husky** + **commitlint** (Conventional Commits)

### Backend (`backend/`)
- **Go 1.23** + **chi** router
- **pgx** (native Postgres driver, no ORM magic)
- **golang-migrate** (SQL migrations)
- **Anthropic Messages API** via raw `net/http` + SSE parsing (streaming chat + Haiku title gen)
- Implements **Vercel AI SDK data stream protocol** sehingga FE pakai `useChat` tanpa perubahan

Mengikuti yang diadaptasi (shadcn/ui menggantikan `@edot/sdk-ui-react`, BE Go terpisah biar "one axios instance per backend service" beneran).

## Quick Start

Butuh **2 terminal** — satu untuk BE, satu untuk FE.

### 1. Backend setup (sekali aja)

```bash
cd backend
cp .env.example .env
# Edit .env — isi:
#   ANTHROPIC_API_KEY (https://console.anthropic.com/settings/keys)
#   DATABASE_URL     (Neon connection string — lihat "Database Setup" di bawah)

go mod download
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

make migrate-up   # apply schema ke Neon
make run          # listen di :8080
```

### 2. Frontend setup (di folder root, terminal lain)

```bash
yarn install
cp .env.example .env.local
# Edit .env.local — defaultnya pointing ke http://localhost:8080
yarn setup:hooks  # init git + husky (sekali aja)
yarn dev          # listen di :3000
```

Buka [http://localhost:3000](http://localhost:3000) — auto-redirect ke `/chat`.

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
│   ├── Sidebar.tsx                            # list + new chat button
│   └── ConversationItem.tsx                   # rename/delete dropdown
├── features/chat/
│   ├── pages/ChatPage.tsx                     # streaming + history + title gen
│   ├── components/                            # ChatBubble, MessageList, ChatInput
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
├── lib/
│   ├── utils.ts                               # cn() helper
│   └── types.ts                               # ApiResponse envelope
└── providers/
    ├── QueryProvider.tsx                      # TanStack Query (singleton)
    └── ThemeProvider.tsx                      # next-themes
```

### Backend (`backend/`)

Lihat [backend/README.md](./backend/README.md) untuk struktur lengkap.

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
---

Built by Aulia Afriza · 2026
