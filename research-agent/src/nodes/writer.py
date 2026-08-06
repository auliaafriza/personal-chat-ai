"""Writer node — synthesize final answer dari reader notes.

Input: query, sub_questions, reader_notes (dengan [N] label), plus optional critique
      suggestions kalau iteration > 0.
Output: markdown answer dengan [N] citation yang match source di notes.

Retry behavior: kalau critique reject dengan suggestions, writer prepend suggestions ke prompt
untuk address di next draft.
"""

import logging
from ..state import ResearchState
from ..llm import chat, LLMError


log = logging.getLogger(__name__)


WRITER_SYSTEM_PROMPT = """Kamu adalah research writer. Task: synthesize komprehensif tapi ringkas answer untuk research query, berdasarkan sumber-sumber yang di-provide.

Aturan:
1. **HANYA gunakan info dari notes.** Jangan tambah fakta yang nggak ada di source.
2. **Cite setiap claim** dengan format `[N]` sesuai nomor note. Semua fakta harus punya citation.
3. Structure answer dengan markdown headers per sub-topic yang natural.
4. Kalau notes conflict antar source, acknowledge conflict + present multiple perspectives.
5. Kalau info nggak cukup untuk claim confident, bilang secara eksplisit ("berdasarkan sources yang ada...").
6. **Jangan** hallucinate URL atau source yang tidak ada di notes.
7. Bahasa: match bahasa query (kalau Indonesia, jawab Indonesia; kalau English, jawab English).

Format output: markdown answer, no preamble, no "Berikut jawabannya:" — langsung ke content."""


async def writer_node(state: ResearchState) -> dict:
    """Writer node handler."""
    query = state["query"]
    sub_questions = state.get("sub_questions", [])
    notes = state.get("reader_notes", [])
    iteration = state.get("iteration", 0)
    critique = state.get("critique")

    if not notes:
        log.warning("[writer] no reader_notes, generating minimal answer")
        return {
            "draft": f"# {query}\n\nMaaf, nggak dapat sumber relevan untuk jawab query ini. Coba rephrase atau perluas scope pencarian.",
            "iteration": iteration + 1,
        }

    log.info(f"[writer] synthesizing draft (iteration {iteration + 1}, notes: {len(notes)})")

    # Build user prompt
    user_parts = [
        f"Research query: {query}",
        f"\nSub-questions yang ditelusuri:",
    ]
    for i, sq in enumerate(sub_questions, 1):
        user_parts.append(f"{i}. {sq}")

    user_parts.append(f"\nNotes dari sources ({len(notes)} total):")
    for note in notes:
        user_parts.append(note)

    # Kalau retry (iteration > 0), prepend critic suggestions
    if iteration > 0 and critique and critique.get("suggestions"):
        user_parts.append(
            "\n\n**PENTING — draft sebelumnya di-reject oleh critic. Address suggestions berikut:**"
        )
        for i, sug in enumerate(critique["suggestions"], 1):
            user_parts.append(f"{i}. {sug}")

    user_parts.append(
        "\n\nSynthesize final answer sekarang. Cite setiap claim dengan [N] sesuai nomor note."
    )

    messages = [
        {"role": "system", "content": WRITER_SYSTEM_PROMPT},
        {"role": "user", "content": "\n".join(user_parts)},
    ]

    try:
        draft = await chat(messages, node="writer", temperature=0.5, max_tokens=3000)
    except LLMError as e:
        log.error(f"[writer] LLM fail: {e}")
        draft = f"# {query}\n\n_Writer LLM failed ({e}). Notes gathered:_\n\n" + "\n\n".join(notes[:5])

    return {
        "draft": draft.strip(),
        "iteration": iteration + 1,
    }
