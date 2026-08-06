"""Smoke test untuk full graph — mock LLM + Tavily."""

import pytest
from unittest.mock import patch, AsyncMock

from src.state import initial_state
from src.graph import build_graph


@pytest.mark.asyncio
async def test_graph_runs_end_to_end():
    """Full graph should complete tanpa error dengan mock external calls."""
    planner_mock = {
        "sub_questions": ["Q1", "Q2"],
        "reasoning": "test decomposition",
    }
    tavily_mock = [
        {"title": "Mock Result", "url": "https://example.com", "content": "Mock content", "score": 0.9},
    ]

    with patch("src.nodes.planner.chat_json", new=AsyncMock(return_value=planner_mock)), \
         patch("src.tools.tavily.search_tavily", new=AsyncMock(return_value=tavily_mock)):
        graph = build_graph()
        state = initial_state("Test query")
        final = await graph.ainvoke(state)

    # Assertions on final state
    assert final["query"] == "Test query"
    assert final["sub_questions"] == ["Q1", "Q2"]
    assert len(final["search_results"]) == 2
    assert final["critique"]["approved"] is True
    assert "final_answer" in final and final["final_answer"]
