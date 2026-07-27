# LinkedIn Cross-Post Templates

Ready-to-paste posts untuk share 2 blog posts + demo video. Update placeholder di `<>` sebelum publish.

## Post 1 — Blog "12 weeks journey" + demo video (post pembuka)

**Best day/time to post:** Tuesday–Thursday, 8-10am WIB (jam Indonesia scroll LinkedIn saat mulai kerja).

---

12 minggu lalu aku Frontend Engineer. Hari ini aku baru ship portfolio AI Engineer pertama aku.

Personal Chat AI — self-hosted chat app yang aku bangun dari nol:

→ RAG dengan hybrid search (vector + BM25 + rerank) pakai Postgres pgvector
→ 24 tools: file operations, Google Calendar, Gmail, web search, calculator
→ Long-term memory yang auto-inject preferensi ke tiap session
→ Observability built-in — every request punya trace yang bisa aku debug
→ Live production di Vercel + Railway, cost ~$3-5/bulan

Stack: Next.js 15 + TypeScript + Go + Neon Postgres + Voyage AI + Groq.

Yang paling penting bukan feature list-nya — tapi keputusan trade-off di setiap layer, dan gimana aku belajar debug production LLM issues. Aku tulis full breakdown di case study + video demo 4 menit.

📝 Blog: <LINK_BLOG_1>
🎥 Demo: <LINK_LOOM>
🔗 Live: personalchat.aulia.dev
💻 Repo: github.com/aulia/personal-chat-ai

Kalau kamu FE engineer yang mikir transisi ke AI, atau kamu hiring AI Engineer, aku happy chat. DM terbuka.

/cc <TAG_AI_ENGINEER_1> <TAG_AI_ENGINEER_2> <TAG_AI_ENGINEER_3> — kalau ada feedback atau approach lebih baik yang harus aku coba, curious dengar.

#AIEngineering #RAG #NextJS #Golang #CareerTransition

---

**Notes on execution:**
- Ganti `<TAG_AI_ENGINEER_*>` dengan 3 akun aktif di AI engineering Indonesia atau global. Candidates:
  - Ruben Hassid (@rubenhssd) — AI content lead
  - Alex Fazio (@alexfazio) — AI engineering builder
  - Simon Willison (@simonw) — LLM engineering luminary
  - Local: Ruangguru AI team, Kata.ai engineers, Traveloka data science
- Reply cepat kalau ada engagement dalam 1-2 jam pertama (LinkedIn algorithm reward early engagement).
- Kalau <20 likes dalam 24 jam, boost dengan reply di comment sendiri linking ke blog kedua.

## Post 2 — Blog technical "Hybrid Search" (post follow-up, 2-3 hari kemudian)

**Best day/time:** Wednesday atau Thursday, 8-10am WIB. Jarak 2-3 hari dari post 1 supaya nggak overlap feed.

---

Punya opinion agak controversial: "cukup pakai vector search" untuk RAG itu bad default advice.

Bulan lalu pas ngebangun chat app pribadi, aku eval 3 config retrieval di 50 query test set:

• Pure vector search: recall@5 = 0.62
• Vector + BM25 (RRF fusion): recall@5 = 0.78 (+26%)
• Vector + BM25 + Voyage rerank: recall@5 = 0.90 (+45%)

Untuk query yang exact-match sensitive — API key, product code, error message — vector search bener-bener miss. BM25 handle ini. Rerank stage tambahan bikin ordering final jauh lebih relevant.

Yang menarik: semua ini bisa dilakukan di Postgres saja. Nggak butuh Pinecone atau vector DB dedicated. pgvector + tsvector + trigger auto-update = complete stack.

Aku tulis full breakdown dengan:
→ SQL schema + indexes (HNSW + GIN)
→ Go orchestrator code — parallel search, RRF fusion, rerank pipeline
→ Eval methodology + numbers
→ Trade-off tiap decision (kenapa Postgres, kenapa Voyage, kenapa RRF k=60)

📝 Full post: <LINK_BLOG_2>

Kalau kamu bangun RAG production, silakan copy pattern-nya. Curious dengar approach lain — kamu pakai reranker apa? BM25 alternative?

#RAG #Postgres #pgvector #AIEngineering #InformationRetrieval

---

**Notes on execution:**
- Ini technical post — targeting engineer audience. Tag lebih spesifik ke technical folks (bukan influencer).
- Post di WhatsApp group AI Indonesia (Kata.ai community, Ruangguru Tech, Bandung AI) untuk warm distribution.
- Cross-post di dev.to + Medium seminggu setelah LinkedIn (SEO purposes, LinkedIn algorithm dislike duplicate content posted same day).

