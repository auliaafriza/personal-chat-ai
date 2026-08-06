"""Test untuk reader node — LLM-based excerpt extraction."""

import pytest
from unittest.mock import patch, AsyncMock

from src.state import initial_state
from src.nodes.reader import reader_node


@pytest.mark.asyncio
async def test_reader_extracts_relevant_excerpts():
    """Happy path: LLM return relevant excerpt, reader wrap dengan [N] format."""
    state = initial_state("Test query")
    state["search_results"] = [
        {
            "sub_question": "What is X?",
            "results": [
                {"title": "X Explained", "url": "https://ex.com/x", "content": "X is a thing.", "score": 0.9, "sub_question": "What is X?"},
            ],
        },
    ]

    with patch("src.nodes.reader.chat", new=AsyncMock(return_value="X is a thing.")):
        result = await reader_node(state)

    assert len(result["reader_notes"]) == 1
    assert "[1]" in result["reader_notes"][0]
    assert "X Explained" in result["reader_notes"][0]
    assert "X is a thing" in result["reader_notes"][0]


@pytest.mark.asyncio
async def test_reader_filters_irrelevant_sources():
    """Kalau LLM return NOT_RELEVANT, source di-skip dari notes."""
    state = initial_state("Test")
    state["search_results"] = [
        {
            "sub_question": "Q1",
            "results": [
                {"title": "Irrelevant", "url": "https://x.com", "content": "unrelated", "score": 0.3, "sub_question": "Q1"},
                {"title": "Relevant", "url": "https://y.com", "content": "answer here", "score": 0.9, "sub_question": "Q1"},
            ],
        },
    ]

    responses = ["NOT_RELEVANT", "answer here"]
    call_count = {"n": 0}

    async def _side_effect(*args, **kwargs):
        r = responses[call_count["n"]]
        call_count["n"] += 1
        return r

    with patch("src.nodes.reader.chat", new=AsyncMock(side_effect=_side_effect)):
        result = await reader_node(state)

    assert len(result["reader_notes"]) == 1
    assert "Relevant" in result["reader_notes"][0]


@pytest.mark.asyncio
async def test_reader_handles_empty_search_results():
    state = initial_state("Test")
    state["search_results"] = []

    result = await reader_node(state)
    assert result["reader_notes"] == []


@pytest.mark.asyncio
async def test_reader_survives_llm_error():
    """Kalau LLM raise error, source skip tapi processing continue."""
    from src.llm import LLMError

    state = initial_state("Test")
    state["search_results"] = [
        {
            "sub_question": "Q",
            "results": [
                {"title": "S1", "url": "https://a.com", "content": "c1", "score": 0.5, "sub_question": "Q"},
                {"title": "S2", "url": "https://b.com", "content": "c2", "score": 0.5, "sub_question": "Q"},
            ],
        },
    ]

    responses = [LLMError("mock fail"), "excerpt from S2"]
    call_count = {"n": 0}

    async def _side_effect(*args, **kwargs):
        r = responses[call_count["n"]]
        call_count["n"] += 1
        if isinstance(r, Exception):
            raise r
        return r

    with patch("src.nodes.reader.chat", new=AsyncMock(side_effect=_side_effect)):
        result = await reader_node(state)

    assert len(result["reader_notes"]) == 1
    assert "S2" in result["reader_notes"][0]
