# LangGraph vs Tool Loop — When to Use What

Companion doc buat research-agent (LangGraph) vs PersonalChatAI (tool loop). Kalau kamu decide pattern untuk agent project, doc ini opinion-focused decision framework.

## TL;DR

- **Tool loop:** user drive flow, agent respond dengan tool call ad-hoc. Cocok untuk chatbot dengan capability tools.
- **LangGraph (state machine):** developer define fixed workflow structure, agent execute in-order. Cocok untuk pipeline dengan predefined steps + conditional branching.

**Kalau kamu bisa gambar flowchart tanpa perlu loop di logic:** tool loop cukup.
**Kalau flowchart ada explicit branching, parallel fanout, atau loop:** LangGraph value-add.

## Side-by-side feature comparison

| Aspect | Tool Loop | LangGraph |
|---|---|---|
| **Setup boilerplate** | Minimal: 1 loop, N tool schemas | Higher: State schema, StateGraph, node functions, edges, conditional edges |
| **Workflow control** | Implicit di system prompt | Explicit di graph structure |
| **Predictability** | Low (LLM decide tool sequence) | High (developer define exact sequence) |
| **Parallelism** | Sequential (satu tool per turn) | Explicit parallel node OR asyncio.gather di node |
| **State management** | Kamu track manual di message history | Built-in via TypedDict state passing |
| **Conditional branching** | Awkward (harus prompt LLM decide) | Native via `add_conditional_edges` |
| **Loops (retry, refinement)** | Manual di code, sulit control | Native via edge that returns to previous node |
| **Human-in-the-loop** | Harus bikin special turn | Native via `interrupt_before` / `interrupt_after` |
| **Resumability** | Manual (persist message history + tool call state) | Native via checkpointer (SQLite, Postgres, Redis) |
| **Debugging** | Message history log | State snapshots per node + LangSmith tracing |
| **Observability** | Custom instrumentation | LangSmith integration, structured tracing |
| **Streaming** | Native (LLM stream response) | Node-level events, more complex |
| **Testing** | Mock tool responses | Mock nodes, test conditional edges |
| **Complexity ceiling** | ~5-10 tools before prompt bloat | 20+ nodes manageable |
| **Learning curve** | Low (LLM API + tool schema) | Medium (graph concepts + checkpointer) |

## Decision framework — 5 questions

Ask ini 5 pertanyaan sebelum decide. Kalau majority "Yes" di kolom "LangGraph", pakai LangGraph. Kalau majority "Yes" di kolom "Tool loop", pakai tool loop.

| # | Question | Tool loop kalau... | LangGraph kalau... |
|---|---|---|---|
| 1 | **Workflow structure predefined?** | User drive, no fixed steps | Fixed pipeline, N sequential steps |
| 2 | **Butuh parallel fanout?** | Nggak, sequential OK | Ya, kebutuhan explicit parallel |
| 3 | **Butuh loop (retry, refine)?** | Nggak, satu-shot response cukup | Ya, writer↔critic pattern atau refinement loop |
| 4 | **Butuh human-in-the-loop?** | Nggak | Ya, user approve/edit intermediate output |
| 5 | **Butuh resume after interruption?** | Nggak, stateless per query | Ya, long-running task yang bisa di-Ctrl+C |

## Use case examples

### Cocok untuk tool loop

| Use case | Kenapa |
|---|---|
| **Chatbot dengan tools** (ChatGPT plugins, PersonalChatAI) | User drive konteks, tool usage ad-hoc |
| **Coding assistant** (Cursor Chat, Continue.dev) | Interactive, no fixed workflow |
| **Personal assistant** (calendar + email + tasks) | User request one-off, respond dengan actions |
| **Q&A dengan retrieval** | Single-shot: retrieve → generate |
| **Simple RAG** | No branching, no loops |

### Cocok untuk LangGraph

| Use case | Kenapa |
|---|---|
| **Research agent** (this project) | Explicit steps: plan → search → read → write → critique |
| **Code review agent** | Steps: read PR → analyze diff → check style → generate comments → summarize |
| **Data pipeline dengan LLM** | ETL-style workflow, explicit stages |
| **Multi-doc summarization** | Fanout ke docs (parallel) → merge → refine |
| **Document QA dengan verification** | Retrieve → answer → verify claim → escalate kalau conflict |
| **Autonomous agent dengan planning** | Plan → execute steps → re-plan kalau fail |
| **Workflow dengan approval gate** | Auto steps + human approval di checkpoint |

