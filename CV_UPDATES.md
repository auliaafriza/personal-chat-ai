# CV + LinkedIn Updates — Recent Projects Section

Ready-to-paste text untuk CV, LinkedIn Experience, dan portfolio site. Update `<PLACEHOLDER>` sesuai actual data kamu.

## Recent Projects — 1 paragraf (CV version)

**PersonalChatAI — Full-stack AI Chat Application** *(2026)*

Built a production-ready AI chat platform from scratch in 12 weeks, transitioning from Frontend to AI Engineering. Full-stack Next.js 15 + Go with hybrid RAG (vector + BM25 + Voyage rerank), 24 integrated tools (file operations, Google Calendar, Gmail, web search), long-term memory with per-user embeddings, and built-in observability (P50/P95 metrics + JSONB span traces). Deployed to production on Vercel + Railway with 45% retrieval recall improvement over vector-only baseline (0.62 → 0.90 recall@5). Cost-optimized to $3-5/month. Stack: Next.js 15 App Router, Go 1.25, Neon Postgres + pgvector, Voyage AI, Groq Llama 3.3 70B, Auth.js v5.

*Live: [personalchat.aulia.dev](https://personalchat.aulia.dev)* · *Code: [github.com/aulia/personal-chat-ai](https://github.com/aulia/personal-chat-ai)*

## Recent Projects — 3 bullet (CV compact version)

**PersonalChatAI — AI Chat with RAG + Tools** *(2026)* — [Live](https://personalchat.aulia.dev) · [Code](https://github.com/aulia/personal-chat-ai)
- Built full-stack AI chat platform in 12 weeks: Next.js 15 + Go backend, RAG with hybrid search (vector + BM25 + rerank), 24 tools including sandboxed code execution and Google Calendar/Gmail integration
- Achieved 45% retrieval improvement (recall@5: 0.62 → 0.90) via hybrid search + Voyage rerank; benchmarked with 50-query eval set + LLM-as-judge quality scoring
- Deployed to production (Vercel + Railway) at ~$3-5/month; instrumented with in-DB observability (JSONB spans + P50/P95 metrics) enabling sub-minute debug of any request

## LinkedIn Experience — Full section (paste to Experience)

**AI Engineering Project — Full-Stack AI Application**
*Personal Project*
*January 2026 – April 2026 · Remote*

Self-directed 12-week roadmap transitioning from Frontend Engineer to AI Engineer role. Designed, built, and deployed production-grade AI chat platform end-to-end.

**Highlights:**
- Architected full-stack app: Next.js 15 App Router frontend + Go backend with chi router, pgx (no ORM), and 8 database migrations
- Implemented hybrid RAG pipeline: vector search (pgvector HNSW) + BM25 (Postgres tsvector) + Reciprocal Rank Fusion + Voyage AI rerank-2; benchmarked 45% recall improvement vs baseline
- Built 24-tool ecosystem: web search, calculator, sandboxed file operations, shell execution, Google Calendar/Gmail integration, task management, long-term memory with auto-inject
- Designed observability layer from scratch: request tracing with span breakdown persisted to Postgres JSONB, dashboard for latency P50/P95 metrics
- Shipped security-hardened: per-user workspace sandbox (3-layer path validation), rate limiter (token bucket), CSP/HSTS headers, JWT shared-secret between FE + BE
- Documented as recruiter-ready case study: architecture diagram, tech decisions with trade-offs, cost analysis ($3-5/month sustainable)

**Tech stack:** Next.js 15, TypeScript, Vercel AI SDK, Auth.js v5, Go 1.25, chi, pgx, Neon Postgres, pgvector, Voyage AI (embed + rerank), Groq Llama 3.3 70B, Tavily, Docker, Railway, Vercel.

**Impact:**
- 8,000+ lines of production-quality code (frontend + backend + migrations + tests)
- 3 blog posts on architecture + technical deep-dive
- 4-minute Loom demo video
- Full case study with 10 tech decision trade-offs documented

## LinkedIn Headline options

Pick one atau A/B test:

**Option A — role-focused:**
> AI Engineer building production RAG + agent systems | Full-stack Next.js + Go | Ex-Frontend Engineer

**Option B — narrative:**
> Frontend Engineer transitioning to AI Engineering | Just shipped 12-week portfolio: RAG + 24 tools + observability | Open to opportunities

**Option C — technical:**
> Building AI applications in production | RAG, agents, LLM ops | Next.js + Go + Postgres pgvector

## LinkedIn About section (rewrite)

Aku engineer yang habis transisi dari Frontend ke AI Engineering. 3 tahun pengalaman FE (React, Next.js, TypeScript) dan sekarang fokus bangun AI applications yang production-ready — bukan cuma demo.

Baru ship portfolio project 12-week: **Personal Chat AI** — full-stack chat app dengan RAG, 24 tools, long-term memory, dan observability built-in. Live di [personalchat.aulia.dev](https://personalchat.aulia.dev).

Yang aku enjoy dari AI Engineering: kombinasi product thinking (kapan RAG worth, kapan skip), infrastructure (streaming protocol, retrieval optimization, cost management), dan systems design (fail-safe pipeline, observability, security). Semua aku pelajari dengan cara build sendiri dari nol, dokumentasikan trade-off, dan iterate berdasarkan metric.

**Sedang looking for:** AI Engineer / AI-fluent Full-Stack roles di companies yang shipping AI product (not just experimenting). Remote-friendly, timezone WIB atau overlap dengan APAC.

**Tech saya sering pakai:** Next.js, TypeScript, Go, Postgres + pgvector, Voyage AI, Groq, LangGraph (upcoming project).

**Konten yang aku share di sini:**
- Reflective post soal keputusan technical + trade-off
- Deep dive technical topics (retrieval, streaming protocol, agent design)
- Career transition experience — untuk engineer lain yang lagi mikir hal serupa

DM open untuk role opportunity, coffee chat, atau kalau kamu punya feedback soal portfolio.

## GitHub Profile README template

Simpen di `github.com/aulia/aulia/README.md` (special repo untuk GitHub profile).

```markdown
# Aulia Afriza

Frontend Engineer transitioning to AI Engineering. Currently shipping AI products in production.

## Featured

- **[Personal Chat AI](https://github.com/aulia/personal-chat-ai)** — Full-stack RAG chat app with 24 tools, memory, observability. Live: [personalchat.aulia.dev](https://personalchat.aulia.dev)

## Currently working on

- Agent framework mini-project with LangGraph (research assistant with human-in-the-loop)
- Fine-tuning Llama 3.2 3B for Indonesian task with LoRA
- Interview prep for AI Engineer roles

## Stack I use most

`Next.js` `TypeScript` `Go` `Postgres + pgvector` `Voyage AI` `Groq` `LangGraph`

## Writing

- [How I built a personal AI assistant in 12 weeks](https://dev.to/aulia/...)
- [Hybrid search in Postgres: vector + BM25 + rerank](https://dev.to/aulia/...)
- [5 lessons from shipping LLM app to production](https://dev.to/aulia/...)

## Reach out

- [LinkedIn](https://linkedin.com/in/aulia-afriza)
- [Email](mailto:auliaafriza@gmail.com)
- DM open for AI Engineer role opportunities
```

## Cover letter template (customized per company)

**Subject:** AI Engineer application — Aulia Afriza (portfolio: personalchat.aulia.dev)

Hi <HIRING_MANAGER_NAME>,

Aku apply untuk role <ROLE_NAME> di <COMPANY> yang aku lihat di <SOURCE>. Setelah baca job description dan produk <COMPANY>, aku confident stack aku match apa yang tim kalian cari.

Background singkat: 3-tahun Frontend Engineer, barusan transisi ke AI Engineer via 12-week self-directed roadmap. Portfolio project: PersonalChatAI — chat app production dengan hybrid RAG, 24 tools, dan observability built-in. Stack: Next.js + Go + Neon Postgres + Voyage + Groq. Live di personalchat.aulia.dev, case study di README.

Yang menarik aku dari <COMPANY>: <1-2_KALIMAT_SPESIFIK — misalnya tentang produk mereka, blog post yang kamu baca, mission yang resonate>.

Kontribusi awal yang bisa aku offer:
- <SKILL_1 sesuai job description>: <bukti dari portfolio, misalnya "designed streaming protocol antara Next.js dan Go backend, debugging Vercel AI SDK compat">
- <SKILL_2>: <bukti dari portfolio>

Attached CV + link portfolio + blog case study. Fleksibel untuk 30-menit call minggu depan, timezone WIB atau UTC overlap.

Terima kasih atas pertimbangannya.

Aulia Afriza
- LinkedIn: linkedin.com/in/aulia-afriza
- Portfolio: personalchat.aulia.dev
- Blog: dev.to/aulia
- Email: auliaafriza@gmail.com

---

**Notes:**
- **Customize** untuk tiap company. Recruiter detect template dalam 5 detik.
- **Sertakan LINK** portfolio + case study. Barrier to entry harus rendah.
- **Bullet point kontribusi** — jangan generalize "aku bisa react + go", spesifik ke apa yang mereka butuhin.
- **Fleksibel timezone** — signal kamu bisa remote-friendly.

## Application answers — common questions

**"Tell us about a challenging problem you solved."**

> Di portfolio project, aku implement multi-turn tool calling dengan Vercel AI SDK di frontend + Go backend custom. Streaming protocol AI SDK expect frame format spesifik (`f:` step start, `9:` tool call, `a:` tool result, `e:` step finish per iteration). Setelah 2 hari debugging, aku find root cause: aku emit step finish frame di akhir semua iteration, bukan per-iteration, dan finish reason string harus "tool-calls" (dash) bukan "tool_calls" (underscore). Setelah fix, multi-turn tool loop jalan smooth. Learning: kalau implement custom protocol, cari reference implementation dulu — saved days of trial and error.

**"How do you approach eval and testing for LLM apps?"**

> Aku bangun 2 layer eval untuk RAG pipeline. Layer 1: retrieval eval dengan 50 manual query-answer pairs, ukur recall@5 dan MRR — objektif, deterministic, run tiap kali ubah retrieval logic. Layer 2: LLM-as-judge dengan Llama 3.1 8B untuk quality scoring — subjective tapi captures answer quality yang retrieval metric miss. Concrete impact: eval framework aku bikin bisa detect regression sub-minute vs manual QA yang butuh 30 menit. Numbers: hybrid search + rerank improve recall@5 dari 0.62 ke 0.90 (measurable, defendable).

**"What's your approach to production observability for LLM apps?"**

> Aku implement in-DB tracing: setiap request generate trace ID, semua major operation jadi span (LLM call, tool call, retrieval, DB query). Span metadata JSONB include input token count, output token count, model, latency, tool args hash, tool result size. Dashboard `/observability` show recent trace + P50/P95 latency by operation type. Untuk portfolio scale ini cukup dan lebih murah dari OpenTelemetry + Datadog. Kalau scale up, interface tracer bisa di-swap ke OTel exporters tanpa ubah call site.

## Distribution checklist

- [ ] Update LinkedIn Experience section (paste template above)
- [ ] Update LinkedIn Headline (pilih option A/B/C)
- [ ] Update LinkedIn About section
- [ ] Update GitHub profile README
- [ ] Update CV: pdf + docx version dengan Recent Projects paragraf
- [ ] Update portfolio site (kalau ada) — landing page + case study page
- [ ] Prepare 3-5 STAR stories untuk interview: technical challenge + collaboration + growth mindset + failure + strategic thinking
- [ ] Voice memo practice: elevator pitch 60 detik, deep tech story 3 menit, question kamu untuk interviewer
