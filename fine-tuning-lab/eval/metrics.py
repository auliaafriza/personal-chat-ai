"""Metrics computation + reporting untuk intent classification.

Reusable module — dipakai baseline eval, fine-tuned eval, dan comparison report.

Key metrics:
- Accuracy
- Macro F1 (equal weight per class — penalize minority failure)
- Weighted F1 (weight by support — closer to accuracy)
- Per-class precision/recall/F1 (identify weakest class)
- Confusion matrix (visualize mis-classification pattern)
- Bootstrap 95% CI untuk accuracy (statistical robustness)
"""

import json
import random
from pathlib import Path
from typing import Any


def _try_import_sklearn():
    """Lazy import — biar module bisa di-parse tanpa sklearn installed."""
    from sklearn.metrics import (
        accuracy_score,
        f1_score,
        precision_recall_fscore_support,
        confusion_matrix,
        classification_report,
    )
    return {
        "accuracy_score": accuracy_score,
        "f1_score": f1_score,
        "precision_recall_fscore_support": precision_recall_fscore_support,
        "confusion_matrix": confusion_matrix,
        "classification_report": classification_report,
    }


def evaluate(y_true: list[str], y_pred: list[str], labels: list[str]) -> dict[str, Any]:
    """Compute all metrics dari (true, pred) label lists.

    Args:
        y_true: ground truth labels
        y_pred: model predictions
        labels: complete list of possible labels (untuk consistent per-class output)

    Returns:
        Dict dengan keys: accuracy, macro_f1, weighted_f1, per_class, confusion_matrix,
                          accuracy_ci_95 (bootstrap)
    """
    sk = _try_import_sklearn()

    if len(y_true) != len(y_pred):
        raise ValueError(f"Length mismatch: y_true={len(y_true)} y_pred={len(y_pred)}")

    accuracy = float(sk["accuracy_score"](y_true, y_pred))
    macro_f1 = float(sk["f1_score"](y_true, y_pred, labels=labels, average="macro", zero_division=0))
    weighted_f1 = float(sk["f1_score"](y_true, y_pred, labels=labels, average="weighted", zero_division=0))

    precision, recall, f1, support = sk["precision_recall_fscore_support"](
        y_true, y_pred, labels=labels, zero_division=0,
    )

    per_class = {
        label: {
            "precision": float(precision[i]),
            "recall": float(recall[i]),
            "f1": float(f1[i]),
            "support": int(support[i]),
        }
        for i, label in enumerate(labels)
    }

    cm = sk["confusion_matrix"](y_true, y_pred, labels=labels).tolist()

    # Bootstrap 95% CI accuracy
    ci = _bootstrap_accuracy_ci(y_true, y_pred, n_iter=1000)

    return {
        "accuracy": accuracy,
        "accuracy_ci_95": ci,
        "macro_f1": macro_f1,
        "weighted_f1": weighted_f1,
        "per_class": per_class,
        "confusion_matrix": cm,
        "labels": labels,
        "n_samples": len(y_true),
    }


def _bootstrap_accuracy_ci(y_true: list[str], y_pred: list[str], n_iter: int = 1000, alpha: float = 0.05):
    """Bootstrap 95% CI untuk accuracy."""
    n = len(y_true)
    if n < 2:
        return {"lower": 0.0, "upper": 0.0}
    rng = random.Random(42)
    accs = []
    for _ in range(n_iter):
        idx = [rng.randrange(n) for _ in range(n)]
        yt = [y_true[i] for i in idx]
        yp = [y_pred[i] for i in idx]
        accs.append(sum(1 for a, b in zip(yt, yp) if a == b) / n)
    accs.sort()
    lo = accs[int(n_iter * alpha / 2)]
    hi = accs[int(n_iter * (1 - alpha / 2))]
    return {"lower": lo, "upper": hi}


def print_report(metrics: dict[str, Any], title: str = "Evaluation") -> None:
    """Pretty-print metrics ke stdout."""
    print(f"\n{'=' * 60}")
    print(f"  {title}")
    print(f"{'=' * 60}")
    print(f"  N samples:      {metrics['n_samples']}")
    print(f"  Accuracy:       {metrics['accuracy']:.4f} "
          f"(CI 95%: [{metrics['accuracy_ci_95']['lower']:.4f}, {metrics['accuracy_ci_95']['upper']:.4f}])")
    print(f"  Macro F1:       {metrics['macro_f1']:.4f}")
    print(f"  Weighted F1:    {metrics['weighted_f1']:.4f}")
    print(f"\n  Per-class metrics:")
    print(f"  {'Label':<22} {'P':>8} {'R':>8} {'F1':>8} {'N':>6}")
    print(f"  {'-' * 22} {'-' * 8} {'-' * 8} {'-' * 8} {'-' * 6}")

    for label in metrics["labels"]:
        pc = metrics["per_class"][label]
        print(f"  {label:<22} {pc['precision']:>8.3f} {pc['recall']:>8.3f} {pc['f1']:>8.3f} {pc['support']:>6}")


def save_metrics(metrics: dict[str, Any], path: str | Path) -> None:
    """Save metrics ke JSON file."""
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    with p.open("w") as f:
        json.dump(metrics, f, indent=2, ensure_ascii=False)
    print(f"✓ Metrics saved to {p}")


def plot_confusion(metrics: dict[str, Any], save_path: str | Path | None = None):
    """Plot confusion matrix heatmap. Return matplotlib figure."""
    import matplotlib.pyplot as plt
    import seaborn as sns
    import numpy as np

    cm = np.array(metrics["confusion_matrix"])
    labels = metrics["labels"]

    fig, ax = plt.subplots(figsize=(10, 8))
    sns.heatmap(
        cm,
        annot=True,
        fmt="d",
        cmap="Blues",
        xticklabels=labels,
        yticklabels=labels,
        cbar_kws={"label": "Count"},
        ax=ax,
    )
    ax.set_xlabel("Predicted")
    ax.set_ylabel("True")
    ax.set_title(f"Confusion Matrix (acc={metrics['accuracy']:.3f}, macro F1={metrics['macro_f1']:.3f})")
    plt.xticks(rotation=45, ha="right")
    plt.yticks(rotation=0)
    plt.tight_layout()

    if save_path:
        p = Path(save_path)
        p.parent.mkdir(parents=True, exist_ok=True)
        fig.savefig(p, dpi=150, bbox_inches="tight")
        print(f"✓ Confusion matrix saved to {p}")

    return fig


def compare_metrics(baseline: dict[str, Any], improved: dict[str, Any], title: str = "Comparison") -> None:
    """Pretty print comparison antara 2 metrics dict."""
    print(f"\n{'=' * 60}")
    print(f"  {title}")
    print(f"{'=' * 60}")
    print(f"  {'Metric':<20} {'Baseline':>12} {'Improved':>12} {'Δ':>10}")
    print(f"  {'-' * 20} {'-' * 12} {'-' * 12} {'-' * 10}")

    for m in ("accuracy", "macro_f1", "weighted_f1"):
        b = baseline[m]
        i = improved[m]
        delta = i - b
        sign = "+" if delta >= 0 else ""
        print(f"  {m:<20} {b:>12.4f} {i:>12.4f} {sign}{delta:>9.4f}")

    print(f"\n  Per-class F1 delta:")
    for label in improved["labels"]:
        b_f1 = baseline["per_class"][label]["f1"]
        i_f1 = improved["per_class"][label]["f1"]
        delta = i_f1 - b_f1
        sign = "+" if delta >= 0 else ""
        print(f"    {label:<22} {b_f1:>7.3f} → {i_f1:>7.3f}  ({sign}{delta:.3f})")
