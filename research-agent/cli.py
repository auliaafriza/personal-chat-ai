"""CLI entry point untuk research agent.

Usage:
    # Basic
    python cli.py "Bandingkan fine-tuning vs RAG"

    # Interactive mode — user approve sub-questions + draft
    python cli.py "Query" --interactive

    # Resume interrupted run
    python cli.py "Query" --thread-id abc123

    # Verbose logging + custom iteration budget
    python cli.py "Query" -v --max-iterations 5

Env:
    Butuh .env dengan GROQ_API_KEY dan TAVILY_API_KEY. Copy .env.example → .env dulu.
"""

import asyncio
import argparse
import logging
import sys
import uuid

from dotenv import load_dotenv
from rich.console import Console
from rich.panel import Panel
from rich.markdown import Markdown
from rich.prompt import Confirm, Prompt
from rich.table import Table

from src.state import initial_state, ResearchState
from src.graph import build_graph
from src.checkpoint import open_checkpointer
from src.usage import reset_tracker, get_tracker


console = Console()


def setup_logging(verbose: bool) -> None:
    level = logging.DEBUG if verbose else logging.WARNING
    logging.basicConfig(
        level=level,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
        datefmt="%H:%M:%S",
    )
    if not verbose:
        logging.getLogger("httpx").setLevel(logging.WARNING)
        logging.getLogger("httpcore").setLevel(logging.WARNING)
    # Node-level info logging tetap muncul di INFO mode
    if not verbose:
        logging.getLogger("src.nodes").setLevel(logging.INFO)
        logging.getLogger("src.tools").setLevel(logging.INFO)


def _print_sub_questions(state: ResearchState) -> None:
    sub_qs = state.get("sub_questions", [])
    console.print(Panel(
        "\n".join(f"{i}. {q}" for i, q in enumerate(sub_qs, 1)),
        title=f"[bold]Sub-questions ({len(sub_qs)})",
        border_style="green",
    ))
    reasoning = state.get("plan_reasoning", "")
    if reasoning:
        console.print(f"[dim italic]Planner reasoning: {reasoning}[/]\n")


def _print_search_summary(state: ResearchState) -> None:
    search_results = state.get("search_results", [])
    total_hits = sum(len(sr["results"]) for sr in search_results)
    console.print(f"[bold]Search results:[/] {total_hits} hits across {len(search_results)} sub-questions")
    for sr in search_results:
        console.print(f"  [cyan]▸[/] {sr['sub_question'][:70]}")
        for r in sr["results"]:
            console.print(f"      [dim]•[/] {r['title'][:80]}")
            console.print(f"        [dim]{r['url']}[/]")
    console.print()


def _print_notes_summary(state: ResearchState) -> None:
    notes = state.get("reader_notes", [])
    console.print(f"[bold]Reader notes:[/] {len(notes)} relevant excerpts extracted\n")


def _print_draft(state: ResearchState) -> None:
    draft = state.get("draft", "")
    if draft:
        console.print(Panel(Markdown(draft), title="[bold]Draft (writer output)", border_style="yellow"))


def _print_critique(state: ResearchState) -> None:
    critique = state.get("critique")
    if not critique:
        return
    color = "green" if critique["approved"] else "red"
    verdict = "APPROVED" if critique["approved"] else "REJECTED"
    console.print(Panel(
        f"[bold {color}]{verdict}[/]\n\n{critique['reasoning']}\n"
        + ("\n[bold]Suggestions:[/]\n" + "\n".join(f"- {s}" for s in critique["suggestions"])
           if critique["suggestions"] else ""),
        title="[bold]Critic verdict",
        border_style=color,
    ))


def _print_cost_summary() -> None:
    summary = get_tracker().summary()
    if summary["total_calls"] == 0:
        return
    table = Table(title="Cost Summary", show_header=True, header_style="bold cyan")
    table.add_column("Node")
    table.add_column("Calls", justify="right")
    table.add_column("Input tok", justify="right")
    table.add_column("Output tok", justify="right")
    table.add_column("Cost (USD)", justify="right")

    for node, stats in summary["by_node"].items():
        table.add_row(
            node,
            str(stats["calls"]),
            f"{stats['input']:,}",
            f"{stats['output']:,}",
            f"${stats['cost']:.4f}",
        )
    table.add_section()
    table.add_row(
        "[bold]TOTAL",
        f"[bold]{summary['total_calls']}",
        f"[bold]{summary['total_input']:,}",
        f"[bold]{summary['total_output']:,}",
        f"[bold]${summary['total_cost_usd']:.4f}",
    )
    console.print()
    console.print(table)