## Post 3 — Demo video only (post follow-up, weekend 1 minggu setelah post 1)

**Best day/time:** Saturday atau Sunday morning, 9-11am WIB. Weekend engagement rate lebih tinggi untuk video content.

---

Bikin video demo 4 menit dari portfolio AI Engineer aku baru shipped.

Tour singkat:
0:20 — Sign in flow (Google OAuth + shared JWT antara Next.js dan Go BE)
0:45 — Upload PDF + tanya isinya, dengan citation link balik ke source
1:30 — Trigger tool `write_file` — LLM bikin file di sandbox per-user
2:15 — Long-term memory yang inget preferensi cross-session
2:45 — Buka /observability, tunjukin trace per request dengan span breakdown

Semua fitur bisa di-explore langsung di production: personalchat.aulia.dev

🎥 Video: <LINK_LOOM>

Kalau ada segment yang kamu mau aku dive lebih deep (technical breakdown, decision rationale, code walkthrough), reply di comment. Aku bakal bikin follow-up post.

#AIEngineering #Portfolio #ProductDemo

---

## Cold DM Template — untuk applications & networking

Untuk DM AI engineers di target companies (Minggu 2 job hunt). Simpen di note untuk personalized send.

**Subject / opener:**
Hi <NAMA>, aku baru ship portfolio AI app 12-week (RAG + tools + observability) — full-stack Next.js + Go, live di production. Aku lihat kamu <SPECIFIC_THING_ABOUT_THEM — role, blog post, project>. Curious kalau kamu ada 15 menit untuk quick feedback tentang portfolio-ku, atau share how you got into <THEIR_ROLE>. Timeline flexible, aku traktir kopi virtual. Portfolio: <LINK>

**Do:**
- Personalize opener minimum 1 sentence tentang mereka (blog post yang kamu baca, project yang kamu suka, dll).
- Kirim link portfolio + demo video, jangan cuma repo — barrier to entry harus low.
- Batasin ask ke 15 menit + specific topic. Nggak minta job.

**Don't:**
- Send ke 10 orang copy-paste sama. Recruiter dan senior engineer bisa detect langsung.
- Send tanpa follow up. Kalau nggak reply dalam 5 hari, kirim polite bump satu kali.
- Ngajak "coffee chat" tanpa specific topic. Waktu mereka mahal, kasih reason spesifik.

## Application email template — untuk formal apply

**Subject:** AI Engineer application — <NAMA>, dengan portfolio production stack

Halo tim <COMPANY>,

Aku Aulia, background Frontend Engineer 3-tahun yang barusan transisi ke AI Engineer via 12-week self-directed roadmap.

Portfolio-ku: Personal Chat AI — chat app production dengan RAG hybrid search (vector + BM25 + rerank), 24 tools termasuk sandboxed code execution, long-term memory, dan observability built-in. Stack: Next.js 15 + Go + Neon Postgres + Voyage + Groq. Live di personalchat.aulia.dev, code di github.com/aulia/personal-chat-ai.

Aku attach CV. Blog case study dengan tech decisions + trade-offs ada di <LINK_BLOG_1>. Video demo 4-menit di <LINK_LOOM>.

Kalau ada 30-menit call minggu depan buat discuss role dan cara aku bisa contribute, aku fleksibel timezone WIB atau UTC.

Terima kasih,
Aulia
LinkedIn: linkedin.com/in/aulia-afriza
Email: auliaafriza@gmail.com

---

## Distribution checklist

Setelah 3 blog + demo video ready:

- [ ] Post 1 di LinkedIn (Tuesday morning)
- [ ] Cross-post Post 1 di dev.to (same day, mark as canonical to LinkedIn)
- [ ] Post 2 di LinkedIn (Thursday/Friday)
- [ ] Cross-post Post 2 di dev.to
- [ ] Post 3 di LinkedIn (Sat/Sun following week)
- [ ] Update Twitter/X bio with pinned tweet linking demo
- [ ] Send cold DM ke 3 target AI engineers via LinkedIn (personalized)
- [ ] Apply ke 2-3 exciting roles dengan cover letter template
- [ ] Update CV bagian "Recent Projects" dengan 1 paragraf tentang PersonalChatAI
- [ ] Log semua applications di spreadsheet target companies

**Success metrics minggu ini:**
- 3 blog posts published (2 major + reserved slot untuk Blog 3 minggu depan)
- 100+ engagement gabungan (likes + comments + shares) di LinkedIn
- 5+ meaningful conversation via DM / comment
- 2-3 applications submitted
