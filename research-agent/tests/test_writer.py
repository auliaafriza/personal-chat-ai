"""Test untuk writer node."""

import pytest
from unittest.mock import patch, AsyncMock

from src.state import initial_state
from src.nodes.writer import writer_node


@pytest.mark.asyncio
async def test_writer_synthesizes_from_notes():
    state = initial_state("What is RAG?")
    state["sub_questions"] = ["What is RAG?", "How does it work?"]
    state["reader_notes"] = [
        "[1] Wikipedia (https://x.com): RAG combines retrieval and generation.",
    ]

    mock_draft = "# RAG\n\nRAG combines retrieval and generation [1]."
    with patch("src.nodes.writer.chat", new=AsyncMock(return_value=mock_draft)):
        result = await writer_node(state)

    assert result["draft"] == mock_draft
    assert result["iteration"] == 1


@pytest.mark.asyncio
async def test_writer_handles_no_notes():
    """Kalau reader_notes kosong, writer return minimal apologetic answer."""
    state = initial_state("Test")
    state["reader_notes"] = []

    result = await writer_node(state)
    assert "Maaf" in result["draft"] or "rephrase" in result["draft"]
    assert result["iteration"] == 1


@pytest.mark.asyncio
async def test_writer_incorporates_critique_suggestions_on_retry():
    """Kalau iteration>0 dan ada critique suggestions, writer prompt harus include."""
    state = initial_state("Test")
    state["reader_notes"] = ["[1] source: content"]
    state["iteration"] = 1
    state["critique"] = {
        "approved": False,
        "reasoning": "Missing coverage",
        "suggestions": ["Add example", "Cite source [1] properly"],
    }

    captured_messages = []

    async def _capture(messages, **kwargs):
        captured_messages.append(messages)
        return "revised draft"

    with patch("src.nodes.writer.chat", new=AsyncMock(side_effect=_capture)):
        result = await writer_node(state)

    # User prompt harus include suggestions
    user_content = captured_messages[0][1]["content"]
    assert "Add example" in user_content
    assert "Cite source [1] properly" in user_content
    assert result["iteration"] == 2
