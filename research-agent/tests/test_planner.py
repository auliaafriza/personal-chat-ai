"""Smoke test untuk planner node — mock LLM biar nggak butuh API key di CI.

Run:
    pytest tests/test_planner.py -v
"""

import pytest
from unittest.mock import patch, AsyncMock

from src.state import initial_state
from src.nodes.planner import planner_node


@pytest.mark.asyncio
async def test_planner_decomposes_query():
    """Happy path: LLM return valid JSON, planner return sub_questions."""
    state = initial_state("What is RAG?")

    mock_response = {
        "sub_questions": ["What is retrieval augmented generation?", "How does RAG work?"],
        "reasoning": "Split ke definisi dan mekanisme.",
    }

    with patch("src.nodes.planner.chat_json", new=AsyncMock(return_value=mock_response)):
        result = await planner_node(state)

    assert result["sub_questions"] == mock_response["sub_questions"]
    assert result["plan_reasoning"] == mock_response["reasoning"]


@pytest.mark.asyncio
async def test_planner_fallback_on_llm_error():
    """Kalau LLM raise LLMError, planner harus fallback ke query utama sebagai single sub-question."""
    from src.llm import LLMError

    state = initial_state("Test query")

    with patch("src.nodes.planner.chat_json", new=AsyncMock(side_effect=LLMError("mock error"))):
        result = await planner_node(state)

    assert result["sub_questions"] == ["Test query"]
    assert "fallback" in result["plan_reasoning"].lower()


@pytest.mark.asyncio
async def test_planner_truncates_excessive_sub_questions(monkeypatch):
    """Kalau LLM return terlalu banyak sub-question, planner harus truncate ke MAX_SUB_QUESTIONS."""
    monkeypatch.setenv("MAX_SUB_QUESTIONS", "3")

    state = initial_state("Test")
    mock_response = {
        "sub_questions": ["q1", "q2", "q3", "q4", "q5"],
        "reasoning": "",
    }

    with patch("src.nodes.planner.chat_json", new=AsyncMock(return_value=mock_response)):
        result = await planner_node(state)

    assert len(result["sub_questions"]) == 3
    assert result["sub_questions"] == ["q1", "q2", "q3"]


@pytest.mark.asyncio
async def test_planner_handles_empty_response():
    """Kalau LLM return kosong sub_questions list, fallback ke query utama."""
    state = initial_state("Empty test")

    mock_response = {"sub_questions": [], "reasoning": ""}

    with patch("src.nodes.planner.chat_json", new=AsyncMock(return_value=mock_response)):
        result = await planner_node(state)

    assert result["sub_questions"] == ["Empty test"]
