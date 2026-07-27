---
title: "How I Built a Personal AI Assistant in 12 Weeks (as a Frontend Engineer)"
published: false
description: "Journey dari FE engineer ke AI engineer: 12 minggu, 8000+ LOC, RAG + 24 tools + observability. Ini yang aku bangun, tech stack-nya, dan yang aku pelajari."
tags: [ai, rag, llm, career]
canonical_url:
cover_image:
---

Delapan bulan lalu aku masih 100% Frontend Engineer — React, Next.js, TypeScript, hidup tenang di client-side. Terus aku sadar satu hal: kalau ML/AI bakal jadi bagian besar dari software engineering ke depan, aku harus ikut belajar sekarang atau ketinggalan.

Jadi aku bikin roadmap 12 minggu buat transisi ke AI Engineer role. Bukan bootcamp. Bukan course. Satu portfolio project besar, dari empty repo ke production. Ini tulisan tentang apa yang aku bangun, kenapa aku pilih stack-nya, dan lima hal yang aku pelajari sepanjang jalan.

## Yang dibangun: Personal Chat AI

Chat app pribadi mirip ChatGPT, tapi bisa aku tune sendiri. Aku ingin sesuatu yang:

- Bisa upload dokumen dan tanya isinya (RAG).
- Punya tools nyata (kalkulator, web search, file read/write, Google Calendar).
- Punya memory jangka panjang (nggak forget preferensi tiap session baru).
- Punya observability built-in (jadi aku tau kenapa response lambat atau salah).

Setelah 12 minggu, aku punya semuanya, live di production, dan open-source di GitHub.

**Live demo:** https://personalchat.aulia.dev (masih Google OAuth restricted — DM aku kalau mau invite)
**Repo:** https://github.com/aulia/personal-chat-ai

## Arsitektur singkat

```
┌──────────────────┐        HTTPS + JWT         ┌──────────────────┐
│   Next.js 15     │  ─────────────────────►    │   Go backend     │
│   (Vercel)       │                            │   (Railway)      │
│                  │  ◄─────────────────────    │                  │
│  - useChat       │       SSE data stream      │  - chi router    │
│  - Auth.js v5    │                            │  - pgx / no ORM  │
└──────────────────┘                            └────────┬─────────┘
                                                         │
                     ┌───────────────────────────────────┼───────────┐
                     ▼                     ▼             ▼           ▼
              ┌──────────────┐    ┌──────────────┐  ┌────────┐  ┌────────┐
              │ Neon         │    │ Voyage AI    │  │ Groq   │  │ Tavily │
              │ Postgres     │    │ - embed 512d │  │ - Llama│  │ - web  │
              │ + pgvector   │    │ - rerank-2   │  │  3.3   │  │  search│
              │ + tsvector   │    │              │  │  70B   │  │        │
              └──────────────┘    └──────────────┘  └────────┘  └────────┘
```

Setiap chat request:
1. Frontend kirim user message + full history + JWT.
2. Backend verify JWT, fetch top-3 relevant memory dari embedding table.
3. Backend hybrid search dokumen: vector top-20 + BM25 top-20 → RRF → Voyage rerank top-5.
4. Backend send context ke Groq LLM (Llama 3.3 70B) sebagai system prompt.
5. LLM stream response. Kalau ada tool_call, backend execute tool, feed result balik, LLM lanjut.
6. Semua step di-trace ke Postgres JSONB — bisa dibuka di `/observability` untuk debug.

## Tech stack + kenapa

