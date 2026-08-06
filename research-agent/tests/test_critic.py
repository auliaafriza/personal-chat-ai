"""Test untuk critic node."""

import pytest
from unittest.mock import patch, AsyncMock

from src.state import initial_state
from src.nodes.critic import critic_node


@pytest.mark.asyncio
async def test_critic_approves_good_draft():
    state = initial_state("Test", max_iterations=3)
    state["draft"] = "Answer with [1] citation."
    state["iteration"] = 1
    state["reader_notes"] = ["[1] src: content"]

    mock_response = {
        "approved": True,
        "reasoning": "Good coverage + citation.",
        "suggestions": [],
    }
    with patch("src.nodes.critic.chat_json", new=AsyncMock(return_value=mock_response)):
        result = await critic_node(state)

    assert result["critique"]["approved"] is True
    assert result["final_answer"] == state["draft"]


@pytest.mark.asyncio
async def test_critic_rejects_bad_draft_with_suggestions():
    state = initial_state("Test", max_iterations=3)
    state["draft"] = "Weak draft without citations."
    state["iteration"] = 1
    state["reader_notes"] = ["[1] src: content"]

    mock_response = {
        "approved": False,
        "reasoning": "No citations found.",
        "suggestions": ["Add [1] citations", "Restructure"],
    }
    with patch("src.nodes.critic.chat_json", new=AsyncMock(return_value=mock_response)):
        result = await critic_node(state)

    assert result["critique"]["approved"] is False
    assert len(result["critique"]["suggestions"]) == 2
    assert "final_answer" not in result  # Not approved, jangan set final


@pytest.mark.asyncio
async def test_critic_force_approves_at_max_iterations():
    """Prevent infinite loop: kalau iteration >= max, force approve tanpa LLM call."""
    state = initial_state("Test", max_iterations=3)
    state["draft"] = "Whatever draft."
    state["iteration"] = 3  # Sudah max

    # LLM shouldn't be called
    with patch("src.nodes.critic.chat_json", new=AsyncMock(side_effect=RuntimeError("shouldn't be called"))):
        result = await critic_node(state)

    assert result["critique"]["approved"] is True
    assert "Max iterations" in result["critique"]["reasoning"]
    assert result["final_answer"] == state["draft"]


@pytest.mark.asyncio
async def test_critic_fail_safe_approve_on_llm_error():
    """Kalau LLM fail, critic force approve (fail-safe, jangan block user)."""
    from src.llm import LLMError

    state = initial_state("Test")
    state["draft"] = "Draft"
    state["iteration"] = 1

    with patch("src.nodes.critic.chat_json", new=AsyncMock(side_effect=LLMError("fail"))):
        result = await critic_node(state)

    assert result["critique"]["approved"] is True
    assert result["final_answer"] == "Draft"


@pytest.mark.asyncio
async def test_critic_no_draft_returns_reject():
    state = initial_state("Test")
    state["draft"] = ""

    result = await critic_node(state)
    assert result["critique"]["approved"] is False
