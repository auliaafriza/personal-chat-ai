"""Critic node — evaluate draft, approve atau suggest improvements.

Evaluate 4 dimensi:
1. Coverage — semua sub_questions terjawab?
2. Citation integrity — setiap claim ada [N] citation? Nomor cocok dengan notes yang ada?
3. Faithfulness — no hallucination, semua fact ada di notes?
4. Clarity — structure logis, bahasa jelas?

Output structured JSON via response_format json_object. Kalau iteration >= max, force approve
untuk cegah infinite loop + budget overshoot.
"""

import logging
import re
from ..state import ResearchState, Critique
from ..llm import chat_json, LLMError


log = logging.getLogger(__name__)


CRITIC_SYSTEM_PROMPT = """Kamu adalah research critic. Task: evaluate draft answer + kasih approve/reject decision.

Evaluate 4 dimensi:
1. **Coverage** — apakah draft cover semua sub-questions yang dilist?
2. **Citation integrity** — apakah setiap claim ada [N] citation? Nomor valid (ada di notes)?
3. **Faithfulness** — apakah semua claim didukung sumber (no hallucination)?
4. **Clarity** — apakah structure logis + bahasa jelas?

Standard approval:
- APPROVE kalau 4 dimensi decent (nggak harus perfect — >75% pass)
- REJECT kalau ada kekurangan major di ≥1 dimensi

Format output JSON:
{
  "approved": true/false,
  "reasoning": "1-3 kalimat kenapa approve/reject",
  "suggestions": ["fix 1", "fix 2"]  // MAX 3, spesifik + actionable, kosong kalau approved
}

Response harus valid JSON, no additional text."""


async def critic_node(state: ResearchState) -> dict:
    """Critic node handler.

    Kalau iteration reached max, force approve tanpa call LLM.
    Kalau tidak, panggil LLM structured evaluation.
    """
    iteration = state.get("iteration", 0)
    max_iter = state.get("max_iterations", 3)
    draft = state.get("draft", "")
    query = state["query"]
    sub_questions = state.get("sub_questions", [])
    notes = state.get("reader_notes", [])

    # Budget cap: kalau sudah max iteration, force approve (prevent infinite loop)
    if iteration >= max_iter:
        log.info(f"[critic] max iterations ({max_iter}) reached — force approve")
        critique: Critique = {
            "approved": True,
            "reasoning": f"Max iterations ({max_iter}) reached, force approve untuk cegah infinite loop.",
            "suggestions": [],
        }
        return {"critique": critique, "final_answer": draft}

    if not draft:
        log.warning("[critic] no draft to evaluate — auto-reject")
        return {
            "critique": {
                "approved": False,
                "reasoning": "No draft di-provide.",
                "suggestions": ["Writer harus generate draft dulu."],
            }
        }

    log.info(f"[critic] evaluating draft (iteration {iteration}, {len(notes)} notes)")

    # Extract citation numbers dari draft untuk validation
    cited_ns = set(int(m) for m in re.findall(r"\[(\d+)\]", draft))
    valid_ns = set(range(1, len(notes) + 1))
    invalid_cites = cited_ns - valid_ns
    citation_stats = (
        f"Draft cite {len(cited_ns)} unique [N]. "
        f"Available notes: 1-{len(notes)}. "
        f"Invalid citations (nggak match notes): {sorted(invalid_cites) if invalid_cites else 'none'}"
    )

    user_content = (
        f"Original query: {query}\n\n"
        f"Sub-questions:\n"
        + "\n".join(f"- {sq}" for sq in sub_questions)
        + f"\n\n[Citation check] {citation_stats}\n\n"
        f"Draft to evaluate:\n{draft}\n\n"
        f"Evaluate + output JSON dengan format yang di-spec."
    )

    messages = [
        {"role": "system", "content": CRITIC_SYSTEM_PROMPT},
        {"role": "user", "content": user_content},
    ]

    try:
        result = await chat_json(messages, node="critic", temperature=0.2, max_tokens=800)
    except LLMError as e:
        log.error(f"[critic] LLM fail: {e} — force approve (fail-safe)")
        return {
            "critique": {
                "approved": True,
                "reasoning": f"Critic LLM failed ({e}), fail-safe approve.",
                "suggestions": [],
            },
            "final_answer": draft,
        }

    approved = bool(result.get("approved", False))
    reasoning = str(result.get("reasoning", ""))
    suggestions = result.get("suggestions", [])
    if not isinstance(suggestions, list):
        suggestions = []
    suggestions = [str(s).strip() for s in suggestions if str(s).strip()][:3]

    log.info(f"[critic] verdict: {'APPROVED' if approved else 'REJECTED'} — {reasoning[:100]}")
    if not approved and suggestions:
        for i, s in enumerate(suggestions, 1):
            log.info(f"[critic]   suggestion {i}: {s}")

    critique = Critique(approved=approved, reasoning=reasoning, suggestions=suggestions)

    out = {"critique": critique}
    if approved:
        out["final_answer"] = draft
    return out
