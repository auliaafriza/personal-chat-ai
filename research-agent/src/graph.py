"""LangGraph StateGraph wiring untuk research agent.

Graph structure:
    START → Planner → Searcher → Reader → Writer ⇄ Critic → END

Conditional edge di Critic:
- kalau approved OR iteration >= max_iterations → END
- kalau reject → loop back ke Writer (dengan suggestions dari Critic)

Interactive mode (--interactive di CLI):
- `interrupt_before=["searcher"]` — user approve/edit sub_questions sebelum search
- `interrupt_before=["critic"]` — user preview draft sebelum critic evaluate/finalize

Interrupt butuh checkpointer supaya state persistent antara pause + resume.
"""

from typing import Any
from langgraph.graph import StateGraph, START, END

from .state import ResearchState
from .nodes import planner_node, searcher_node, reader_node, writer_node, critic_node


def should_continue_writing(state: ResearchState) -> str:
    """Conditional edge dari Critic:
    - approved atau max iteration reached → END
    - kalau tidak → loop balik ke Writer
    """
    critique = state.get("critique")
    iteration = state.get("iteration", 0)
    max_iter = state.get("max_iterations", 3)

    if critique is None:
        return END

    if critique.get("approved") or iteration >= max_iter:
        return END
    return "writer"


def build_graph(
    *,
    checkpointer: Any = None,
    interactive: bool = False,
):
    """Build + compile the LangGraph StateGraph.

    Args:
        checkpointer: SqliteSaver atau MemorySaver instance. Required kalau interactive=True.
        interactive: kalau True, interrupt sebelum searcher + critic untuk human review.

    Return compiled graph siap di-invoke.
    """
    graph = StateGraph(ResearchState)

    graph.add_node("planner", planner_node)
    graph.add_node("searcher", searcher_node)
    graph.add_node("reader", reader_node)
    graph.add_node("writer", writer_node)
    graph.add_node("critic", critic_node)

    graph.add_edge(START, "planner")
    graph.add_edge("planner", "searcher")
    graph.add_edge("searcher", "reader")
    graph.add_edge("reader", "writer")
    graph.add_edge("writer", "critic")

    graph.add_conditional_edges(
        "critic",
        should_continue_writing,
        {"writer": "writer", END: END},
    )

    compile_kwargs: dict[str, Any] = {}
    if checkpointer is not None:
        compile_kwargs["checkpointer"] = checkpointer
    if interactive:
        # Interrupt sebelum searcher (setelah planner) + sebelum critic (setelah writer)
        compile_kwargs["interrupt_before"] = ["searcher", "critic"]

    return graph.compile(**compile_kwargs)
