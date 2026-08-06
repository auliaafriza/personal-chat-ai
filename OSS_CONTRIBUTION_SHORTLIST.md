# OSS Contribution Shortlist — Minggu 3 Roadmap v2

Target: **1 PR merged** dalam 5-7 hari. Fokus ke lib yang kamu udah pakai sehari-hari — sudah tau context, tinggal contribute back.

## Prinsip pilih PR

**Do:**
- Cari `good first issue` atau `help wanted` label — maintainer sudah agree issue itu valid + boleh outside contribution
- Kontribusi ke lib yang kamu pakai di PersonalChatAI (Vercel AI SDK, pgvector, dll) — kamu punya real production use case sebagai motivator
- Scope kecil: docs improvement, test coverage, bug fix di function isolated — 100-300 LOC max, merged dalam 1 minggu
- Kontribusi yang bisa disebutin di CV/interview: "aku fix hydration bug di Vercel AI SDK" > "aku fix typo di README"

**Don't:**
- Ambil feature request besar tanpa diskusi dulu — bisa reject setelah 2 minggu kerja
- Kontribusi ke lib kamu belum pernah pakai — extra ramp-up waste time
- PR docs typo saja — merge cepat tapi zero signal buat recruiter

## Top 3 candidates (ranked by fit)

### #1 — Vercel AI SDK
**Repo:** https://github.com/vercel/ai
**Kenapa fit:** Kamu pakai `useChat` di FE, sudah familiar dengan stream protocol (2 hari debug — ini asset). Maintainer aktif merge PR mingguan.

**Cara cari issue yang cocok:**
1. https://github.com/vercel/ai/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
2. https://github.com/vercel/ai/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22
3. Filter kategori: `provider` (misalnya bug di Groq provider integration), `docs` (missing example untuk multi-turn tool loop), `useChat` (client hook improvement).

**Kandidat PR yang high-impact untuk kamu:**
- **Docs contribution:** Cara integrate custom backend (non-Vercel AI SDK Node.js). Kamu udah bikin Go backend dari nol — pengalaman ini valuable buat comm dengan orang lain yang bikin backend custom. PR: tambah "Adapting to non-Node backends" section di docs, dengan example minimal.
- **Provider bug:** Cek issue di `packages/groq`. Kalau ada bug di tool call parsing (kamu pengalaman debug ini), fix + tambah test case.
- **Type improvement:** Kalau ada issue soal TypeScript type inference di `useChat` (misalnya `messages` type nggak infer dari `experimental_output`), submit fix + regression test.

**Effort:** 4-8 jam untuk docs contribution, 8-16 jam untuk code fix. Merged timeline: 1-2 minggu (maintainer responsive).

### #2 — pgvector-go
**Repo:** https://github.com/pgvector/pgvector-go
**Kenapa fit:** Kamu pakai pgvector di Go backend, kenal driver + query patterns. Repo lebih kecil (maintainer 1-2 orang) tapi PR simpler.

**Cara cari issue:**
1. https://github.com/pgvector/pgvector-go/issues
2. Cek pinned issues + `enhancement` label.

**Kandidat PR:**
- **Example addition:** Tambah example untuk `HNSW index creation` di `examples/` dir. Repo ada example untuk basic vector insert + query, tapi belum ada full example dengan HNSW tuning params (`m`, `ef_construction`). Kamu punya production experience — bisa contribute example yang real-world tested.
- **Test coverage:** Cek coverage report, kalau ada function tanpa test (misalnya edge case NULL vector handling), tambah test case.
- **Docs improvement:** README kadang miss info tentang `pgx` v5 compatibility. Kalau ada issue soal ini, kontribusi fix + example.

**Effort:** 4-6 jam. Merged timeline: 1-3 minggu (maintainer volume lower).

### #3 — Groq SDK (unofficial atau official)
**Repo:** https://github.com/groq/groq-typescript atau https://github.com/groq/groq-python
**Kenapa fit:** Kamu pakai Groq intensively, familiar dengan API quirks (tool_calls parsing edge case, streaming format).

**Cara cari issue:**
- Cek `good first issue` di kedua repo.
- Cek closed issues yang belum di-fix di docs.

**Kandidat PR:**
- **Docs:** Tambah cookbook example untuk "multi-turn tool calling with Llama 3.3" di official docs repo. Kamu udah implement ini di production, code ready-to-adapt.
- **Bug fix:** Kalau ada issue soal tool_calls response format inconsistency (kamu pengalaman debug ini), submit fix di client SDK.

**Effort:** 4-8 jam. Merged timeline: 1-2 minggu.

## Backup candidates (kalau top 3 nggak nemu issue cocok)

- **react-syntax-highlighter** (https://github.com/react-syntax-highlighter/react-syntax-highlighter) — kamu pakai di chat code block rendering. Aktif tapi maintainer slower.
- **Auth.js / next-auth** (https://github.com/nextauthjs/next-auth) — kamu pakai v5 dengan Google OAuth extended scopes. Ada banyak `good first issue` untuk provider fix.
- **golang-migrate/migrate** (https://github.com/golang-migrate/migrate) — kamu pakai untuk DB migration. Repo mature, PR queue panjang tapi merge steady.

## Execution timeline (5-7 hari)

**Hari 1 (2-3 jam):**
- Browse issue tracker top 3 repo, shortlist 3-5 issue yang match
- Comment di issue: "Interested to work on this. Any pointers on where to start?" — signal maintainer + confirm scope

**Hari 2 (waiting):**
- Wait maintainer response (biasanya <24 jam untuk aktif repo)
- Kalau dapat green light, mulai implement

**Hari 3-4 (4-6 jam):**
- Fork repo, clone, setup dev environment
- Implement fix + tambah test case (semua PR harus include test kalau modify code)
- Run local test suite, pastikan pass
- Follow contribution guide (CODE_OF_CONDUCT, CONTRIBUTING.md, commit message format)

**Hari 5 (1 jam):**
- Draft PR description dengan clear "what", "why", "how tested"
- Screenshot atau code example kalau relevant
- Submit PR, link ke issue

**Hari 6-7 (waiting + iterate):**
- Respond maintainer feedback dalam 24 jam
- Iterate sampai merged

## Kalau PR di-reject / stuck

Nggak apa-apa, tetap ada value:

- PR public visible di GitHub profile — bukti kamu try contribute
- Learnings dari code review = hal yang bisa disebutin di interview
- Rebuild ke different repo (backup candidates)

## Distribution setelah merged

- Tweet: "Just got my first PR merged to @vercel AI SDK — added <THING>. Loved contributing back to lib I use daily."
- LinkedIn post: rangkum experience contribute OSS untuk pertama kali (kalau iya) — 200 kata max.
- Update CV: bagian "Open Source" — line "Contributor to vercel/ai (link ke PR)"

## Interview talking points setelah merged

Prepare 60-second story:

> "Aku pakai Vercel AI SDK di portfolio project, ketemu bug/gap di X. Investigate root cause di source code, drafted PR fixing Y dengan test case Z. Maintainer accept setelah 2 iterasi review. Belajar ada trade-off soal ABC yang aku nggak notice sebelum contribute. Sekarang production project aku pakai canary version berisi fix aku sendiri."

Cerita ini demonstrate:
- Familiar dengan lib production
- Bisa navigate codebase asing
- Bisa handle code review feedback
- Ownership mindset
