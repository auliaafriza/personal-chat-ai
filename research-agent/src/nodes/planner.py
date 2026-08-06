"""Planner node — decompose research query jadi 3-5 sub-questions.

Contoh:
    Input: "Bandingkan performa fine-tuning vs RAG buat domain-specific chatbot"
    Output:
      - "Apa itu fine-tuning LLM dan kapan cocok?"
      - "Apa itu RAG dan bagaimana cara kerjanya?"
      - "Trade-off cost antara fine-tuning vs RAG?"
      - "Latency dan quality benchmark tipikal keduanya?"
      - "Kapan harus kombinasiin keduanya (RAG-tuning)?"

Sub-questions ini nanti di-fanout ke Searcher untuk parallel search.
"""

import os
import logging
from ..state import ResearchState
from ..llm import chat_json, LLMError

log = logging.getLogger(__name__)


PLANNER_SYSTEM_PROMPT = """Kamu adalah research planner. Task kamu: decompose user query jadi 3-5 sub-questions yang independen dan searchable.

Aturan:
1. Setiap sub-question harus specific, self-contained (bisa di-search tanpa konteks tambahan)
2. Sub-questions harus cover different angles dari query utama (bukan rephrase yang sama)
3. Sub-questions harus factual/objective — bukan opinion atau prediksi
4. Prefer 3-4 sub-questions untuk query simple, 5 untuk query kompleks

Output JSON dengan struktur:
{
  "sub_questions": ["question 1", "question 2", ...],
  "reasoning": "1-2 kalimat kenapa decompose gini"
}

Response harus valid JSON, no additional text."""


async def planner_node(state: ResearchState) -> dict:
    """Planner node handler.

    Ambil `query` dari state, panggil LLM untuk decompose, return partial state update.
    LangGraph auto-merge partial state ke full state.
    """
    query = state["query"]
    max_sub = int(os.environ.get("MAX_SUB_QUESTIONS", "5"))
    temperature = float(os.environ.get("PLANNER_TEMPERATURE", "0.3"))

    log.info(f"[planner] decomposing query: {query[:80]}...")

    messages = [
        {"role": "system", "content": PLANNER_SYSTEM_PROMPT},
        {
            "role": "user",
            "content": f"Query: {query}\n\nDecompose ke max {max_sub} sub-questions.",
        },
    ]

    try:
        result = await chat_json(messages, node="planner", temperature=temperature, max_tokens=1024)
    except LLMError as e:
        log.error(f"[planner] LLM error: {e}")
        # Graceful degradation: kalau planner gagal, treat query utama sebagai satu sub-question
        return {
            "sub_questions": [query],
            "plan_reasoning": f"Planner LLM failed ({e}), fallback ke query utama saja.",
        }

    sub_questions = result.get("sub_questions", [])
    reasoning = result.get("reasoning", "")

    # Validation & sanitization
    if not isinstance(sub_questions, list) or not sub_questions:
        log.warning(f"[planner] invalid sub_questions in response: {result}")
        return {
            "sub_questions": [query],
            "plan_reasoning": "Planner returned invalid format, fallback ke query utama.",
        }

    # Truncate kalau LLM ngeluarin terlalu banyak
    sub_questions = [str(q).strip() for q in sub_questions if str(q).strip()][:max_sub]

    log.info(f"[planner] generated {len(sub_questions)} sub-questions")
    for i, q in enumerate(sub_questions, 1):
        log.debug(f"[planner]   {i}. {q}")

    return {
        "sub_questions": sub_questions,
        "plan_reasoning": reasoning,
    }
