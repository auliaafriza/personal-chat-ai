"""Thread-safe token usage tracker + cost estimation.

Wraps LLM call → collect usage stats → aggregate ke total per run.

Kenapa perlu: agent workflow sering multi-step LLM call (planner, reader per source, writer,
critic, retry). Tanpa tracking, gampang over-spend tanpa sadar.
"""

import asyncio
import logging
from dataclasses import dataclass, field
from typing import Any

log = logging.getLogger(__name__)

# Groq pricing per Aug 2026 — USD per 1M token
GROQ_PRICING = {
    "llama-3.3-70b-versatile": {"input": 0.59, "output": 0.79},
    "llama-3.1-8b-instant":    {"input": 0.05, "output": 0.08},
    "llama-3.1-70b-versatile": {"input": 0.59, "output": 0.79},
}


@dataclass
class UsageRecord:
    """1 LLM call record."""
    node: str
    model: str
    input_tokens: int
    output_tokens: int

    @property
    def cost_usd(self) -> float:
        pricing = GROQ_PRICING.get(self.model)
        if not pricing:
            return 0.0
        return (self.input_tokens / 1_000_000 * pricing["input"]
                + self.output_tokens / 1_000_000 * pricing["output"])


@dataclass
class UsageTracker:
    """Thread-safe accumulator untuk 1 agent run."""
    records: list[UsageRecord] = field(default_factory=list)
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock)

    async def record(self, node: str, model: str, usage: dict[str, Any] | None) -> None:
        """Append 1 record dari Groq response usage field."""
        if not usage:
            return
        input_toks = int(usage.get("prompt_tokens", 0))
        output_toks = int(usage.get("completion_tokens", 0))
        async with self._lock:
            self.records.append(UsageRecord(
                node=node,
                model=model,
                input_tokens=input_toks,
                output_tokens=output_toks,
            ))

    def total_input(self) -> int:
        return sum(r.input_tokens for r in self.records)

    def total_output(self) -> int:
        return sum(r.output_tokens for r in self.records)

    def total_cost(self) -> float:
        return sum(r.cost_usd for r in self.records)

    def summary(self) -> dict[str, Any]:
        """Grouped by node untuk display."""
        by_node: dict[str, dict[str, Any]] = {}
        for r in self.records:
            if r.node not in by_node:
                by_node[r.node] = {"calls": 0, "input": 0, "output": 0, "cost": 0.0}
            by_node[r.node]["calls"] += 1
            by_node[r.node]["input"] += r.input_tokens
            by_node[r.node]["output"] += r.output_tokens
            by_node[r.node]["cost"] += r.cost_usd
        return {
            "by_node": by_node,
            "total_calls": len(self.records),
            "total_input": self.total_input(),
            "total_output": self.total_output(),
            "total_cost_usd": self.total_cost(),
        }


# Module-level singleton — 1 tracker per run (reset via reset())
_current_tracker: UsageTracker | None = None


def get_tracker() -> UsageTracker:
    """Get current tracker, bikin baru kalau belum ada."""
    global _current_tracker
    if _current_tracker is None:
        _current_tracker = UsageTracker()
    return _current_tracker


def reset_tracker() -> None:
    """Reset — dipanggil di awal setiap CLI run."""
    global _current_tracker
    _current_tracker = UsageTracker()
