"""Searcher node — parallel Tavily search untuk semua sub-questions.

Design decision: parallel via asyncio.gather. Tavily punya rate limit (free tier ~100 req/menit),
5 sub-questions concurrent well within limit. Untuk scale, tambah semaphore.

Graceful degradation: kalau salah satu search gagal (rate limit, network), sub-question itu
dapat empty result. Reader node handle "kalau results kosong, skip aja sub-question ini".
"""

import os
import logging
import asyncio

from ..state import ResearchState, SearchResult, SubQuestionSearches
from ..tools.tavily import search_tavily


log = logging.getLogger(__name__)


async def _search_one(sub_question: str, max_results: int) -> SubQuestionSearches:
    """Search 1 sub-question, wrap result dengan struktur SubQuestionSearches."""
    tavily_results = await search_tavily(sub_question, max_results=max_results)

    # Tag setiap result dengan sub_question source untuk traceability
    search_results: list[SearchResult] = [
        SearchResult(
            title=r["title"],
            url=r["url"],
            content=r["content"],
            score=r["score"],
            sub_question=sub_question,
        )
        for r in tavily_results
    ]

    return SubQuestionSearches(
        sub_question=sub_question,
        results=search_results,
    )


async def searcher_node(state: ResearchState) -> dict:
    """Searcher node handler.

    Ambil sub_questions dari state, fan-out ke parallel Tavily search,
    aggregate ke SearchResults list.
    """
    sub_questions = state.get("sub_questions", [])
    max_results = int(os.environ.get("SEARCH_RESULTS_PER_QUESTION", "3"))

    if not sub_questions:
        log.warning("[searcher] no sub_questions in state, skipping")
        return {"search_results": []}

    log.info(f"[searcher] parallel search untuk {len(sub_questions)} sub-questions")

    # Parallel search
    tasks = [_search_one(q, max_results) for q in sub_questions]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    # Filter out exceptions (log tapi jangan crash)
    valid_results: list[SubQuestionSearches] = []
    for i, r in enumerate(results):
        if isinstance(r, Exception):
            log.error(f"[searcher] sub-question {i} failed: {r}")
            # Tambah empty result supaya downstream tau ada gap
            valid_results.append(SubQuestionSearches(
                sub_question=sub_questions[i],
                results=[],
            ))
        else:
            valid_results.append(r)

    total_hits = sum(len(r["results"]) for r in valid_results)
    log.info(f"[searcher] total {total_hits} search hits across {len(sub_questions)} questions")

    return {"search_results": valid_results}
