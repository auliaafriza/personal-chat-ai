"""SQLite checkpointer wrapper untuk resume interrupted runs.

Kenapa perlu:
- Interactive mode (interrupt_before) butuh checkpointer supaya state persist antara pause + resume.
- Kalau CLI di-Ctrl+C tengah workflow, bisa resume dengan --thread-id yang sama.
- Debug: bisa inspect state di step tertentu tanpa re-run.

Default location: ./checkpoints.sqlite di cwd. Override via CHECKPOINT_DB env.
"""

import os
import aiosqlite  # noqa: F401  (imported oleh langgraph-checkpoint-sqlite runtime)
from typing import AsyncIterator
from contextlib import asynccontextmanager

from langgraph.checkpoint.sqlite.aio import AsyncSqliteSaver


DEFAULT_DB_PATH = "./checkpoints.sqlite"


@asynccontextmanager
async def open_checkpointer(db_path: str | None = None) -> AsyncIterator[AsyncSqliteSaver]:
    """Async context manager untuk SQLite checkpointer.

    Usage:
        async with open_checkpointer() as ckpt:
            graph = build_graph(checkpointer=ckpt, interactive=True)
            await graph.ainvoke(state, config={"configurable": {"thread_id": "abc"}})
    """
    db_path = db_path or os.environ.get("CHECKPOINT_DB", DEFAULT_DB_PATH)
    async with AsyncSqliteSaver.from_conn_string(db_path) as saver:
        yield saver