Setiap keputusan aku ada trade-off-nya. Ini short version — full breakdown di [README case study](https://github.com/aulia/personal-chat-ai#case-study).

| Layer | Pilihan | Kenapa |
|---|---|---|
| **Frontend** | Next.js 15 App Router + Vercel AI SDK | `useChat` handles streaming + tool cards out-of-the-box |
| **Backend** | Go + chi + pgx (no ORM) | Concurrency native, no runtime surprises, satu binary deploy |
| **Database** | Neon Postgres + pgvector | Serverless Postgres yang punya vector search — nggak perlu Pinecone terpisah |
| **Embeddings** | Voyage AI voyage-3-lite (512 dim) | Cheaper + faster dari OpenAI, quality comparable |
| **Reranker** | Voyage rerank-2 | Rerank top-40 ke top-5 boost recall signifikan (aku benchmark) |
| **LLM** | Groq Llama 3.3 70B | Latency <500ms untuk first token, generous free tier |
| **Auth** | Auth.js v5 Google OAuth + shared HS256 JWT | Single sign-on, FE dan BE tetap decoupled |
| **Hosting** | Vercel (FE) + Railway with Volume (BE) | Free tier cover semua kebutuhan portfolio |

Cost total production: **~$3-5/bulan**. Untuk single-user portfolio, ini masih dalam free tier semua service kecuali sedikit di Railway.

## Fitur yang di-ship

**RAG dengan hybrid search.** Upload PDF, DOCX, MD, TXT. Chunk 500 token dengan 50 overlap. Vector + BM25 + rerank. Chat response include citation `[1]`, `[2]` yang bisa di-hover buat lihat snippet + source file.

**24 tools.** Calculator, web search (Tavily), fetch URL, current time, file operations (read/write/list/search), shell command execution (sandboxed), task management (add/list/complete/delete/search), Google Calendar (list/create/update/delete events), Gmail (list/read messages), memory management (add/search/delete), translate ID↔EN.

**Long-term memory.** Separate embeddings table. Setiap chat, top-3 memory paling relevan auto-inject ke system prompt. Assistant inget preferensi ("aku suka response singkat", "aku engineer di Jakarta") tanpa harus re-tell.

**Per-user sandboxed workspace.** Coding tools (write_file dll) semua confined ke `/data/workspaces/{user_id}/`. Path validation 3-layer: reject absolute, reject `..`, verify via `filepath.Rel`. Nggak bisa escape ke root filesystem.

**Observability built-in.** Setiap request generate trace dengan spans (retrieval, llm_stream per iteration, tool.write_file, dll). Persisted ke Postgres JSONB. Ada dashboard `/observability` untuk view trace + P50/P95 latency. Nggak butuh Datadog atau external tracing service.

**Eval framework.** Retrieval eval (recall@k + MRR) untuk RAG quality tracking. LLM-as-judge dengan Groq Llama 3.1 8B untuk quality scoring. Bisa jalanin tiap kali ubah retrieval logic buat detect regression.

## Lima hal yang aku pelajari

**1. Vercel AI SDK data stream protocol tricky sekali.** Format frame-nya spesifik: `f:` untuk step start, `0:` untuk text delta, `9:` untuk tool call, `a:` untuk tool result, `e:` untuk step finish, `d:` untuk done. Multi-turn tool loop butuh emit `e:` per iteration dengan finish reason `tool-calls` (dash, bukan underscore) untuk step transition. Debug ini makan 2 hari.

**2. Hybrid search beat pure vector setiap kali.** Vector search bagus buat semantic ("cara login" match "authentication flow"), tapi BM25 unbeatable untuk exact terms ("Bearer YXXK7"). RRF fusion (k=60) dari kedua ranking simple tapi effective. Rerank stage tambahan boost recall lagi 15-20%.

**3. Import cycle Go bisa di-fix dengan local interface.** Package `tools` butuh call `service.Translator`, tapi `service` sudah import `tools` (untuk Schema type). Solusi: define `translatorInterface` di package `tools`, `main.go` inject `service.Translator` yang satisfy interface itu. Clean, no cycle.

**4. Graceful degradation lebih penting dari feature completeness.** Setiap external API bisa fail (Voyage down, Groq rate-limited, Tavily blocked). Aku wrap semua external call, kalau gagal continue tanpa feature itu (no rerank, no web search, no citation). User dapat degraded experience, bukan 500 error page.

**5. Observability dari day-one saves debugging weeks.** Aku bikin tracer minggu ke-11. Kalau lebih awal, bakal save banyak jam debugging "kenapa chat lama?" atau "tool gagal kenapa?". Setiap request sekarang punya trace ID, kalau ada bug user bisa share ID + aku bisa pinpoint exact span yang failure.

## Yang berikutnya

12 minggu ini bikin aku confident bilang: aku bisa bangun AI product production dari nol, punya opinion soal trade-off tech stack, dan tau cara debug LLM app kalau ada issue.

Sekarang aku extend portfolio: agent framework mini-project (LangGraph research assistant), fine-tuning demo (Llama 3.2 3B untuk task Indonesian), plus prep interview intensive untuk AI Engineer role.

Kalau kamu hiring AI Engineer atau AI-fluent full-stack — atau kamu FE engineer yang lagi mikir transisi serupa — DM aku di [LinkedIn](https://linkedin.com/in/aulia-afriza). Happy to chat.

---

*Full tech decisions + trade-offs breakdown ada di [repo README](https://github.com/aulia/personal-chat-ai#case-study). Live demo di [personalchat.aulia.dev](https://personalchat.aulia.dev). Video demo 4-menit di [Loom](https://loom.com/share/...).*
