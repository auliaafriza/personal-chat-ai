"""Synthetic data generator via Groq LLM.

Untuk setiap intent, generate N samples baru dengan diversity requirements:
- Register mix (informal 60%, semi-formal 30%, formal 10%)
- Length variation (short/medium/long)
- Typo tolerance (15%)
- Code-switching (10%)
- Emoji usage (20%)

Prompt LLM dengan seed samples + intent description, minta output JSONL batch.
Dedupe di sisi client.

Usage:
    python scripts/generate_synthetic.py --output data/synthetic_v1.jsonl --samples-per-intent 55

Butuh GROQ_API_KEY di .env.
"""

import os
import json
import argparse
import asyncio
import hashlib
import re
from pathlib import Path
from typing import Any

import httpx
from dotenv import load_dotenv
from tqdm import tqdm


GROQ_URL = "https://api.groq.com/openai/v1/chat/completions"


GENERATOR_SYSTEM_PROMPT = """Kamu adalah data augmentation assistant untuk task Indonesian customer support intent classification.

Task: generate N samples baru untuk 1 intent tertentu, dengan diversity:
- Register mix: 60% informal (gw/loe/min), 30% semi-formal (saya/kak/mba), 10% formal
- Length: 30% pendek (<20 kata), 50% medium (20-40 kata), 20% panjang (40-80 kata)
- 15% samples pakai typo/singkatan natural (gmn, gk, trm, dll, blm)
- 10% samples pakai code-switching Indonesian + English (checkout, cancel, refund, etc)
- 20% samples pakai emoji (:), :(, 🙏, 😭, etc)

Aturan STRICT:
1. Samples HARUS unambiguous match intent target — jangan bikin yang bisa di-classify jadi intent lain
2. JANGAN copy-paste seed samples, harus phrasing baru
3. Realistic — meniru cara user real chat ke customer support Indonesian e-commerce (Shopee, Tokopedia)
4. Setiap sample harus complete sentence/question, bukan fragment

Output format: JSON dengan struktur:
{
  "samples": [
    {"text": "sample text here", "reasoning": "why this fits the intent"},
    ...
  ]
}

Output cuma JSON valid, no additional text."""


async def generate_batch(
    client: httpx.AsyncClient,
    intent: dict,
    seed_examples: list[str],
    batch_size: int,
    api_key: str,
    model: str,
) -> list[str]:
    """Generate 1 batch of samples untuk 1 intent."""
    seed_text = "\n".join(f'- "{s}"' for s in seed_examples[:5])

    user_prompt = f"""Intent target: **{intent['label']}**
Description: {intent['description_id']}

Contoh seed samples untuk intent ini:
{seed_text}

Generate {batch_size} samples BARU untuk intent ini yang match description tapi phrasing berbeda dari seed."""

    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": GENERATOR_SYSTEM_PROMPT},
            {"role": "user", "content": user_prompt},
        ],
        "temperature": 0.9,  # High diversity
        "max_tokens": 2000,
        "response_format": {"type": "json_object"},
    }

    try:
        resp = await client.post(
            GROQ_URL,
            headers={"Authorization": f"Bearer {api_key}"},
            json=payload,
            timeout=60.0,
        )
        resp.raise_for_status()
    except httpx.HTTPError as e:
        print(f"[warn] intent={intent['label']} batch failed: {e}")
        return []

    data = resp.json()
    content = data["choices"][0]["message"]["content"]

    try:
        parsed = json.loads(content)
    except json.JSONDecodeError:
        print(f"[warn] intent={intent['label']} invalid JSON: {content[:200]}")
        return []

    samples = parsed.get("samples", [])
    texts = [str(s.get("text", "")).strip() for s in samples if s.get("text")]
    # Filter garbage
    texts = [t for t in texts if 3 < len(t) < 300]
    return texts


def hash_text(text: str) -> str:
    """Normalize + hash text untuk dedupe."""
    normalized = re.sub(r"\s+", " ", text.lower().strip())
    normalized = re.sub(r"[^\w\s]", "", normalized)  # strip punctuation
    return hashlib.md5(normalized.encode()).hexdigest()


