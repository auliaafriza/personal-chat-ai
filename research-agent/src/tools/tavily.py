"""Tavily search API wrapper.

Tavily = search API yang optimized untuk LLM agent (return content excerpt langsung,
bukan cuma URL). Free tier 1000 searches/bulan.

Docs: https://docs.tavily.com/docs/rest-api/api-reference
"""

import os
import logging
from typing import TypedDict
import httpx


log = logging.getLogger(__name__)

TAVILY_URL = "https://api.tavily.com/search"


class TavilyResult(TypedDict):
    title: str
    url: str
    content: str  # Excerpt (bukan full page)
    score: float  # Relevance score dari Tavily 0-1


class TavilyError(Exception):
    """Raised kalau Tavily API return error."""


async def search_tavily(
    query: str,
    *,
    max_results: int = 3,
    search_depth: str = "basic",  # "basic" atau "advanced" (advanced = 2 credit)
    include_answer: bool = False,
    timeout: float = 30.0,
) -> list[TavilyResult]:
    """Kirim search query ke Tavily, return top-N result.

    Kalau error, return empty list (graceful degradation — searcher node handle).
    """
    api_key = os.environ.get("TAVILY_API_KEY")
    if not api_key:
        log.warning("[tavily] TAVILY_API_KEY not set, skipping search")
        return []

    payload = {
        "api_key": api_key,
        "query": query,
        "max_results": max_results,
        "search_depth": search_depth,
        "include_answer": include_answer,
    }

    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.post(TAVILY_URL, json=payload)
    except httpx.HTTPError as e:
        log.error(f"[tavily] HTTP error for query '{query[:50]}': {e}")
        return []

    if resp.status_code != 200:
        log.error(f"[tavily] non-200 for query '{query[:50]}': {resp.status_code} {resp.text[:200]}")
        return []

    data = resp.json()
    raw_results = data.get("results", [])

    results: list[TavilyResult] = []
    for r in raw_results[:max_results]:
        results.append(TavilyResult(
            title=str(r.get("title", "")),
            url=str(r.get("url", "")),
            content=str(r.get("content", "")),
            score=float(r.get("score", 0.0)),
        ))

    log.info(f"[tavily] '{query[:60]}' → {len(results)} results")
    return results
