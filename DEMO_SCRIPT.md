# Demo Video Script — Personal Chat AI

Target: 3-5 menit Loom video buat share ke recruiter + LinkedIn. Fokus ke **hasil dan keputusan teknis**, bukan step-by-step coding.

## Persiapan sebelum recording

**Setup browser:**
- Buka 3 tab, urutan penting:
  1. Landing page production (`https://personalchat.aulia.dev`) — belum sign in
  2. `/observability` — kosong dulu, refresh nanti setelah demo chat
  3. GitHub repo (README case study section terbuka)
- Zoom Chrome ke 110% biar text readable di video
- Close semua notifications (Slack, email, Do Not Disturb ON)
- Clean desktop background

**Setup data:**
- Sign in ke 1 demo account, tapi sign out sebelum recording
- Pre-upload 1-2 dokumen PDF ke `/documents` (misal CV atau paper singkat) supaya RAG demo cepat — nggak perlu wait upload+embed di depan kamera
- Siapkan 1 memory di `/memory` (misal: "User prefer response singkat max 3 paragraf")

**Tools:**
- Loom desktop app (bukan browser extension — quality lebih baik)
- Kalau ada, pakai external mic
- Rehearsal 1-2x sebelum record final

## Shot list & timing

Total ~4 menit. Timing per section adalah target, boleh sedikit flex.

### 0:00 – 0:20 · Intro (20 detik)

**Shot:** Landing page hero, face cam pojok kanan bawah

**Talking points:**
> "Hi, saya Aulia. Ini Personal Chat AI — portfolio project untuk transisi dari Frontend ke AI Engineer. Full-stack chat app dengan RAG, tool calling, long-term memory, dan built-in observability. Backend Go, frontend Next.js, deploy di Railway + Vercel. Live URL ada di deskripsi."

**Do:** Point ke tagline di landing page + scroll pelan sekali.
**Don't:** Skip intro. Recruiter perlu tau siapa kamu dan apa produknya dalam 20 detik pertama.

### 0:20 – 0:45 · Sign in + arsitektur singkat (25 detik)

**Shot:** Click "Sign in with Google" → OAuth flow → landed di `/chat`

**Talking points sambil OAuth loading:**
> "Auth pakai Google OAuth via Auth.js v5. Backend Go verify HS256 JWT dengan shared secret — jadi FE dan BE tetap decoupled tapi single sign-on. Google token juga di-forward buat scope Calendar + Gmail nanti."

**Do:** Sekilas tunjukin OAuth consent screen (Google account picker). Skip kalau lambat.

### 0:45 – 1:30 · RAG demo dengan dokumen (45 detik)

**Shot:** `/chat` — kirim pertanyaan yang match dokumen yang udah di-upload

**Prompt contoh (pilih salah satu):**
- "Berdasarkan CV yang aku upload, apa aja pengalaman kerja aku di frontend?"
- "Ringkas paper yang aku upload dalam 3 bullet."

**Talking points sambil streaming:**
> "Ini RAG dengan hybrid search — vector search pakai Voyage AI top 20, plus BM25 fulltext top 20, di-fuse pakai Reciprocal Rank Fusion, lalu rerank pakai Voyage rerank-2 buat ambil top 5. Citations `[1]`, `[2]` di response bisa di-hover buat lihat sumbernya."

**Do:**
- Hover salah satu citation → tooltip dengan snippet muncul
- Scroll ke Sources footer di bawah bubble → tunjukin file source

**Don't:** Baca full response bareng viewer. Cukup point ke citations + sources.

### 1:30 – 2:15 · Tool calling — coding assistant (45 detik)

**Shot:** New chat atau lanjutan — trigger tool

**Prompt:**
> "Bikin file utils/math.go isinya function Add(a, b int) int"

**Talking points sambil tool card muncul:**
> "24 tools total — web search, calculator, file operations, task management, Google Calendar, Gmail. LLM decide sendiri kapan pakai tool. Ini contoh multi-turn: pertama `write_file` bikin file, terus `list_directory` verify. Tool result balik ke LLM, LLM lanjut generate final answer. Semua per-user sandboxed — nggak bisa escape ke root filesystem."