async def generate_for_intent(
    client: httpx.AsyncClient,
    intent: dict,
    seed_by_label: dict[str, list[str]],
    target_count: int,
    batch_size: int,
    api_key: str,
    model: str,
    seen_hashes: set,
) -> list[dict]:
    """Generate target_count samples untuk 1 intent, dedupe on-the-fly."""
    label = intent["label"]
    seed_examples = seed_by_label.get(label, [])
    collected: list[dict] = []
    attempt = 0
    max_attempts = 10  # Kalau LLM sering duplicate, cap attempts

    with tqdm(total=target_count, desc=f"[{label:22s}]", leave=False) as pbar:
        while len(collected) < target_count and attempt < max_attempts:
            attempt += 1
            texts = await generate_batch(client, intent, seed_examples, batch_size, api_key, model)
            for text in texts:
                if len(collected) >= target_count:
                    break
                h = hash_text(text)
                if h in seen_hashes:
                    continue
                seen_hashes.add(h)
                collected.append({
                    "id": f"synth_{label}_{len(collected):04d}",
                    "text": text,
                    "label": label,
                    "source": "synthetic_v1",
                })
                pbar.update(1)

    if len(collected) < target_count:
        print(f"[warn] intent={label} only got {len(collected)}/{target_count} after {attempt} attempts")

    return collected


async def main() -> None:
    load_dotenv()

    parser = argparse.ArgumentParser()
    parser.add_argument("--intents-file", default="data/intents.json")
    parser.add_argument("--seed-file", default="data/seed_samples.jsonl")
    parser.add_argument("--output", default="data/synthetic_v1.jsonl")
    parser.add_argument("--samples-per-intent", type=int, default=55)
    parser.add_argument("--batch-size", type=int, default=10,
                        help="Berapa samples per LLM call (max 15-20 sebelum quality drop)")
    parser.add_argument("--model", default=None)
    args = parser.parse_args()

    api_key = os.environ.get("GROQ_API_KEY")
    if not api_key:
        raise SystemExit("GROQ_API_KEY not set — add to .env")

    model = args.model or os.environ.get("SYNTHETIC_MODEL", "llama-3.3-70b-versatile")

    # Load intents
    intents_path = Path(args.intents_file)
    intents_data = json.loads(intents_path.read_text())
    intents = intents_data["intents"]

    # Load seed samples, group by label
    seed_by_label: dict[str, list[str]] = {}
    with Path(args.seed_file).open() as f:
        for line in f:
            sample = json.loads(line)
            seed_by_label.setdefault(sample["label"], []).append(sample["text"])

    print(f"Loaded {sum(len(v) for v in seed_by_label.values())} seed samples across {len(seed_by_label)} intents")
    print(f"Generating {args.samples_per_intent} samples per intent → {args.samples_per_intent * len(intents)} total")
    print(f"Model: {model}, batch_size: {args.batch_size}\n")

    seen_hashes = set()
    # Include seed hashes untuk dedupe against seed too
    for texts in seed_by_label.values():
        for t in texts:
            seen_hashes.add(hash_text(t))

    all_samples: list[dict] = []
    async with httpx.AsyncClient() as client:
        for intent in intents:
            samples = await generate_for_intent(
                client, intent, seed_by_label,
                args.samples_per_intent, args.batch_size, api_key, model,
                seen_hashes,
            )
            all_samples.extend(samples)

    # Save output
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w") as f:
        for sample in all_samples:
            f.write(json.dumps(sample, ensure_ascii=False) + "\n")

    # Summary
    by_label: dict[str, int] = {}
    for s in all_samples:
        by_label[s["label"]] = by_label.get(s["label"], 0) + 1
    print(f"\n✓ Saved {len(all_samples)} samples to {output_path}")
    for label, count in sorted(by_label.items()):
        print(f"  {label:22s} {count}")


if __name__ == "__main__":
    asyncio.run(main())
