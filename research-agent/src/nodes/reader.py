"""Reader node — LLM-based excerpt extraction per source.

Per (sub_question, search_result) pair: prompt LLM extract 1-3 kalimat paling relevan.
Filter out result yang irrelevant (return None). Parallel across pairs.

Output format tiap note: `[N] title (url): excerpt` — supaya Writer bisa cite `[N]` langsung.

Kenapa per-pair, bukan batch: kalau batch semua sources sekaligus, prompt gede + LLM harder to
focus. Per-pair prompt kecil + parallel = latency lebih baik + quality lebih tinggi.
"""

import os
import logging
import asyncio

from ..state import ResearchState
from ..llm import chat, LLMError


log = logging.getLogger(__name__)


READER_SYSTEM_PROMPT = """Kamu adalah research reader. Task: extract 1-3 kalimat paling relevan dari source untuk jawab sub-question.

Aturan:
1. HANYA extract dari source content, jangan tambah info dari knowledge kamu.
2. Kalau source nggak relevan sama sekali dengan sub-question, output: NOT_RELEVANT
3. Kalau relevan, output 1-3 kalimat verbatim atau close paraphrase, no additional commentary.
4. Prefer kalimat yang factual, dengan angka/nama/definition kalau ada.

Format output: cuma extracted excerpts, no header, no preamble."""


async def _read_one(
    idx: int,
    sub_question: str,
    title: str,
    url: str,
    content: str,
) -> str | None:
    """Extract excerpt dari 1 source. Return `[N] title (url): excerpt` atau None kalau irrelevant."""
    messages = [
        {"role": "system", "content": READER_SYSTEM_PROMPT},
        {
            "role": "user",
            "content": (
                f"Sub-question: {sub_question}\n\n"
                f"Source title: {title}\n"
                f"Source URL: {url}\n"
                f"Source content:\n{content[:3000]}"  # Cap content ke 3k char untuk safety
            ),
        },
    ]

    try:
        excerpt = await chat(messages, node="reader", temperature=0.2, max_tokens=400)
    except LLMError as e:
        log.warning(f"[reader] source {idx} LLM fail: {e}")
        return None

    excerpt = excerpt.strip()
    if not excerpt or "NOT_RELEVANT" in excerpt.upper():
        log.debug(f"[reader] source {idx} deemed irrelevant untuk '{sub_question[:50]}'")
        return None

    return f"[{idx}] {title} ({url}): {excerpt}"


async def reader_node(state: ResearchState) -> dict:
    """Reader node handler.

    Fan-out ke parallel LLM extraction untuk semua (sub_question, source) pair.
    Return notes list yang siap dipakai Writer.
    """
    search_results = state.get("search_results", [])
    if not search_results:
        log.warning("[reader] no search_results, skipping")
        return {"reader_notes": []}

    # Build flat list of (sub_question, source) pairs
    pairs = []
    for sub_result in search_results:
        sub_q = sub_result["sub_question"]
        for r in sub_result["results"]:
            pairs.append((sub_q, r))

    if not pairs:
        log.warning("[reader] search_results ada tapi kosong per sub_q")
        return {"reader_notes": []}

    log.info(f"[reader] extracting excerpts dari {len(pairs)} sources (parallel)")

    # Concurrency limit — Groq free tier 30 req/min, cap concurrent
    max_concurrent = int(os.environ.get("READER_CONCURRENCY", "5"))
    sem = asyncio.Semaphore(max_concurrent)

    async def _bounded(idx, sub_q, source):
        async with sem:
            return await _read_one(idx, sub_q, source["title"], source["url"], source["content"])

    tasks = [_bounded(i + 1, sub_q, source) for i, (sub_q, source) in enumerate(pairs)]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    notes: list[str] = []
    for r in results:
        if isinstance(r, Exception):
            log.error(f"[reader] task failed: {r}")
            continue
        if r is not None:
            notes.append(r)

    log.info(f"[reader] {len(notes)} relevant notes extracted (dari {len(pairs)} sources)")
    return {"reader_notes": notes}
