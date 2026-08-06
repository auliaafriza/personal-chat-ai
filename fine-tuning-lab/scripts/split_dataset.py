"""Stratified train/val/test split.

Ensure per-class distribution consistent across splits.
Default: 70/15/15.

Usage:
    python scripts/split_dataset.py --input data/seed_samples.jsonl,data/synthetic_v1.jsonl \\
        --out-dir data/splits --seed 42
"""

import json
import argparse
import random
from pathlib import Path
from collections import defaultdict


def load_jsonl(path: Path) -> list[dict]:
    samples = []
    with path.open() as f:
        for line in f:
            line = line.strip()
            if line:
                samples.append(json.loads(line))
    return samples


def save_jsonl(samples: list[dict], path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as f:
        for s in samples:
            f.write(json.dumps(s, ensure_ascii=False) + "\n")


def stratified_split(samples: list[dict], train_ratio: float, val_ratio: float, seed: int):
    """Split samples stratified by label."""
    by_label: dict[str, list[dict]] = defaultdict(list)
    for s in samples:
        by_label[s["label"]].append(s)

    rng = random.Random(seed)
    train, val, test = [], [], []

    for label, items in by_label.items():
        rng.shuffle(items)
        n = len(items)
        n_train = int(n * train_ratio)
        n_val = int(n * val_ratio)
        # Test dapat sisa (biasanya n - n_train - n_val)

        train.extend(items[:n_train])
        val.extend(items[n_train:n_train + n_val])
        test.extend(items[n_train + n_val:])

    # Shuffle each split (jangan biarkan urutan by-label)
    rng.shuffle(train)
    rng.shuffle(val)
    rng.shuffle(test)

    return train, val, test


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True,
                        help="Comma-separated JSONL files to merge")
    parser.add_argument("--out-dir", default="data/splits")
    parser.add_argument("--train-ratio", type=float, default=0.70)
    parser.add_argument("--val-ratio", type=float, default=0.15)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    # Load + merge
    input_files = [Path(p.strip()) for p in args.input.split(",")]
    all_samples: list[dict] = []
    for f in input_files:
        loaded = load_jsonl(f)
        all_samples.extend(loaded)
        print(f"Loaded {len(loaded)} from {f}")

    print(f"\nTotal: {len(all_samples)} samples")

    # Dedupe by text (safety net)
    seen = set()
    deduped = []
    for s in all_samples:
        key = s["text"].lower().strip()
        if key not in seen:
            seen.add(key)
            deduped.append(s)
    if len(deduped) < len(all_samples):
        print(f"Deduped: removed {len(all_samples) - len(deduped)} duplicate")
    all_samples = deduped

    # Class distribution
    by_label: dict[str, int] = defaultdict(int)
    for s in all_samples:
        by_label[s["label"]] += 1
    print("\nClass distribution (before split):")
    for label, count in sorted(by_label.items()):
        print(f"  {label:22s} {count}")

    # Split
    train, val, test = stratified_split(
        all_samples,
        train_ratio=args.train_ratio,
        val_ratio=args.val_ratio,
        seed=args.seed,
    )

    print(f"\nSplit sizes: train={len(train)}, val={len(val)}, test={len(test)}")

    # Verify stratification
    for split_name, split_data in [("train", train), ("val", val), ("test", test)]:
        dist = defaultdict(int)
        for s in split_data:
            dist[s["label"]] += 1
        print(f"\n{split_name.upper()} distribution:")
        for label, count in sorted(dist.items()):
            print(f"  {label:22s} {count}")

    # Save
    out_dir = Path(args.out_dir)
    save_jsonl(train, out_dir / "train.jsonl")
    save_jsonl(val, out_dir / "val.jsonl")
    save_jsonl(test, out_dir / "test.jsonl")

    print(f"\n✓ Splits saved to {out_dir}/")


if __name__ == "__main__":
    main()
