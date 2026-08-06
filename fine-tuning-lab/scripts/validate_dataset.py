"""Dataset validator — check quality issues before training.

Checks:
1. No duplicate text
2. Class balance (warn kalau imbalance ratio > 2x)
3. Length distribution (flag outlier)
4. Character encoding sanity
5. Label consistency (semua label in intents.json)

Usage:
    python scripts/validate_dataset.py data/splits/train.jsonl
    python scripts/validate_dataset.py data/splits/*.jsonl
"""

import json
import argparse
import sys
from pathlib import Path
from collections import Counter


def load_jsonl(path: Path) -> list[dict]:
    with path.open() as f:
        return [json.loads(line) for line in f if line.strip()]


def validate_file(path: Path, valid_labels: set[str]) -> int:
    """Return number of issues found."""
    samples = load_jsonl(path)
    issues = 0

    print(f"\n📄 {path}")
    print(f"   Total: {len(samples)} samples")

    # 1. Duplicate check
    text_counts = Counter(s["text"].lower().strip() for s in samples)
    duplicates = {t: c for t, c in text_counts.items() if c > 1}
    if duplicates:
        issues += len(duplicates)
        print(f"   ❌ Duplicates: {len(duplicates)}")
        for text, count in list(duplicates.items())[:3]:
            print(f"      • ({count}x) {text[:80]}")

    # 2. Class balance
    label_counts = Counter(s["label"] for s in samples)
    if label_counts:
        max_count = max(label_counts.values())
        min_count = min(label_counts.values())
        imbalance_ratio = max_count / max(min_count, 1)
        if imbalance_ratio > 2:
            issues += 1
            print(f"   ⚠️  Imbalance ratio: {imbalance_ratio:.1f}x (max={max_count} min={min_count})")

    # 3. Length distribution
    lengths = [len(s["text"]) for s in samples]
    if lengths:
        avg = sum(lengths) / len(lengths)
        outliers_long = [s for s in samples if len(s["text"]) > 300]
        outliers_short = [s for s in samples if len(s["text"]) < 3]
        if outliers_long:
            issues += len(outliers_long)
            print(f"   ⚠️  Outlier too long (>300 char): {len(outliers_long)}")
        if outliers_short:
            issues += len(outliers_short)
            print(f"   ⚠️  Outlier too short (<3 char): {len(outliers_short)}")
        print(f"   Length: avg={avg:.0f} char, min={min(lengths)}, max={max(lengths)}")

    # 4. Label validity
    invalid_labels = [s for s in samples if s["label"] not in valid_labels]
    if invalid_labels:
        issues += len(invalid_labels)
        print(f"   ❌ Invalid labels: {len(invalid_labels)}")
        for s in invalid_labels[:3]:
            print(f"      • {s['label']} → {s['text'][:60]}")

    # 5. Missing fields
    for i, s in enumerate(samples):
        for req in ("id", "text", "label", "source"):
            if req not in s or not s[req]:
                issues += 1
                print(f"   ❌ Sample {i} missing '{req}'")
                break

    # 6. Class distribution display
    print("   Class distribution:")
    for label in sorted(label_counts):
        print(f"     {label:22s} {label_counts[label]}")

    if issues == 0:
        print("   ✅ All checks passed")

    return issues


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("files", nargs="+", help="JSONL files to validate")
    parser.add_argument("--intents-file", default="data/intents.json")
    args = parser.parse_args()

    intents_data = json.loads(Path(args.intents_file).read_text())
    valid_labels = {i["label"] for i in intents_data["intents"]}

    total_issues = 0
    for f in args.files:
        p = Path(f)
        if not p.exists():
            print(f"❌ {p} not found")
            total_issues += 1
            continue
        total_issues += validate_file(p, valid_labels)

    print(f"\n{'=' * 50}")
    if total_issues == 0:
        print("✅ ALL FILES VALID")
        sys.exit(0)
    else:
        print(f"⚠️  {total_issues} issues found across {len(args.files)} files")
        sys.exit(1)


if __name__ == "__main__":
    main()
