# Deployment Guide

Step-by-step deploy Personal Chat AI ke production. Estimasi total waktu: ~90 menit kalau semua account sudah ada, ~2.5 jam kalau baru daftar semua service.

## Prerequisites

Setup sekali sebelum mulai. Semua free tier untuk portfolio scale.

| Service | Kegunaan | Setup |
|---|---|---|
| **Neon** | Postgres + pgvector | Signup di [console.neon.tech](https://console.neon.tech) — no credit card |
| **Groq** | Chat LLM | Signup di [console.groq.com/keys](https://console.groq.com/keys) — free rate limits generous |
| **Voyage AI** | Embeddings + rerank | Signup di [voyageai.com](https://www.voyageai.com) — 200M tokens/bulan free |
| **Tavily** (opsional) | Web search tool | Signup di [tavily.com](https://tavily.com) — 1000 searches/bulan free |
| **Google Cloud** | OAuth + Calendar + Gmail | [console.cloud.google.com](https://console.cloud.google.com) |
| **Railway** | Backend hosting | [railway.app](https://railway.app) — $5/mo credit free |
| **Vercel** | Frontend hosting | [vercel.com](https://vercel.com) — Hobby plan free |
| **Custom domain** (opsional) | Nice URL | Namecheap / Cloudflare / Google Domains |

## Step 1 — Generate shared secret (5 menit)

Backend dan frontend perlu shared `AUTH_SECRET` (HS256 JWT). Generate sekali, pakai di 2 tempat:

```bash
openssl rand -hex 32
# Output contoh: 3f9c8a1b2d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a
```

Simpen aman — bakal dipake di `backend/.env` + Railway env vars + Vercel env vars.

## Step 2 — Neon Postgres (10 menit)

1. Buat project baru di Neon Console
2. Region: **AWS ap-southeast-1** (Singapore) untuk latency terbaik dari Indonesia
3. Compute size: **Free tier** (0.25 vCPU, cukup untuk portfolio)
4. Copy connection string dari Dashboard → **Connection Details** → "Pooled connection"
   ```
   postgresql://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require
   ```
5. Verify pgvector: buka Neon SQL Editor, run:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   SELECT * FROM pg_extension WHERE extname = 'vector';
   ```
   Kalau muncul row, pgvector aktif.

Migrations bakal apply otomatis dari BE saat pertama kali deploy (via `make migrate-up`, atau manual step di Step 5).

## Step 3 — Google OAuth Client (15 menit)

Set up OAuth credentials + enable APIs yang di-pake.

### 3.1 Bikin OAuth Client

1. Buka [Google Cloud Console → APIs & Services → Credentials](https://console.cloud.google.com/apis/credentials)
2. **Create Credentials → OAuth client ID**
3. Application type: **Web application**
4. Name: `Personal Chat AI Production`
5. **Authorized JavaScript origins**:
   - `https://personalchat.aulia.dev` (ganti sama domain FE production kamu)
   - `http://localhost:3000` (untuk dev — sekalian, biar 1 client dipake dev + prod)
6. **Authorized redirect URIs**:
   - `https://personalchat.aulia.dev/api/auth/callback/google`
   - `http://localhost:3000/api/auth/callback/google`
7. Save → copy **Client ID** + **Client Secret**

### 3.2 Enable APIs

Di **APIs & Services → Library**, enable:
- Google Calendar API
- Gmail API

Tanpa enable ini, tools calendar/gmail akan gagal.

### 3.3 OAuth Consent Screen

**APIs & Services → OAuth consent screen**:
- User type: **External**
- App info: nama, support email
- **Scopes**: tambahin manual:
  - `https://www.googleapis.com/auth/calendar`
  - `https://www.googleapis.com/auth/gmail.readonly`
- **Test users**: tambahin email kamu sendiri (kalau app masih dalam "Testing" mode, cuma test users yang bisa sign in — 100 max)

> Warning: kalau kamu belum verify app, user consent screen akan show "unverified" warning. Untuk portfolio personal use ini OK.

## Step 4 — Backend deploy (Railway) — 30 menit

### 4.1 Setup Railway project

1. Buka [railway.app](https://railway.app) → **New Project → Deploy from GitHub repo**
2. Pilih repo `portofolio-ai-aulia`
3. **Root directory**: `backend` (penting! biar Dockerfile ke-pick)
4. Railway auto-detect Dockerfile

### 4.2 Environment variables

Di service Settings → Variables, tambahin:

```
GROQ_API_KEY=gsk_...              # dari console.groq.com/keys
VOYAGE_API_KEY=pa-...             # dari voyageai.com dashboard
TAVILY_API_KEY=tvly-...           # opsional; kalau kosong, web_search tool nggak di-register
DATABASE_URL=postgresql://...     # dari Neon (pooled connection, ?sslmode=require di akhir)
AUTH_SECRET=<hasil openssl rand>  # same value dengan FE Vercel
CORS_ORIGINS=https://personalchat.aulia.dev
ENV=production
# WORKSPACE_ROOT auto-set ke /data/workspaces di Dockerfile
```

Kalau nggak mau custom domain, `CORS_ORIGINS` bisa isi domain default Railway (`https://your-service-name.up.railway.app`).

### 4.3 Attach Volume (WAJIB untuk coding tools)

Coding tools (write_file, list_directory, dll) butuh persistent filesystem. Tanpa Volume, file hilang tiap deploy.

- Service Settings → **Volumes → New Volume**
- Mount path: `/data`
- Size: 1 GB (cukup untuk portfolio)

Volume free tier Railway sudah cover 1 GB.

### 4.4 Custom domain (opsional)

- Service Settings → **Networking → Custom Domain**
- Tambahin `api.personalchat.aulia.dev`
- Set CNAME di registrar kamu ke `<railway-generated>.up.railway.app`

### 4.5 First deploy + migrate

Deploy pertama otomatis jalan setelah env vars set. Setelah service up:

- Cek healthcheck: `curl https://api.personalchat.aulia.dev/healthz` → `{"ok":true}`
- Run migrations sekali dari Railway CLI atau SSH ke service:

  Option A — Railway CLI:
  ```bash
  railway link  # pilih project
  railway run make migrate-up
  ```

  Option B — SSH ke service:
  ```
  Railway UI → service → Deploy tab → "..." → SSH → run: make migrate-up
  ```

Log harus show 8 migrations applied. Verify di Neon SQL Editor:
```sql
SELECT count(*) FROM users;
SELECT count(*) FROM documents;
SELECT count(*) FROM user_memories;
```

Semua harus return 0 (table exist, empty).

## Step 5 — Frontend deploy (Vercel) — 20 menit

### 5.1 Setup Vercel project

1. [vercel.com](https://vercel.com) → **Add New → Project → Import GitHub repo**
2. Pilih repo yang sama (`portofolio-ai-aulia`)
3. Root directory: **`.`** (root — Next.js ada di root)
4. Build settings: auto-detected (Next.js). Framework preset: **Next.js**. Build command: `yarn build`. Output: `.next`.

### 5.2 Environment variables

Vercel Project Settings → Environment Variables:

```
NEXT_PUBLIC_API_BASE_URL=https://api.personalchat.aulia.dev  # BE URL dari Step 4
AUTH_GOOGLE_ID=<client_id_dari_step_3>
AUTH_GOOGLE_SECRET=<client_secret_dari_step_3>
AUTH_SECRET=<hasil_openssl_sama_dengan_BE>
AUTH_URL=https://personalchat.aulia.dev  # domain FE production
```

Environment scope: **Production, Preview, Development** untuk semua.

### 5.3 Custom domain (opsional)

- Vercel Project → **Settings → Domains**
- Add `personalchat.aulia.dev`
- Update DNS di registrar: CNAME atau A record sesuai Vercel instructions

### 5.4 Deploy + verify

- Push ke branch `main` → auto-deploy
- Buka domain → landing page publik muncul
- Sign in dengan Google account kamu (yang udah listed di Test users di Step 3.3)
- Setelah masuk `/chat`, coba kirim pesan → assistant harusnya reply

## Step 6 — Smoke test end-to-end (10 menit)

Verifikasi semua feature jalan di production:

- [ ] `GET /healthz` return `{"ok":true}`
- [ ] Landing page load, click Sign In → Google OAuth → redirect back ke `/chat`
- [ ] Chat kirim "Halo" → assistant reply streaming
- [ ] Upload PDF ke `/documents` → parse + embed sukses (cek chunk count)
- [ ] Chat tanya sesuatu yang match dokumen → citation `[1]` muncul + Sources footer
- [ ] Chat: "Berapa hasil sqrt(2)?" → tool card calculator muncul → result 1.414...
- [ ] Chat: "Bikin file test.txt isinya hello" → tool card write_file muncul → success
- [ ] Buka `/observability` → recent trace muncul dengan span breakdown
- [ ] `/tasks` — add task, mark done, delete
- [ ] `/memory` — add memory, search, delete

Kalau ada yang gagal, cek Railway logs (`railway logs` atau UI) — sekarang task handler + others udah proper structured logging.

## Common issues

### `429 Too Many Requests` di chat
Kamu spam. Rate limiter aktif per user: 20 burst + 0.5/sec refill. Tunggu ~40 detik atau tweak di `internal/middleware/ratelimit.go`.

### `unauthorized` di semua request
`AUTH_SECRET` beda antara FE Vercel dan BE Railway. Set ulang, redeploy keduanya.

### Google OAuth redirect_uri_mismatch
Redirect URI production belum di-add di Google Cloud Console (Step 3.1 poin 6). Add + tunggu 2-3 menit propagate.

### Tools calendar/gmail fail `google auth expired`
User perlu sign out + sign in ulang setelah tambah scopes (Step 3.3). Token existing nggak punya scope Calendar/Gmail.

### Rate limit dari Voyage / Groq / Tavily
Kamu hit free tier limit. Cek dashboard masing-masing untuk usage. Upgrade paid tier atau throttle client-side.

### Chat kirim `500 internal server error`
Cek Railway logs untuk stack trace. Common cause: `DATABASE_URL` invalid, atau migration belum apply. Verify Step 4.5.

### Landing page redirect loop
Session cookie invalid. Clear cookies + retry. Kalau persist, cek `AUTH_URL` env di Vercel sudah match domain kamu (bukan localhost).

---

## Cost estimate

Production personal use (1 user, ~200 chat/hari, 50 documents indexed):

| Service | Free tier | Estimated actual |
|---|---|---|
| Neon | 0.5 GB storage, 190 compute hours | Fit in free |
| Railway | $5/mo credit | ~$3-4/mo (small service + volume 1 GB) |
| Vercel | Unlimited bandwidth Hobby | Free |
| Groq | Rate-limited but generous | Free |
| Voyage | 200M tokens/bulan | Free (jauh di bawah) |
| Tavily | 1000 searches/bulan | Free |
| Google | Free untuk personal | Free |
| Domain | ~$10-15/year | Optional |

**Total ~$3-5/bulan** untuk full stack production. Kalau skip custom domain: free entirely.
