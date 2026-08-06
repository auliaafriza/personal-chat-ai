"""State schema untuk research agent graph.

TypedDict-based state supaya LangGraph bisa validate + auto-merge di setiap node.
Field bertambah seiring workflow: Planner nambah `sub_questions`, Searcher nambah `search_results`, dll.
"""

from typing import Annotated, TypedDict
from operator import add


class SearchResult(TypedDict):
    """Satu hasil search dari Tavily (atau provider lain)."""
    title: str
    url: str
    content: str
    score: float
    sub_question: str  # Sub-question yang trigger search ini — buat trace source


class SubQuestionSearches(TypedDict):
    """Hasil search untuk 1 sub-question."""
    sub_question: str
    results: list[SearchResult]


class Critique(TypedDict):
    """Output dari critic node."""
    approved: bool
    reasoning: str
    suggestions: list[str]  # Kalau not approved, apa yang harus difix


class ResearchState(TypedDict, total=False):
    """State yang di-pass antara nodes.

    total=False artinya field bisa optional — awalnya cuma `query`, node lain nambah field
    seiring execution.
    """
    # Input awal
    query: str

    # Planner output
    sub_questions: list[str]
    plan_reasoning: str  # Kenapa planner decompose gini

    # Searcher output
    search_results: list[SubQuestionSearches]

    # Reader output (Minggu 5) — extracted relevant excerpts + summaries
    reader_notes: Annotated[list[str], add]  # Accumulate across iterations

    # Writer output (Minggu 5)
    draft: str

    # Critic output + loop control (Minggu 5)
    critique: Critique
    iteration: int  # Counter untuk cegah infinite writer↔critic loop
    max_iterations: int  # Default 3

    # Final output
    final_answer: str


def initial_state(query: str, max_iterations: int = 3) -> ResearchState:
    """Bikin state awal — cuma butuh query + config."""
    return ResearchState(
        query=query,
        sub_questions=[],
        search_results=[],
        reader_notes=[],
        draft="",
        iteration=0,
        max_iterations=max_iterations,
        final_answer="",
    )
