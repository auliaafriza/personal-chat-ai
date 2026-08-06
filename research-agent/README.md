# Research Agent

LangGraph-based research assistant. Decompose complex query jadi sub-questions, parallel search, dan synthesize jawaban dengan critic feedback loop.

Bagian dari Roadmap AI Engineer v2 Minggu 4-5 (mini-project A: agent framework).

## Status

- **Part 1 (Minggu 4):** ✅ Planner + Searcher + full graph wiring + CLI + tests
- **Part 2 (Minggu 5):** ✅ Reader + Writer + Critic full impl + human-in-the-loop + checkpointer + cost tracking + blog post + demo script

Companion docs:
- **[COMPARISON.md](./COMPARISON.md)** — LangGraph vs tool loop pattern decision framework
- **[DEMO_SCRIPT.md](./DEMO_SCRIPT.md)** — 2-menit Loom video script + LinkedIn post template

## Architecture

```
                              ┌─────────────┐
                              │    START    │
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │   Planner   │  Decompose query → sub-questions
                              │   (Groq)    │  Structured JSON output
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │   Searcher  │  Parallel Tavily search
                              │   (Tavily)  │  Top-3 per sub-question
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │    Reader   │  Extract relevant excerpts
                              │    (Groq)   │  (Part 2 — LLM relevance filter)
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                     ┌───────►│    Writer   │  Synthesize draft with citations
                     │        │    (Groq)   │  Increment iteration counter
                     │        └──────┬──────┘
                     │               ▼
                     │        ┌─────────────┐
                     │        │    Critic   │  Evaluate coverage + accuracy
                     │        │    (Groq)   │  Approved? / Suggest fixes
                     │        └──────┬──────┘
                     │               │
              reject │               │ approved OR
              (loop) │               │ iteration >= max_iter
                     │               ▼
                     │        ┌─────────────┐
                     └────────┤    END      │
                              └─────────────┘
```

**State schema** (`src/state.py`) — TypedDict yang di-pass antara nodes. Setiap node return partial update, LangGraph auto-merge.

**Conditional edge** di Critic (`src/graph.py`) — kalau reject dan iteration < max, loop balik ke Writer dengan suggestions. Kalau approved atau max reached, END.

## Kenapa LangGraph vs simple tool loop?