### Cocok untuk keduanya (bisa dua-duanya bekerja)

| Use case | Tool loop win kalau... | LangGraph win kalau... |
|---|---|---|
| **RAG chatbot dengan tool call** | Simple RAG + occasional tool call | Complex reasoning multi-step |
| **Customer support agent** | Semi-simple, respond dari FAQ | Complex escalation flow dengan human handoff |
| **Sales research (find company info)** | Ad-hoc research | Batch: process 100 companies dengan same pipeline |

## Implementation cost comparison (hands-on)

Basis: research-agent (LangGraph, ~450 LOC Python) vs PersonalChatAI (tool loop, ~2000 LOC Go untuk chat handler saja).

**Tool loop (PersonalChatAI chat handler pattern):**
```
Time to first working prototype: ~2 hari
Time to production-ready: ~2 minggu
LOC untuk core loop: ~150
Test complexity: rendah (mock tool call responses)
Observability effort: high (harus bikin sendiri tracer)
```

**LangGraph (research-agent):**
```
Time to first working prototype: ~2 hari (butuh baca docs LangGraph 3-4 jam dulu)
Time to production-ready: ~1 minggu
LOC untuk graph + nodes: ~250
Test complexity: sedang (mock LLM per node + test edges)
Observability effort: rendah (LangSmith integration atau custom via node instrumentation)
```

LangGraph investment awal lebih tinggi (learn graph concepts), tapi long-run maintain lebih mudah untuk workflow kompleks.

## Anti-patterns to avoid

### Tool loop anti-patterns

1. **20+ tools:** Prompt bloat, LLM confused, tool call quality drop. Split ke multiple agents atau switch ke LangGraph.
2. **Complex state tracking di message history:** Kalau kamu inject `<state>...</state>` XML di message dengan manual parsing, kamu re-invent state machine — pakai LangGraph.
3. **Workflow yang butuh strict ordering:** Kalau tool A HARUS run sebelum tool B, tool loop tidak guarantee ini. Force ke sequential via explicit prompt = fragile.

### LangGraph anti-patterns

1. **Simple linear pipeline tanpa branching:** Overkill. Plain function chain cukup.
2. **Single-node graph:** Boilerplate tanpa benefit. Pakai plain async function.
3. **State schema yang di-mutate in-place:** LangGraph expect immutable-ish partial returns. Modify state directly = bug.
4. **No checkpointer + interrupt_before:** Interrupt tanpa checkpointer akan lose state. Wasted config.

## Migration path

Started with tool loop, hit complexity ceiling? Migration path:

1. **Identify explicit workflow structure** — gambar flowchart mental model kamu, cari nodes + edges
2. **Extract state** — kumpulkan variable yang di-track di message history ke TypedDict state
3. **Split tool loop jadi nodes** — setiap "phase" kerjaan jadi 1 node
4. **Add edges** — hard-code sequence + tambah conditional kalau ada branching
5. **Test** — parity check output antar 2 pattern, tambah test case untuk edges

Backward direction (LangGraph → tool loop) jarang perlu. Kalau merasa perlu, biasanya sign workflow over-engineered.

## Bottom line

Dua pattern ini bukan competitor, mereka **complementary**. Aku pakai keduanya di 2 project berbeda karena problem-nya beda struktur.

- **PersonalChatAI:** user drive, chat-like interaction dengan tools. Tool loop natural fit.
- **Research agent:** developer define pipeline dengan explicit steps + critic loop. LangGraph natural fit.

Rule sederhana: **pilih pattern yang match struktur task, bukan yang lagi hype**. Tool loop simple, LangGraph powerful — dua-duanya production-ready.

## References

- [LangGraph docs](https://langchain-ai.github.io/langgraph/) — official
- [PersonalChatAI](https://github.com/aulia/personal-chat-ai) — tool loop example
- [Research Agent](https://github.com/aulia/research-agent) — LangGraph example
- [Anthropic "Building Agents"](https://www.anthropic.com/research/building-effective-agents) — general framework thinking
