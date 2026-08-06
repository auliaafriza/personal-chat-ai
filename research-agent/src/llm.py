"""Thin async wrapper around Groq chat completions API.

Kenapa nggak pakai langchain-groq: biar dependencies minimum + explicit soal apa yang jadi
bagian dari agent (state, graph, nodes) vs LLM call detail. Semua LLM call di project ini
lewat sini, jadi easy swap ke provider lain kalau perlu.
"""

import os
import json
import httpx
from typing import Any

from .usage import get_tracker


GROQ_URL = "https://api.groq.com/openai/v1/chat/completions"


class LLMError(Exception):
    """Raised kalau Groq return error atau parsing fail."""


async def chat(
    messages: list[dict[str, str]],
    *,
    node: str = "unknown",
    model: str | None = None,
    temperature: float = 0.3,
    max_tokens: int = 2048,
    response_format: dict[str, Any] | None = None,
    timeout: float = 60.0,
) -> str:
    """Kirim chat completion ke Groq, return content string.

    Args:
        messages: [{"role": "system"|"user"|"assistant", "content": "..."}]
        node: Label untuk usage tracking (misal "planner", "reader")
        model: Override GROQ_MODEL env var
        temperature: Sampling temperature
        max_tokens: Max output tokens
        response_format: `{"type": "json_object"}` untuk force JSON output
        timeout: HTTP timeout in seconds

    Raises:
        LLMError kalau API return non-2xx atau response format aneh.
    """
    api_key = os.environ.get("GROQ_API_KEY")
    if not api_key:
        raise LLMError("GROQ_API_KEY not set")

    model = model or os.environ.get("GROQ_MODEL", "llama-3.3-70b-versatile")

    payload: dict[str, Any] = {
        "model": model,
        "messages": messages,
        "temperature": temperature,
        "max_tokens": max_tokens,
    }
    if response_format is not None:
        payload["response_format"] = response_format

    async with httpx.AsyncClient(timeout=timeout) as client:
        try:
            resp = await client.post(
                GROQ_URL,
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                },
                json=payload,
            )
        except httpx.HTTPError as e:
            raise LLMError(f"Groq HTTP error: {e}") from e

        if resp.status_code != 200:
            raise LLMError(f"Groq returned {resp.status_code}: {resp.text[:500]}")

        data = resp.json()

        # Record usage untuk cost tracking (best-effort, jangan crash kalau shape aneh)
        try:
            await get_tracker().record(node, model, data.get("usage"))
        except Exception:
            pass

        try:
            return data["choices"][0]["message"]["content"]
        except (KeyError, IndexError) as e:
            raise LLMError(f"Unexpected Groq response shape: {data}") from e


async def chat_json(
    messages: list[dict[str, str]],
    **kwargs: Any,
) -> dict[str, Any]:
    """Wrapper kalau butuh structured JSON output.

    Groq support `response_format={"type": "json_object"}` untuk force valid JSON.
    Model tetep harus di-prompt untuk output JSON — response_format cuma guarantee validity.
    """
    content = await chat(messages, response_format={"type": "json_object"}, **kwargs)
    try:
        return json.loads(content)
    except json.JSONDecodeError as e:
        raise LLMError(f"Groq returned invalid JSON: {content[:500]}") from e