Aku juga bangun tool-loop assistant di [PersonalChatAI](https://github.com/aulia/personal-chat-ai) — pattern beda, konteks beda. Kapan pakai apa?

| Pattern | Cocok untuk | Trade-off |
|---|---|---|
| **Tool loop** (single-LLM + tools) | Chat multi-turn dengan tool calling ad-hoc. User drive konteks. | Cepat setup, susah orchestrate multi-step workflow yang predictable |
| **LangGraph** (state machine + nodes) | Workflow explicit dengan langkah defined + conditional branch (research, ETL, review loop) | Boilerplate lebih banyak, tapi explicit + testable + resumable via checkpointer |

Research pipeline (decompose → search → read → write → critique) fit LangGraph karena:
1. Langkah predictable (5 nodes)
2. Perlu conditional loop (writer↔critic)
3. Perlu parallel fanout (searcher searches semua sub-question simultan)
4. Perlu human-in-the-loop hook (Minggu 5)

Tool loop cocok kalau user boleh drive urutan. State machine cocok kalau urutan fixed.

## Setup

Requires Python 3.11+.

```bash
# Clone atau cd ke folder ini
cd research-agent

# Install deps (pakai uv atau pip)
uv sync
# atau
pip install -e ".[dev]"

# Setup API keys
cp .env.example .env
# Edit .env dan isi GROQ_API_KEY + TAVILY_API_KEY
```

## Usage

```bash
# Basic
python cli.py "Bandingkan fine-tuning vs RAG buat domain-specific chatbot"

# Verbose logging (debug)
python cli.py "What are best practices for RAG evaluation?" --verbose

# Limit iteration writer↔critic
python cli.py "Explain LoRA" --max-iterations 2

# Interactive mode — user approve sub-questions + draft
python cli.py "Query" --interactive

# Resume interrupted run (checkpoint auto-persist ke ./checkpoints.sqlite)
python cli.py "Query" --thread-id abc123
```

Output includes:
- Sub-questions dari planner (dengan reasoning)
- Search hits dari Tavily (title + URL per sub-question)
- Reader notes (LLM-extracted relevant excerpts)
- Draft (writer output)
- Critic verdict + suggestions (kalau reject)
- Final answer synthesized dengan `[N]` citations
- Cost summary table (per-node breakdown)

## Testing

```bash
pytest -v
```

Tests use mocked LLM + Tavily calls — no API keys needed for CI.

Coverage:
- `test_planner.py` — happy path, LLM error fallback, sub-question truncation, empty response
- `test_graph.py` — full graph end-to-end dengan mocked external calls

## Project structure

```
research-agent/
├── pyproject.toml       # deps + config
├── .env.example         # API keys template
├── cli.py               # entry point
├── src/
│   ├── state.py         # TypedDict state schema
│   ├── graph.py         # StateGraph wiring + conditional edges + interrupt support
│   ├── llm.py           # Groq async wrapper (with usage tracking)
│   ├── checkpoint.py    # SQLite checkpointer for resume/interrupt
│   ├── usage.py         # Token usage + cost tracker
│   ├── nodes/
│   │   ├── planner.py   # ✅ Query decomposition (structured JSON)
│   │   ├── searcher.py  # ✅ Parallel Tavily search
│   │   ├── reader.py    # ✅ Parallel LLM excerpt extraction per source
│   │   ├── writer.py    # ✅ Synthesis with citations + retry-aware prompt
│   │   └── critic.py    # ✅ 4-dimension evaluation + budget cap
│   └── tools/
│       └── tavily.py    # Tavily API client
└── tests/
    ├── test_planner.py
    ├── test_reader.py
    ├── test_writer.py
    ├── test_critic.py
    ├── test_usage.py
    └── test_graph.py
```

## Design decisions

**Kenapa Python instead of TypeScript?**
LangGraph Python jauh lebih matang (docs, examples, community). TypeScript version masih catch-up. Untuk portfolio demonstrate LangGraph specifically, Python better.

**Kenapa Groq bukan OpenAI?**
Cost + latency + free tier. Aku pakai Groq intensif di PersonalChatAI, consistency stack membantu.

**Kenapa Tavily bukan SerpAPI atau Brave?**
Tavily API optimized for LLM agents — return content excerpt langsung, bukan cuma URL. Free tier 1000/bulan cukup untuk portfolio. SerpAPI lebih mahal, Brave less mature untuk agent use case.

**Kenapa TypedDict state instead of Pydantic model?**
LangGraph docs recommend TypedDict untuk state (lighter, no validation overhead di setiap node). Pydantic bagus untuk external API boundary (llm.py, tavily.py), TypedDict bagus untuk internal state passing.

**Kenapa parallel search di Searcher, not sequential?**
5 sub-questions × ~2s per Tavily call = 10s sequential, ~2s parallel. Latency win obvious. Rate limit Tavily free tier (100/menit) way above 5 concurrent, aman.

**Graceful degradation di setiap node:**
- Planner fail → treat query utama as single sub-question
- Tavily fail per query → empty result untuk sub-question itu, continue
- Reader fail → skip note, continue
- Writer fail → return draft placeholder
- Critic fail → auto-approve (better UX than infinite loop)

## References

- LangGraph docs: https://langchain-ai.github.io/langgraph/
- ReAct paper: https://arxiv.org/abs/2210.03629
- Reflexion paper: https://arxiv.org/abs/2303.11366 (untuk Part 2 critic pattern)
- Tavily API: https://docs.tavily.com/
- Groq API: https://console.groq.com/docs

## License

MIT
