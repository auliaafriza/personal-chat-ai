# Fine-Tuning Lab

Fine-tune Llama 3.2 3B untuk Indonesian customer support intent classification pakai LoRA.

Part of [Roadmap AI Engineer v2](https://github.com/aulia/portofolio-ai-aulia/blob/main/Roadmap_AI_Engineer_v2.docx) Minggu 6-7 (mini-project B: fine-tuning + LoRA).

## Status

- **Minggu 6 (this week):** ✅ Task spec + dataset schema + seed samples + synthetic generator + split utility + baseline eval notebook
- **Minggu 7 (next week):** 🚧 LoRA fine-tuning + comparison eval + push to HF Hub + Gradio demo + blog post + demo video

## What this lab does

**Task:** Classify Indonesian customer support message ke 1 dari 10 intent (check_order_status, request_refund, product_complaint, dll).

**Model:** meta-llama/Llama-3.2-3B-Instruct.

**Dataset:** 600 samples total (50 seed manual + 550 synthetic via Groq).

**Method:** LoRA fine-tuning (r=8, alpha=16, lr=2e-4, 3-5 epochs) di Colab T4 GPU (free tier).

**Success criteria:** accuracy 90%+ vs baseline zero-shot ~60%, few-shot ~70%.

## Project structure

```
fine-tuning-lab/
├── TASK_SPEC.md              # Task rationale + dataset spec + eval methodology
├── requirements.txt          # Colab deps
├── .env.example              # API keys template (GROQ untuk gen data)
├── data/
│   ├── intents.json          # 10-intent schema dengan description + routing
│   ├── seed_samples.jsonl    # 50 manual seed samples (5 per intent)
│   ├── synthetic_v1.jsonl    # Generated via scripts/generate_synthetic.py
│   └── splits/               # Stratified train/val/test (after split_dataset.py)
├── scripts/
│   ├── generate_synthetic.py # Groq-based synthetic data generator
│   ├── split_dataset.py      # Stratified split (70/15/15)
│   └── validate_dataset.py   # Dedup + balance + length checks
├── eval/
│   ├── __init__.py
│   └── metrics.py            # accuracy, macro F1, per-class, confusion matrix, bootstrap CI
├── notebooks/
│   ├── 01_baseline_eval.ipynb    # ✅ Zero-shot + few-shot Llama 3.2 3B baseline
│   └── 02_finetune_lora.ipynb    # 🚧 Minggu 7 — LoRA fine-tune + comparison
└── results/                  # Generated after eval — metrics JSON + confusion matrix PNG
```

## Setup untuk Minggu 6

### Prasyarat

- Python 3.11+
- Groq API key (untuk synthetic data generation) — dari [console.groq.com/keys](https://console.groq.com/keys)
- HuggingFace account (untuk model access + Colab login) — [huggingface.co/join](https://huggingface.co/join)
- Access grant ke Llama 3.2 3B — request di [huggingface.co/meta-llama/Llama-3.2-3B-Instruct](https://huggingface.co/meta-llama/Llama-3.2-3B-Instruct) (approval biasanya <1 jam)
- Google account untuk Colab

### Step 1: Local setup (data gen + validation)

```bash
cd fine-tuning-lab

# Setup venv
python3 -m venv .venv
source .venv/bin/activate

# Install deps yang diperlukan lokal (subset dari requirements.txt — heavy training deps di Colab)
pip install httpx python-dotenv tqdm scikit-learn

# Configure API key
cp .env.example .env
# Edit .env, isi GROQ_API_KEY
```

### Step 2: Generate synthetic data (5-10 menit)

```bash
# Default: 55 samples per intent × 10 intent = 550 samples
python scripts/generate_synthetic.py --output data/synthetic_v1.jsonl
```

Output: `data/synthetic_v1.jsonl` dengan ~550 samples. Cost: ~$0.10 di Groq free tier (well within limits).

### Step 3: Split dataset

```bash
python scripts/split_dataset.py \
    --input data/seed_samples.jsonl,data/synthetic_v1.jsonl \
    --out-dir data/splits \
    --seed 42
```

Output: `data/splits/{train,val,test}.jsonl` — 70/15/15 stratified split.

### Step 4: Validate dataset

```bash
python scripts/validate_dataset.py data/splits/train.jsonl data/splits/val.jsonl data/splits/test.jsonl
```

Cek: no duplicates, class balance OK (max/min ratio <2x), length distribution reasonable, labels valid.

### Step 5: Baseline eval di Colab (10-15 menit di T4)

1. Upload folder `data/`, `eval/`, dan `notebooks/01_baseline_eval.ipynb` ke Colab (atau clone repo)
2. Runtime → Change runtime type → **GPU (T4)** — free tier OK
3. Open `notebooks/01_baseline_eval.ipynb`
4. Run all cells
5. Save `results/` folder back ke lokal — commit ke repo

### Step 6: Review baseline metrics

Setelah notebook selesai, cek:
- `results/baseline_zeroshot.json` — zero-shot metrics
- `results/baseline_fewshot.json` — few-shot metrics (3 examples per intent)
- `results/baseline_zeroshot_cm.png` — confusion matrix zero-shot
- `results/baseline_fewshot_cm.png` — confusion matrix few-shot
- `results/baseline_summary.md` — ready-to-paste markdown summary

## Setup untuk Minggu 7 (preview)

Notebook `02_finetune_lora.ipynb` akan cover:
- Load Llama 3.2 3B + LoRA adapter (PEFT)
- Format samples ke chat template
- SFTTrainer dengan `TrainingArguments(lr=2e-4, num_train_epochs=4, batch_size=8)`
- Monitor loss curve via TensorBoard
- Eval di test set, bandingin ke baseline (`compare_metrics` dari eval/metrics.py)
- Push adapter ke HF Hub
- Bikin Gradio demo di HF Spaces

Full plan di [TASK_SPEC.md](./TASK_SPEC.md).

## Expected results (Minggu 6 baseline)

Ballpark dari task spec (real numbers akan varying based on data quality):

| Method | Accuracy | Macro F1 |
|---|---|---|
| Zero-shot Llama 3.2 3B | 55-65% | 50-60% |
| Few-shot (3 ex/intent) | 65-75% | 60-70% |
| **Fine-tuned LoRA (target)** | **90%+** | **88%+** |

## Cost breakdown

**Minggu 6 (this week):**
- Groq synthetic gen: ~$0.10
- Colab T4 baseline eval: Free (T4 tier)

**Minggu 7 (LoRA training):**
- Colab T4 training: Free (4 epochs on 400 samples ~30-45 menit)
- Kalau butuh A100/T4-hi: Colab Pro $10/bulan optional
- HF Hub storage adapter: Free

Total lab cost: **~$0.10 kalau pakai Colab free tier**.

## References

- HuggingFace course [chapter 3 (fine-tuning)](https://huggingface.co/learn/nlp-course/chapter3), [chapter 5 (PEFT)](https://huggingface.co/learn/nlp-course/chapter5)
- [LoRA paper](https://arxiv.org/abs/2106.09685) — original
- [PEFT library docs](https://huggingface.co/docs/peft/index)
- [Sebastian Raschka: Fine-tuning LLMs with LoRA](https://www.youtube.com/results?search_query=sebastian+raschka+lora) — recommended video

## License

MIT
