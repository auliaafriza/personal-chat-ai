"""Test cost tracker."""

import pytest
from src.usage import UsageTracker, reset_tracker, get_tracker


@pytest.mark.asyncio
async def test_tracker_accumulates_usage():
    tracker = UsageTracker()
    await tracker.record("planner", "llama-3.3-70b-versatile",
                         {"prompt_tokens": 500, "completion_tokens": 100})
    await tracker.record("writer", "llama-3.3-70b-versatile",
                         {"prompt_tokens": 1000, "completion_tokens": 500})

    assert tracker.total_input() == 1500
    assert tracker.total_output() == 600
    # Cost: (1500/1M * 0.59) + (600/1M * 0.79) = 0.000885 + 0.000474 = 0.001359
    assert abs(tracker.total_cost() - 0.001359) < 1e-6


@pytest.mark.asyncio
async def test_tracker_summary_grouped_by_node():
    tracker = UsageTracker()
    await tracker.record("planner", "llama-3.3-70b-versatile", {"prompt_tokens": 500, "completion_tokens": 100})
    await tracker.record("planner", "llama-3.3-70b-versatile", {"prompt_tokens": 300, "completion_tokens": 50})
    await tracker.record("writer", "llama-3.3-70b-versatile", {"prompt_tokens": 1000, "completion_tokens": 500})

    s = tracker.summary()
    assert s["total_calls"] == 3
    assert s["by_node"]["planner"]["calls"] == 2
    assert s["by_node"]["planner"]["input"] == 800
    assert s["by_node"]["writer"]["calls"] == 1


@pytest.mark.asyncio
async def test_tracker_handles_missing_usage():
    """Kalau usage None, record no-op."""
    tracker = UsageTracker()
    await tracker.record("x", "llama-3.3-70b-versatile", None)
    assert len(tracker.records) == 0


def test_reset_tracker_creates_new_instance():
    t1 = get_tracker()
    reset_tracker()
    t2 = get_tracker()
    assert t1 is not t2