**Do:**
- Expand tool invocation card → tunjukin args + result JSON
- Kalau punya waktu, follow-up prompt: "list file di workspace ku" — tunjukin file baru muncul

### 2:15 – 2:45 · Memory & tasks (30 detik)

**Shot:** Buka `/memory` di tab baru atau navigate

**Talking points:**
> "Long-term memory — separate embedding table. Setiap chat, top 3 memory paling relevan auto-inject ke system prompt. Jadi assistant inget preferensi tanpa harus re-tell tiap session."

**Do:** Tunjukin 1-2 memory entries yang udah di-add.

Cepet ke `/tasks` juga:
> "Task management bisa lewat UI atau lewat chat — LLM punya tool add/list/complete tasks."

### 2:45 – 3:30 · Observability tour (45 detik)

**Shot:** Buka `/observability` (harusnya udah ada trace dari demo tadi)

**Talking points:**
> "Yang bikin ini unik: full observability built-in. Setiap chat request di-trace. Cek trace dari demo barusan..."

**Do:**
- Click salah satu trace → detail muncul
- Tunjukin span breakdown: `retrieval` (X ms), `llm_stream` iter 0, `tool.write_file` (Y ms), `llm_stream` iter 1
- Point ke total latency + P50/P95 metrics kalau ada di dashboard

**Talking points lanjutan:**
> "Semua span persisted ke Postgres JSONB. Jadi bisa audit performance regression, cari slow query, atau debug tool failure tanpa external service kayak Datadog. Untuk portfolio scale ini cukup, dan kalau scale up gampang di-swap ke OpenTelemetry."

### 3:30 – 4:00 · Closing & call-to-action (30 detik)

**Shot:** Back to landing page atau GitHub README

**Talking points:**
> "Yang menarik buat aku dari build ini: bikin RAG pipeline yang genuinely berguna, debug streaming protocol AI SDK, dan build sandboxed coding tools yang aman. Detail tech decisions dan trade-offs ada di README case study — link di deskripsi. Kalau kamu hiring AI Engineer atau AI-fluent full-stack, boleh hubungin aku di [email/LinkedIn]. Thanks!"

**Do:** Slow scroll ke README case study section sebentar (biar viewer noted ada tech deep-dive).

## Recording tips

- **Talk slower than natural.** Recording bikin tempo terasa cepat — pelan-pelan aja.
- **Mistake handling.** Kalau salah ngomong, jangan restart — pause 2 detik, ulang kalimat, edit di post.
- **Voiceover kalau shy.** Rekam screen dulu tanpa suara, tambahin voiceover pakai Loom native tool.
- **Retake threshold.** Kalau ada bug atau tool gagal live, itu OK untuk retake — recruiter bakal notice.

## Post-processing

**Loom editing:**
- Trim awkward pauses (>2 detik silence)
- Add chapter markers di timestamps section headers di atas
- Cover thumbnail: screenshot landing page hero dengan text "Personal Chat AI by Aulia"

**Loom settings:**
- Video quality: HD 1080p
- Allow comments: ON (recruiter feedback)
- Password: NO (kamu mau discoverable)
- Custom URL slug: `personal-chat-ai-demo`

## Distribution checklist

Setelah video final:

- [ ] Loom URL di GitHub README (paling atas, next to Live Demo badge)
- [ ] Loom URL di LinkedIn post — hook: "Just shipped my portfolio project transitioning from FE to AI Engineer. 12 weeks, 8000+ LOC, RAG + 24 tools + observability. Full breakdown:"
- [ ] Loom URL di Twitter/X thread
- [ ] Link di CV / portfolio website
- [ ] Send ke 3-5 AI Engineer recruiter di target companies (personalized DM, jangan spray)

## Alternate: 90-detik hyper-short version

Kalau mau bikin versi pendek buat social media (attention span kebanyakan orang <2 menit):

- 0:00 – 0:10 · Intro + tagline
- 0:10 – 0:35 · Sign in + kirim 1 prompt dengan RAG citation
- 0:35 – 1:00 · 1 tool call demo (write_file)
- 1:00 – 1:20 · Buka `/observability`, tunjukin trace
- 1:20 – 1:30 · Closing + link

Buat dua-duanya kalau punya waktu — long version ke recruiter/hiring manager, short version ke social feed.
