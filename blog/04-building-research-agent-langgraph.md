---
title: "Building a Research Agent with LangGraph: State Machine vs Tool Loop"
published: false
description: "How I built a 5-node research agent with LangGraph — planner, searcher, reader, writer, critic. Full walkthrough dengan code, human-in-the-loop, checkpointing, cost tracking, plus honest comparison ke tool-loop pattern."
tags: [langgraph, agent, ai, python]
---

Setelah ship RAG chat app dengan tool-loop pattern di [PersonalChatAI](https://personalchat.aulia.dev), aku penasaran: kapan pattern state-machine (LangGraph) actually better dari simple tool loop? Jawaban terbaik: bikin sendiri, jangan baca 20 blog post.

Weekend lalu aku bangun research assistant pakai LangGraph — decompose query, parallel search, synthesize dengan critic feedback loop. Full source di [github.com/aulia/research-agent](https://github.com/aulia/research-agent).

Tulisan ini walkthrough architecture, key implementation decisions, plus **honest opinion soal kapan LangGraph worth boilerplate-nya vs tool loop yang lebih simple**.

## Yang di-build: research assistant 5-node

Input: research query kompleks. Output: markdown answer dengan citation dari sources.

Contoh:
```
$ python cli.py "Bandingkan performa fine-tuning vs RAG buat domain-specific chatbot"
```

Agent akan:
1. **Planner** — decompose query jadi 4 sub-questions specific
2. **Searcher** — parallel Tavily search per sub-question (top-3 hasil each)
3. **Reader** — LLM extract 1-3 kalimat relevan dari setiap source
4. **Writer** — synthesize markdown answer dengan `[N]` citations
5. **Critic** — evaluate coverage/citation/faithfulness → approve atau reject
6. Kalau reject → loop back ke Writer dengan suggestions (max 3 iterations)

Total runtime: ~15-25 detik. Total cost: ~$0.005 per run (13 LLM call rata-rata, mostly Groq Llama 3.3 70B).

## Graph structure

```
                              ┌─────────────┐
                              │    START    │
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │   Planner   │
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │   Searcher  │
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                              │    Reader   │
                              └──────┬──────┘
                                     ▼
                              ┌─────────────┐
                     ┌───────►│    Writer   │
                     │        └──────┬──────┘
                     │               ▼
                     │        ┌─────────────┐
                     │        │    Critic   │
                     │        └──────┬──────┘
                     │               │
              reject │               │ approved OR max_iter
                     │               ▼
                     │        ┌─────────────┐
                     └────────┤    END      │
                              └─────────────┘
```

## State schema — TypedDict, bukan Pydantic

LangGraph state di-pass antara nodes, setiap node return partial update yang auto-merge. TypedDict cocok karena lightweight (no validation overhead per node call), plus type checker tetap happy.

```python
from typing import TypedDict

class ResearchState(TypedDict, total=False):
    query: str
    sub_questions: list[str]
    plan_reasoning: str
    search_results: list[SubQuestionSearches]
    reader_notes: list[str]
    draft: str
    critique: Critique
    iteration: int
    max_iterations: int
    final_answer: str
```

`total=False` artinya field optional — start dengan cuma `query`, node lain nambah field seiring workflow. LangGraph merge partial state pakai dict update semantics by default (kalau kamu perlu accumulate list across iterations, wrap dengan `Annotated[list, add]`).

## Planner — structured JSON output

Groq support `response_format={"type": "json_object"}` — force valid JSON. Model tetap harus di-prompt untuk output JSON, tapi format guarantee membantu.

```python
async def planner_node(state: ResearchState) -> dict:
    query = state["query"]

    messages = [
        {"role": "system", "content": PLANNER_SYSTEM_PROMPT},
        {"role": "user", "content": f"Query: {query}\nDecompose ke max 5 sub-questions."},
    ]

    try:
        result = await chat_json(messages, node="planner", temperature=0.3)
    except LLMError:
        # Graceful degradation: treat query utama as single sub-question
        return {"sub_questions": [query], "plan_reasoning": "Fallback"}

    sub_qs = [str(q).strip() for q in result.get("sub_questions", []) if str(q).strip()][:5]
    return {"sub_questions": sub_qs, "plan_reasoning": result.get("reasoning", "")}
```

Prompt penting-nya:
- Enforce independence antar sub-questions (bisa di-search terpisah)
- Prefer 3-4 untuk query simple, 5 untuk kompleks
- Factual/objective, bukan opinion

## Searcher — parallel via asyncio.gather

Fan-out ke Tavily untuk semua sub-questions simultan. 5 sub-questions sequential = ~10 detik, parallel = ~2 detik.

```python
async def searcher_node(state: ResearchState) -> dict:
    sub_qs = state.get("sub_questions", [])
    tasks = [_search_one(q, max_results=3) for q in sub_qs]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    valid = []
    for i, r in enumerate(results):
        if isinstance(r, Exception):
            # Empty result untuk sub_q ini, continue overall
            valid.append({"sub_question": sub_qs[i], "results": []})
        else:
            valid.append(r)
    return {"search_results": valid}
```

Tavily API optimized for LLM agents — return content excerpt (bukan cuma URL). Free tier 1000 searches/bulan cukup untuk portfolio.

## Reader — parallel excerpt extraction

Per `(sub_question, source)` pair, LLM extract 1-3 kalimat relevant. Kalau irrelevant, output literal string `NOT_RELEVANT` → reader skip.

```python
async def _read_one(idx, sub_question, title, url, content) -> str | None:
    messages = [
        {"role": "system", "content": READER_SYSTEM_PROMPT},
        {"role": "user", "content": f"Sub-question: {sub_question}\n\nSource:\n{content[:3000]}"},
    ]
    excerpt = await chat(messages, node="reader", temperature=0.2)
    if "NOT_RELEVANT" in excerpt.upper():
        return None
    return f"[{idx}] {title} ({url}): {excerpt}"
```

Kenapa per-pair bukan batch semua sources sekali? Batch prompt akan gede + LLM harder to focus per-source. Per-pair prompt kecil + parallel = latency similar, quality lebih tinggi. Trade-off: LLM call count naik (5 sub_q × 3 source = 15 call), tapi cost tetap murah (~$0.003 di 8B model).

Concurrency capped via semaphore biar tidak trip Groq rate limit (free tier 30 req/min).

## Writer — synthesis dengan retry-aware prompt

Writer prompt include:
- Original query + sub-questions
- All reader notes (with `[N]` labels)
- **Kalau iteration > 0**, prepend critic suggestions dari attempt sebelumnya

```python
async def writer_node(state: ResearchState) -> dict:
    iteration = state.get("iteration", 0)
    critique = state.get("critique")

    user_parts = [f"Research query: {query}", "Notes:", *notes]

    if iteration > 0 and critique and critique.get("suggestions"):
        user_parts.append("\n**Draft sebelumnya di-reject. Address suggestions:**")
        for s in critique["suggestions"]:
            user_parts.append(f"- {s}")

    draft = await chat(messages, node="writer", temperature=0.5, max_tokens=3000)
    return {"draft": draft, "iteration": iteration + 1}
```

Critical detail: `iteration += 1` di return supaya conditional edge di Critic bisa cap loop.

## Critic — structured evaluation dengan budget cap

Critic evaluate 4 dimensi (coverage, citation integrity, faithfulness, clarity), return structured JSON `{approved, reasoning, suggestions[]}`. Kritis: **budget cap** — kalau iteration >= max_iterations, force approve tanpa LLM call untuk cegah infinite loop.

```python
async def critic_node(state: ResearchState) -> dict:
    iteration = state.get("iteration", 0)
    max_iter = state.get("max_iterations", 3)

    # Budget cap
    if iteration >= max_iter:
        return {
            "critique": {"approved": True, "reasoning": "Max iter reached", "suggestions": []},
            "final_answer": state["draft"],
        }

    # ... normal evaluation via LLM
```

Bonus: sebelum call LLM, extract `[N]` citations dari draft, cross-check dengan available notes range, feed ke prompt sebagai "citation stats" — bikin critic evaluation lebih grounded.

## Conditional edge

Di `graph.py`:

```python
def should_continue_writing(state: ResearchState) -> str:
    critique = state.get("critique")
    iteration = state.get("iteration", 0)
    max_iter = state.get("max_iterations", 3)

    if critique.get("approved") or iteration >= max_iter:
        return END
    return "writer"

graph.add_conditional_edges("critic", should_continue_writing, {"writer": "writer", END: END})
```

## Human-in-the-loop — interrupt_before

LangGraph support pause execution di step tertentu, resume setelah user input. Butuh checkpointer (SQLite) supaya state persist.

```python
graph = graph_builder.compile(
    checkpointer=AsyncSqliteSaver.from_conn_string("checkpoints.sqlite"),
    interrupt_before=["searcher", "critic"],
)
```

Di CLI:
1. Invoke graph → auto-stop sebelum searcher, return state dengan sub_questions
2. Print sub_questions, tanya user "approve?"
3. Kalau approve → resume dengan `ainvoke(None, config)`
4. Kalau reject → `aupdate_state(config, {"sub_questions": edited_list})` → resume
5. Ulangi untuk critic step

Full CLI ada di [cli.py](https://github.com/aulia/research-agent/blob/main/cli.py).

Feels natural once you get it — but interrupt+resume pattern definitely more boilerplate than plain function calls.

## Cost tracking — instrument LLM calls

Setiap LLM call di-record ke module-level `UsageTracker`:

```python
async def chat(messages, *, node="unknown", model=None, ...):
    resp = await client.post(GROQ_URL, ...)
    data = resp.json()
    await get_tracker().record(node, model, data.get("usage"))  # <-- instrument here
    return data["choices"][0]["message"]["content"]
```

Setelah run, CLI print table:

```
        Cost Summary
┏━━━━━━━━━━┳━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
┃ Node     ┃ Calls ┃ Input tok ┃ Output tok ┃ Cost (USD) ┃
┡━━━━━━━━━━╇━━━━━━━╇━━━━━━━━━━━╇━━━━━━━━━━━━╇━━━━━━━━━━━━┩
│ planner  │     1 │       350 │        180 │    $0.0004 │
│ reader   │    15 │     4,500 │        900 │    $0.0034 │
│ writer   │     1 │     2,800 │      1,200 │    $0.0026 │
│ critic   │     1 │     1,500 │        250 │    $0.0011 │
├──────────┼───────┼───────────┼────────────┼────────────┤
│ TOTAL    │    18 │     9,150 │      2,530 │    $0.0075 │
└──────────┴───────┴───────────┴────────────┴────────────┘
```

Data-driven cost optimization jadi trivial.

## LangGraph vs tool loop — kapan pakai apa?

Dua-duanya valid pattern. Pilihan tergantung structure task:

| Aspect | Tool loop | LangGraph |
|---|---|---|
| **Workflow structure** | User drive, ad-hoc | Fixed pipeline, predefined |
| **Parallelism** | Sequential (satu tool per turn) | Explicit parallel (multiple nodes) |
| **State transitions** | Implicit di system prompt | Explicit di graph edges |
| **Human-in-the-loop** | Awkward — harus special turn | Built-in via interrupt_before |
| **Resumability** | Sesuai kualitas kamu track state | Built-in via checkpointer |
| **Setup boilerplate** | Minimal — 1 loop + tool schemas | Higher — state, graph, nodes, conditional edges |
| **Best for** | Chatbot dengan tools, user-driven | Research pipeline, ETL, agentic workflow |

**Tool loop cocok kalau:**
- User drive konteks (chatbot, coding assistant)
- Tool usage ad-hoc, sequence tidak predictable
- Simple workflow, no complex conditional branching
- Contoh: PersonalChatAI, ChatGPT plugins

**LangGraph cocok kalau:**
- Workflow ada structure predefined (pipeline, DAG)
- Butuh conditional loop atau branching
- Perlu parallel fanout eksplisit
- Perlu human-in-the-loop hooks
- Perlu resumable execution
- Contoh: research agent, document processing pipeline, review workflow

Rule of thumb: **kalau kamu bisa gambar flowchart untuk workflow tanpa loop di logic-nya, tool loop cukup**. Kalau flowchart ada branching + explicit orchestration, LangGraph value-add.

## Lessons learned dari build ini

**1. Parallelism di orchestration = big win.** Sequential Searcher ~10s, parallel ~2s. Sequential Reader ~30s (15 source × 2s), parallel ~4s. LangGraph nggak automatic parallel — kamu tetap perlu `asyncio.gather` di dalam node.

**2. Interrupt_before dan checkpointer must-have together.** interrupt tanpa checkpointer bakal lose state. checkpointer tanpa interrupt overhead tanpa benefit.

**3. TypedDict > Pydantic untuk graph state.** Docs LangGraph explicit recommend TypedDict — Pydantic overhead di setiap merge. Pydantic fine untuk external boundary (API request/response validation).

**4. Structured JSON output game-changer.** `response_format={"type": "json_object"}` di Groq + prompt yang enforce schema = 99% reliable structured output. Manual `json.loads` fallback tetap perlu untuk safety.

**5. Budget cap penting untuk agent yang self-loop.** Writer↔critic pattern powerful, tapi tanpa iteration limit, LLM bisa nyangkut loop 10+ round + burn budget. Hard cap force-approve di iteration max.

**6. Cost tracking dari day 1.** Aku hampir skip ini sampai lihat 1 test run kena $0.02 karena reader over-eager call LLM per source. Instrument dari awal = detect regression instantly.

## Yang aku nggak bangun (scope creep)

Sengaja skip biar scope ke portfolio project pertama:

- **Multi-agent orchestration** — kayak CrewAI, AutoGen. Research agent ini basically single-agent-multi-node. Multi-agent = beda agent independent, terinter-koordinasi. Overkill untuk research task.
- **Reflection loop di reader** — LLM re-read sendiri, self-critique per excerpt. Redundant sama critic node.
- **Long-term memory across queries** — agent stateless per query. Cukup untuk research use case.
- **Streaming response** — writer emit token stream. Nice UX tapi break structured graph output pattern.

Kalau scale ke production, semuanya worth investigasi. Untuk portfolio + interview prep, current scope hits sweet spot.

## Cara coba

```bash
git clone https://github.com/aulia/research-agent
cd research-agent
pip install -e .
cp .env.example .env
# Isi GROQ_API_KEY + TAVILY_API_KEY

# Basic
python cli.py "Compare RAG vs fine-tuning trade-offs"

# Interactive human-in-the-loop
python cli.py "Query" --interactive

# Resume interrupted run
python cli.py "Query" --thread-id abc123
```

Feedback + PR welcome. Kalau kamu build sesuatu serupa dengan pattern beda, curious dengar approach kamu.

---

*Companion project: [PersonalChatAI](https://github.com/aulia/personal-chat-ai) — tool-loop pattern di production. Bandingin implementation-nya.*
