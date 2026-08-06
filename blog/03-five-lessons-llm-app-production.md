---
title: "5 Lessons Bikin LLM App di Production (Yang Nggak Ditulis di Tutorial)"
published: false
description: "Honest retrospective 12 minggu bikin RAG chat app dari nol ke production. Bug yang aku salah design, decision yang aku regret, dan yang aku bakal lakuin beda kalau restart."
tags: [ai, rag, engineering, career]
---

Bulan lalu aku ship portfolio AI project 12-week pertama aku — Personal Chat AI, chat app dengan RAG + tools + memory + observability. Live di production, running cost ~$3-5/bulan.

Kelihatan berhasil, tapi kalau aku restart dari nol, ada 5 keputusan yang bakal aku lakuin beda. Tulisan ini bukan brag list, ini honest retrospective — mistake, painful debug session, dan trade-off yang aku pilih salah. Semoga membantu kalau kamu lagi build LLM app sekarang.

## Lesson 1: Observability day-one, bukan minggu ke-11

Aku bikin tracer buat request pipeline di minggu ke-11 dari 12. Sepuluh minggu pertama, kalau ada bug — "kenapa chat lambat?", "kenapa tool fail?", "kenapa citation nggak muncul?" — aku debug pakai `fmt.Println` scattered di codebase, lalu grep di Railway logs.

Ini stupid. Setelah tracer masuk, semua request generate trace dengan span breakdown: retrieval (150ms), llm_stream iter 0 (800ms), tool.write_file (30ms), llm_stream iter 1 (600ms), total 1.58s. Bug jadi obvious dalam detik, bukan jam.

**Yang aku pelajari:** observability bukan optimization, itu **debugging tool**. Bikin instrumentation dari commit pertama. Untuk LLM app minimum ada 3 span kategori: LLM call (input token, output token, model, latency, finish reason), tool call (name, args hash, result size, latency), retrieval (candidates count per stage, final top-K, latency). JSONB di Postgres cukup untuk portfolio scale, upgrade ke OpenTelemetry kalau serious.

**Kalau restart:** hari pertama setup Tracer interface, wire ke semua handler, bikin `/observability` view sederhana. Sisanya build di atas foundation itu.

## Lesson 2: Streaming protocol tuning butuh reference implementation

Aku pakai Vercel AI SDK v4 di frontend (`useChat` hook), backend Go custom. Protocol data stream Vercel AI SDK tricky sekali — format frame spesifik: `f:` untuk step start, `0:` untuk text delta, `9:` untuk tool call, `a:` untuk tool result, `e:` untuk step finish, `d:` untuk stream done.

Bug yang bikin aku frustasi 2 hari: multi-turn tool loop. Setelah tool call, LLM lanjut generate response. Tapi frontend nggak update — tool card muncul, tapi assistant text hilang. Aku curiga backend bug, JSON parse bug, race condition. Bukan.

Root cause: aku emit `e:` step finish sekali di akhir semua iteration, bukan per-iteration. Vercel AI SDK expect `e:` per step transition, dengan finish reason mapping "tool_calls" → `tool-calls` (dash, bukan underscore).

**Yang aku pelajari:** kalau kamu implement protocol custom, cari **reference implementation open-source** dulu (Vercel AI SDK repo punya example server di Python + JS). Reverse engineer dari console log frontend lebih lama dari baca kode reference 15 menit.

**Kalau restart:** buka reference implementation dulu, port ke bahasa aku baru mulai. Atau — realistically — pertimbangin pakai SDK official kalau ada (misal Vercel AI SDK ada JS backend, kalau aku pakai Node.js sudah plug-and-play).

## Lesson 3: Graceful degradation > feature completeness

External API akan fail. Voyage kadang timeout. Groq kadang rate-limited (free tier). Tavily kadang return empty. Google API kadang refresh token expired.

Awal-awal aku wrap semua external call dengan `return nil, fmt.Errorf(...)` — kalau satu fail, request crash dan user dapat 500 error.

Setelah refactor: setiap external call di-guard, kalau fail, pipeline continue tanpa feature itu:

- Voyage rerank fail → fall back ke top-K by RRF score. User dapat quality lebih rendah, tapi jawaban tetap.
- BM25 search fail → pakai vector only.
- Web search fail → LLM continue tanpa tool result, ngasih answer dari training.
- Google token expired → tool return "please re-authenticate", chat continue normal.

User rarely notice degraded quality kalau alternative reasonable ada. User definitely notice "500 internal server error" page.

**Yang aku pelajari:** every external dependency = potential failure mode. Design pipeline supaya bisa lose salah satu tanpa collapse. Test dengan **chaos engineering ringan**: kill Voyage API key sengaja, verify chat masih jalan.

**Kalau restart:** dari awal wrap external call dengan `try-except-return-fallback` pattern, plus catat di trace `service.voyage.rerank: fallback_used=true`.

## Lesson 4: Rerank stage worth setiap millisecond

