# Task Spec — Indonesian Customer Support Intent Classification

Fine-tuning mini-project untuk Roadmap v2 Minggu 6-7. Task pick + justification, dataset schema, dan evaluation methodology.

## Task selection: kenapa intent classification?

Roadmap kasih 2 opsi:
1. SQL generation dari query bahasa Indonesia
2. Intent classification untuk customer support Indonesian

**Pick: intent classification.** Alasan:

| Criteria | SQL Gen | Intent Classification |
|---|---|---|
| Cocok untuk 3B model | ❌ Marginal — SQL gen butuh reasoning kompleks, 3B sering fail | ✅ Well-suited — classification 8-12 class dalam scope 3B capability |
| Eval clarity | 🟡 Butuh SQL execution env, error handling nested query | ✅ Simple: accuracy + macro F1 dari confusion matrix |
| Dataset acquisition | ❌ Butuh Spider/WikiSQL, translate ke Indonesian, mahal + lambat | ✅ Manual seed + synthetic augmentation via LLM, 1-2 hari |
| Real-world use case | ✅ SQL agent = bermanfaat untuk data analyst | ✅ Router-first-step untuk chatbot production, sangat common |
| Portfolio impact | ✅ Impressive tapi kalau baseline lemah, ceritanya nggak enak | ✅ Clear improvement story: baseline 60% → fine-tuned 90% |
| Timeline 1 minggu | ❌ Risk overrun karena eval complex | ✅ Realistic scope |

**Bonus:** intent classification hasil bisa langsung integrate ke PersonalChatAI sebagai "intent router" — 2 project connect, portfolio narrative kuat.

## Task definition

**Input:** Text message dari user customer support e-commerce, dalam Bahasa Indonesia (informal), max ~200 karakter.

Contoh:
- "gan pesanan gw kapan nyampe?"
- "duit refund udah dikirim belom min"
- "produk yg gw beli beda sama gambar :("

**Output:** 1 dari 10 intent labels (single-label classification).

## Intent schema — 10 labels

| # | Label | Description | Example |
|---|---|---|---|
| 1 | `check_order_status` | Tanya progress pengiriman, tracking, estimasi sampai | "pesanan saya kapan sampai?", "cek resi dong" |
| 2 | `request_refund` | Minta pengembalian uang, komplain refund belum masuk | "duit refund kapan cair?", "mau refund pesanan #12345" |
| 3 | `product_complaint` | Komplain produk (rusak, tidak sesuai, palsu) | "barang datang rusak", "produk beda sama foto" |
| 4 | `product_inquiry` | Tanya spek/stok/varian/harga produk | "ada ukuran L?", "warna hitam masih ready?" |
| 5 | `payment_issue` | Masalah pembayaran (gagal, double charge, metode) | "bayar gagal terus", "kena charge 2 kali" |
| 6 | `shipping_issue` | Masalah pengiriman selain tracking (rusak paket, salah alamat, kurir) | "paket rusak pas dibuka", "kurir salah alamat" |
| 7 | `cancel_order` | Minta cancel pesanan yang sudah dibuat | "batalin pesanan gw dong", "cancel order #789" |
| 8 | `account_help` | Masalah akun (login, verifikasi, ganti data) | "gabisa login", "lupa password", "verifikasi email" |
| 9 | `promo_voucher` | Tanya promo, voucher, cashback, event | "voucher 50rb bisa dipake?", "ada promo apa?" |
| 10 | `general_greeting` | Salam, thank you, small talk, tidak actionable | "halo min", "makasih ya", "ok siap" |

Intent 1-9 = actionable (butuh routing ke tim spesifik). Intent 10 = fallback (respond generic).

## Dataset spec

**Target size:** 600 total samples
- Train: 420 (70%)
- Val: 90 (15%)
- Test: 90 (15%)

**Per-intent distribution:** 60 samples per intent (balanced). Rationale: 10-class classification dengan small model — imbalance bikin minority class recall drop.

**Diversity requirements:**
- **Register mix:** 60% informal ("gw", "loe", "min"), 30% semi-formal ("saya", "kak", "min"), 10% formal ("saya", "mohon")
- **Length variation:** 30% <20 kata, 50% 20-40 kata, 20% 40-80 kata
- **Typo tolerance:** 15% samples dengan intentional typo/singkatan ("gmn", "gk", "trm ksh")
- **Code-switching:** 10% samples dengan sedikit English ("checkout error", "cancel dong")
- **Emojis:** 20% samples pakai emoji

**Format:** JSONL, 1 sample per line.

```json
{"text": "gan pesanan gw kapan nyampe? udah 3 hari nih", "label": "check_order_status", "id": "sample_0001", "source": "seed"}
```

Field `source`:
- `seed` = manual-written (~50 samples, 5 per intent)
- `synthetic_v1` = generated via Groq LLM (~550 samples)

## Baseline expectations

**Zero-shot Llama 3.2 3B via prompt classification:**
- Accuracy estimate: 55-65% (10-class multiple choice, small model)
- Macro F1: 50-60% (imbalance di prediction distribution)
- Weakness: confuse intent yang mirip (product_complaint vs product_inquiry)

**Few-shot Llama 3.2 3B (3 examples per intent in prompt):**
- Accuracy estimate: 65-75%
- Macro F1: 60-70%
- Weakness: prompt jadi panjang (~2000 token), latency naik

**Fine-tuned Llama 3.2 3B dengan LoRA (target Minggu 7):**
- Accuracy target: 90%+
- Macro F1 target: 88%+
- Weakness: mungkin overfit ke synthetic phrasing kalau data augmentation kurang divers

**Success criteria:** ≥ 20 percentage point improvement dari best baseline (few-shot) ke fine-tuned. Kalau achievable, bisa ceritakan di CV: "Fine-tuned Llama 3.2 3B for Indonesian intent classification — accuracy 68% → 92%".

## Evaluation methodology

**Metrics:**
1. **Accuracy** — overall correct predictions / total
2. **Macro F1** — average F1 across all classes (equal weight), penalize minority class failure
3. **Per-class precision/recall** — identify which intent hardest to distinguish
4. **Confusion matrix** — visualize mis-classification patterns

**Statistical robustness:**
- Report mean ± std dari 3 seed run (biar bukan lucky-seed result)
- Bootstrap confidence interval untuk accuracy (95%)

**Qualitative analysis:**
- Sample 20 misclassification, inspect patterns
- Categorize failure modes: ambiguity, typo, code-switch, adversarial

## Timeline

**Minggu 6 (this week):**
- [x] Task spec + intent schema (this doc)
- [ ] 50 manual seed samples
- [ ] Synthetic data generator script (Groq-based)
- [ ] Generate 550+ synthetic samples
- [ ] Dataset split + validation
- [ ] Baseline eval script (Colab)
- [ ] Baseline metrics recorded

**Minggu 7:**
- [ ] Fine-tune Llama 3.2 3B dengan LoRA (r=8, alpha=16, lr=2e-4, 3-5 epochs)
- [ ] Eval fine-tuned vs baseline, comparison report
- [ ] Push model ke HuggingFace Hub
- [ ] Gradio demo di HF Spaces
- [ ] Blog post + demo video
