# Demo Video Script — Research Agent (LangGraph)

Target: **2 menit Loom video** untuk share ke LinkedIn + tambah portfolio. Fokus: architecture story + live CLI demo + cost transparency.

## Persiapan sebelum recording

**Setup terminal:**
- Terminal font size 16pt+ (readable di video 720p)
- Dark theme (kontras lebih baik)
- Clean terminal: `clear` sebelum start
- Cd ke `research-agent/` folder
- `.env` sudah isi API keys (test dulu sebelum record)

**Setup browser (untuk architecture diagram shot):**
- Open `README.md` di GitHub atau local preview
- Zoom ke architecture ASCII diagram (fits screen)

**Pre-run:**
- Test query sekali tanpa record supaya API keys work + timing feel
- Query candidate (pick 1 yang concise output):
  - "What are the trade-offs between RAG and fine-tuning for domain-specific chatbots?"
  - "Compare LangGraph vs LangChain agents"
  - "Best practices for evaluating LLM output quality"

**Recording tool:** Loom desktop app, 720p atau 1080p HD.

## Shot list & timing

Total ~2 menit. Timing tight — rehearse 1-2x.

### 0:00 – 0:15 · Intro + hook (15 detik)

**Shot:** Terminal + face cam pojok kanan

**Talking points:**
> "Hi, aku Aulia. Ini research agent yang aku bangun pakai LangGraph — decompose query kompleks, parallel search, synthesize answer dengan critic feedback loop. Companion project untuk PersonalChatAI aku, tapi pattern beda."

### 0:15 – 0:30 · Architecture shot (15 detik)

**Shot:** Switch ke README ASCII diagram di browser

**Talking points:**
> "5 nodes: planner decompose query, searcher parallel Tavily, reader extract excerpts pakai LLM, writer synthesize dengan citations, critic evaluate. Kalau critic reject, loop balik ke writer dengan suggestions. Max 3 iterations."

### 0:30 – 1:15 · Live CLI demo (45 detik)

**Shot:** Switch back ke terminal, run query

```bash
python cli.py "What are trade-offs between RAG and fine-tuning for domain-specific chatbots?"
```

**Talking points sambil output stream:**

While planner runs:
> "Watch — planner decompose ke 4 sub-questions. Groq structured JSON output pakai response_format json_object."

While searcher runs:
> "Now searcher — parallel Tavily search buat semua sub-questions simultan via asyncio.gather. Bukan sequential."

While reader runs:
> "Reader ekstrak 1-3 kalimat relevan per source. Parallel juga, capped 5 concurrent buat respect Groq rate limit."

While writer + critic run:
> "Writer synthesize dengan markdown + [N] citations. Critic evaluate coverage + citation integrity, approve atau kasih suggestions untuk retry."

### 1:15 – 1:35 · Cost transparency (20 detik)

**Shot:** Scroll to bottom terminal, tunjukin cost table

**Talking points:**
> "Cost tracking built-in. 1 run ini: 18 LLM call, ~9k input token, ~2.5k output. Total cost $0.0075 — di bawah 1 sen. Sustainable buat portfolio use."

**Do:** Point ke row breakdown per node (planner/reader/writer/critic).

### 1:35 – 1:55 · Interactive mode teaser (20 detik)

**Shot:** Run interactive mode

```bash
python cli.py "Same query" --interactive
```

**Talking points:**
> "Interactive mode — pakai LangGraph interrupt_before + checkpointer buat human-in-the-loop. Setelah planner, tanya user approve sub-questions. Kalau reject, edit + resume dari checkpoint. Sama juga sebelum critic finalize."

**Do:** Show planner output, hit Enter approve, cut before search selesai (waktu-nya tidak cukup untuk full run).

### 1:55 – 2:05 · Closing (10 detik)

**Shot:** Back to README di GitHub

**Talking points:**
> "Source lengkap di github, plus companion doc bandingin ke tool-loop pattern PersonalChatAI. Kalau kamu build agent similar, curious dengar approach kamu."

**Do:** Slow scroll ke README, point ke repo URL.

## Alternate shot: 90-detik version

Kalau mau versi lebih pendek buat Twitter/X reel:

- 0:00 – 0:10 · Intro + tagline: "Research agent pake LangGraph"
- 0:10 – 0:25 · Architecture diagram 15 detik
- 0:25 – 1:00 · Live CLI 35 detik (skip interactive mode)
- 1:00 – 1:20 · Cost table 20 detik
- 1:20 – 1:30 · Closing + link

## LinkedIn post companion

Post untuk share video:

---

Bikin research agent pakai LangGraph weekend lalu.

Input: research query kompleks.
Output: markdown answer dengan citations dari 15-30 sources.

Pipeline 5-node: planner → searcher → reader → writer → critic. Critic bisa loop balik ke writer kalau draft kurang. Max 3 iterations, force-approve di iteration cap supaya no infinite loop.

Yang menarik dari build ini:
→ Parallel via asyncio.gather di dalam node (LangGraph nggak automatic parallel)
→ Structured JSON output dari Groq bikin structured evaluation reliable
→ Human-in-the-loop via interrupt_before + SqliteSaver checkpointer
→ Cost tracking built-in — total ~$0.005 per run

Companion doc: compare pattern LangGraph vs tool loop yang aku pake di PersonalChatAI. TL;DR: dua-duanya production-ready, pilih yang match structure task.

🎥 Demo 2-menit: <LOOM_URL>
💻 Code: github.com/aulia/research-agent
📄 Comparison doc: github.com/aulia/research-agent/blob/main/COMPARISON.md

Interested dengar approach lain — kamu pakai framework apa buat agent workflow?

#LangGraph #AIEngineering #Python

---

## Recording tips

- **Rehearse 2x tanpa record.** Timing tight, familiarize dengan talking points
- **Slow down.** Recording bikin tempo terasa cepat — pelan aja
- **Salah ngomong OK.** Pause 2 detik, ulang kalimat, edit post-recording di Loom
- **Query pick pendek.** Kalau chosen query ternyata butuh 40 detik search, pick query lebih simple

## Post-processing

**Loom editing:**
- Trim awkward silence >2 detik
- Add chapter markers per section
- Cover thumbnail: screenshot cost table (visually striking)

**Loom settings:**
- Video quality: 720p atau 1080p HD
- Allow comments: ON
- Custom URL slug: `research-agent-langgraph-demo`

## Distribution checklist

- [ ] Loom URL di GitHub README (near top)
- [ ] LinkedIn post (weekday morning 8-10am WIB)
- [ ] Cross-post Twitter/X
- [ ] Add ke portfolio site (kalau ada)
- [ ] DM 2-3 AI engineer di target companies dengan companion doc + video