Aku sempet pertimbangin skip rerank stage karena tambahan latency 300ms per request. Tapi setelah aku eval:

| Config | Recall@5 | MRR |
|---|---|---|
| Pure vector | 0.62 | 0.41 |
| Vector + BM25 fusion | 0.78 | 0.55 |
| Vector + BM25 + rerank | **0.90** | **0.71** |

Improvement 45% recall dari baseline vector-only ke full pipeline. 300ms latency vs response quality trade-off jelas menang di quality.

Yang bikin aku hampir skip: rerank feel expensive (API call terpisah, ada latency budget). Tapi kenyataannya, users tolerate 300ms extra latency **jauh lebih baik** dari answer yang miss context relevan.

**Yang aku pelajari:** **eval before optimize**. Bikin eval set 30-50 query manual di awal (nggak perlu banyak). Ukur baseline. Ukur setiap perubahan retrieval logic. Regret decision tanpa eval jauh lebih mahal dari waktu bikin eval set.

**Kalau restart:** minggu ke-3 aku bakal spend 1 hari full bikin eval infrastructure (retrieval eval + LLM judge) sebelum lanjut fitur baru. Semua retrieval decision setelahnya data-driven.

## Lesson 5: Sandboxed workspace tool paling worth build

24 tools aku build, yang paling delightful buat user (aku sendiri, dogfooding tiap hari) adalah file operations: read_file, write_file, list_directory, search_code, run_shell.

Kenapa? Karena chat jadi personal dev environment. "Bikin file utils/math.go isinya function Add(a, b int) int" — 3 detik file ready. "List file di workspace ku" — instant overview. "Search string 'TODO' di semua file .go" — instant grep result. "Run go test ./..." — LLM execute + parse hasil.

Tapi ini butuh **security sandbox serius**. User workspace confined ke `/data/workspaces/{user_id}/`. Path validation 3-layer:

1. Reject absolute path (`/etc/passwd`).
2. Reject `..` component.
3. Verify via `filepath.Rel(base, resolved)` result starts dengan `..` → escape attempt.

Shell command punya allowlist (`go`, `ls`, `cat`, `grep`, etc), execution timeout 30 detik, output truncate ke 10KB. Nggak boleh network access (kalau kamu run di container, network isolated).

**Yang aku pelajari:** feature paling powerful sering butuh security investment paling besar. Kalau nggak siap invest security, jangan build feature-nya. Alternative sederhana: batasin ke read-only file operations (read + list), no write, no shell.

**Kalau restart:** ini yang aku bakal build lebih awal (minggu 6 vs actual minggu 8), karena dogfooding sendiri jadi motivator terbesar buat improve produk.

## Bonus: Yang aku over-engineer

Untuk balance — 3 hal yang aku overthink dan buang waktu:

**Rate limiter token bucket in-memory.** Aku implement per-user token bucket dari nol karena pengen "learn". Untuk portfolio single-user, kalau boleh jujur, hardcoded `time.Sleep(100ms)` between request cukup. Kalau scale, pakai Redis-based (misalnya `redis-rate`) yang battle-tested.

**Custom JWT verification instead of Auth.js middleware.** Aku bikin shared HS256 JWT antara FE Auth.js dan BE Go — kompleksitas medium (share secret, verify signature, parse claims). Alternative simpler: BE cukup verify session via Auth.js callback endpoint. Trade-off latency, tapi lebih standard.

**LLM-as-judge eval script.** Bagus untuk portfolio "aku bisa design evaluation framework", tapi untuk actual quality monitoring aku jarang jalanin. Metric yang aku track sehari-hari: recall@5 di retrieval eval + P95 latency di trace. Cukup.

## Kalau kamu build LLM app sekarang

Kalau aku restart hari ini:

1. **Hari 1:** setup observability infrastructure (tracer + `/observability` view).
2. **Hari 2:** setup eval infrastructure (30-query test set + retrieval eval + baseline metrics).
3. **Hari 3-14:** build minimum viable RAG (vector + BM25 + fusion, skip rerank sampai eval bilang butuh).
4. **Hari 15-30:** add tool calling framework, mulai dari 3 tools (calculator, web search, file read). Instrumentation setiap tool call di trace.
5. **Hari 31-45:** add rerank kalau eval menunjukan improvement, add memory kalau UX butuh cross-session context.
6. **Hari 46+:** polish, deploy, distribute (blog + demo video + LinkedIn).

Bottom line: **fondasi (observability + eval) sebelum feature**. Fondasi bikin semua feature setelahnya cepat di-iterate dengan confidence.

---

Kalau ada lesson kamu sendiri yang aku miss atau kamu disagree soal trade-off aku, reply di comment. Aku curious dengar experience developer lain — terutama yang scale beyond portfolio ke real user traffic.

Repo full-nya di [github.com/aulia/personal-chat-ai](https://github.com/aulia/personal-chat-ai) kalau kamu mau lihat implementation detail.