async def _run_non_interactive(query: str, max_iterations: int, thread_id: str | None) -> None:
    """Basic mode — no interrupt, straight through."""
    async with open_checkpointer() as ckpt:
        graph = build_graph(checkpointer=ckpt)
        state_in = initial_state(query, max_iterations=max_iterations)
        config = {"configurable": {"thread_id": thread_id or str(uuid.uuid4())}}

        console.print(f"[dim]Thread ID: {config['configurable']['thread_id']}[/]\n")

        # Ainvoke satu shot — semua node berjalan sampai END
        final = await graph.ainvoke(state_in, config=config)

    _print_sub_questions(final)
    _print_search_summary(final)
    _print_notes_summary(final)
    _print_critique(final)

    final_answer = final.get("final_answer") or final.get("draft", "")
    if final_answer:
        console.print()
        console.print(Panel(Markdown(final_answer), title="[bold]Final Answer", border_style="green"))


async def _run_interactive(query: str, max_iterations: int, thread_id: str | None) -> None:
    """Interactive mode — pause di setiap interrupt point untuk user approve."""
    thread_id = thread_id or str(uuid.uuid4())
    config = {"configurable": {"thread_id": thread_id}}

    console.print(f"[dim]Thread ID: {thread_id} — resume dengan --thread-id {thread_id}[/]\n")

    async with open_checkpointer() as ckpt:
        graph = build_graph(checkpointer=ckpt, interactive=True)
        state_in = initial_state(query, max_iterations=max_iterations)

        # === Step 1: run sampai interrupt sebelum searcher ===
        console.print("[cyan]▶ Step 1: Planner[/]")
        result = await graph.ainvoke(state_in, config=config)
        _print_sub_questions(result)

        # Human review sub-questions
        if not Confirm.ask("[bold yellow]Approve sub-questions dan lanjut ke search?[/]", default=True):
            edited = Prompt.ask(
                "[bold]Enter revised sub-questions (semicolon-separated), atau blank to abort[/]",
                default="",
            )
            if not edited.strip():
                console.print("[red]Aborted[/]")
                return
            new_sub_qs = [q.strip() for q in edited.split(";") if q.strip()]
            # Update state via checkpointer
            await graph.aupdate_state(config, {"sub_questions": new_sub_qs})
            console.print(f"[green]Updated sub-questions:[/] {new_sub_qs}\n")

        # === Step 2: resume, run sampai interrupt sebelum critic ===
        console.print("[cyan]▶ Step 2: Search + Read + Write[/]")
        result = await graph.ainvoke(None, config=config)
        _print_search_summary(result)
        _print_notes_summary(result)
        _print_draft(result)

        if not Confirm.ask("[bold yellow]Approve draft manually (skip critic)?[/]", default=False):
            # === Step 3: resume dengan critic ===
            console.print("[cyan]▶ Step 3: Critic evaluation + potential retry loop[/]")
            result = await graph.ainvoke(None, config=config)
            _print_critique(result)
        else:
            # User approve manual — set final_answer, skip critic
            await graph.aupdate_state(config, {
                "final_answer": result.get("draft", ""),
                "critique": {
                    "approved": True,
                    "reasoning": "Manually approved by user (skip critic).",
                    "suggestions": [],
                },
            })
            result = await graph.aget_state(config)
            result = result.values

        final_answer = result.get("final_answer") or result.get("draft", "")
        if final_answer:
            console.print()
            console.print(Panel(Markdown(final_answer), title="[bold]Final Answer", border_style="green"))


async def run(query: str, max_iterations: int, interactive: bool, thread_id: str | None) -> None:
    reset_tracker()
    console.print(Panel(f"[bold cyan]Query:[/] {query}", title="Research Agent"))
    console.print()

    if interactive:
        await _run_interactive(query, max_iterations, thread_id)
    else:
        await _run_non_interactive(query, max_iterations, thread_id)

    _print_cost_summary()


def main() -> None:
    load_dotenv()

    parser = argparse.ArgumentParser(
        description="Research agent — decompose query, search, synthesize dengan critic loop.",
    )
    parser.add_argument("query", help="Research query")
    parser.add_argument("--max-iterations", type=int, default=3,
                        help="Max writer↔critic loop iterations (default: 3)")
    parser.add_argument("-i", "--interactive", action="store_true",
                        help="Interactive mode — user approve sub-questions + draft")
    parser.add_argument("--thread-id", default=None,
                        help="Resume interrupted run dengan ID ini (default: auto-generated)")
    parser.add_argument("-v", "--verbose", action="store_true", help="Debug logging")
    args = parser.parse_args()

    setup_logging(args.verbose)

    try:
        asyncio.run(run(args.query, args.max_iterations, args.interactive, args.thread_id))
    except KeyboardInterrupt:
        console.print("\n[yellow]Interrupted — resume dengan --thread-id (kalau ada)[/]")
        sys.exit(130)
    except Exception as e:
        console.print(f"\n[bold red]Error:[/] {e}")
        if args.verbose:
            import traceback
            traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
