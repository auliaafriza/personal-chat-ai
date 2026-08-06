# Cost Analysis — PersonalChatAI Production

Detailed breakdown biaya per 1M token untuk semua external API. Untuk interview prep ("apakah proyek kamu sustainable secara cost?") dan pricing decision kalau scale up.

**Note:** Semua harga per Agustus 2026. Cek pricing page masing-masing service untuk update terbaru.

## Token pricing per service

### Groq (LLM inference)

| Model | Input ($/1M) | Output ($/1M) | Notes |
|---|---|---|---|
| llama-3.3-70b-versatile | $0.59 | $0.79 | Main chat model |
| llama-3.1-8b-instant | $0.05 | $0.08 | Title gen + LLM judge |

**Cara hitung per chat:**
- Rata-rata input per chat: 3000 token (system prompt 500 + memory 300 + RAG context 1500 + history 700)
- Rata-rata output per chat: 500 token (typical Indonesian response)
- Cost per chat (70B): `(3000/1M × $0.59) + (500/1M × $0.79)` = $0.00177 + $0.000395 = **~$0.00217** (~Rp 34)

**Kalau pakai 8B untuk title/judge:** cost ~10x lebih murah, negligible.

**Free tier Groq:**
- 30 req/min
- 6000 tokens/min
- 30k tokens/day untuk 8B, 12k tokens/day untuk 70B
- Cukup untuk single-user portfolio use.

### Voyage AI (Embeddings + Rerank)

| Endpoint | Model | Price |
|---|---|---|
| Embeddings | voyage-3-lite (512d) | $0.02 per 1M input token |
| Rerank | rerank-2 | $0.05 per 1M input token |

**Cara hitung:**
- Embedding per document upload: dokumen 10-page PDF → ~10k token → cost `10k/1M × $0.02` = **$0.0002 per upload** (~Rp 3)
- Embedding per chat (query): query 20 token → cost negligible
- Rerank per chat: 40 candidate × 200 token = 8000 token → cost `8k/1M × $0.05` = **$0.0004 per chat** (~Rp 6)

**Free tier Voyage:**
- 200M input token/bulan free (jauh di bawah aku pakai)
- Cukup untuk portfolio + moderate production scale

### Tavily (Web search)

| Plan | Search count | Price |
|---|---|---|
| Free | 1000 searches/bulan | $0 |
| Pro | 10k searches/bulan | $30/bulan |

**Cost per search:** Included dalam plan, bukan per-token. Kalau kamu chat 5x/hari dan 1 dari 5 chat trigger web search, monthly usage ~30 search → **free tier cukup**.

### Neon Postgres

| Plan | Storage | Compute | Price |
|---|---|---|---|
| Free | 0.5 GB | 190 compute hours/bulan | $0 |
| Launch | 10 GB | Unlimited | $19/bulan |

**Untuk PersonalChatAI production:**
- Storage rata-rata: 200 MB (dokumen chunks + embeddings + traces + memories)
- Compute: Neon auto-suspend, actual usage 50-100 hours/bulan
- **Fit di free tier**

### Railway (Backend hosting)

| Component | Cost |
|---|---|
| Small service (0.5 GB RAM) | ~$3/bulan |
| Volume 1 GB | Free |
| Bandwidth | Free tier generous |
| **Total** | **~$3-4/bulan** |

**Free credit:** $5/bulan untuk hobby developers. Cukup untuk cover cost.

### Vercel (Frontend hosting)

**Hobby plan:** Free. Unlimited requests, 100 GB bandwidth. Cukup untuk portfolio.

### Google Cloud (OAuth + Calendar + Gmail API)

**Free tier:** 1M requests/bulan untuk Calendar + Gmail API. **Fit** untuk single-user.

## Total monthly cost breakdown

**Scenario: single user, 5 chat/hari, 20 hari kerja/bulan, upload 5 dokumen/bulan.**

| Service | Usage | Cost |
|---|---|---|
| Groq LLM | 100 chat × $0.00217 | **$0.22** |
| Voyage embed (upload) | 5 doc × $0.0002 | $0.001 |
| Voyage rerank (chat) | 100 × $0.0004 | $0.04 |
| Voyage embed (query) | 100 × ~$0 | negligible |
| Tavily search | ~20/bulan | Free tier |
| Neon Postgres | 200MB, 100hr compute | Free tier |
| Railway BE | Small service + volume | $3-4 |
| Vercel FE | Hobby | Free |
| Google APIs | Personal use | Free |
| **Total** | | **~$3-5/bulan** |

**Kalau skip Railway (pakai serverless alternative):** free entirely.

## Cost scaling assumption

Kalau app growth ke 100 concurrent users, 30 chat/user/hari:

| Service | Usage @ scale | Cost |
|---|---|---|
| Groq LLM | 100 × 30 × 30 × $0.00217 | **$195/bulan** |
| Voyage embed + rerank | ~$5-10/bulan | |
| Tavily | Need Pro plan | $30/bulan |
| Neon | Need Launch plan | $19/bulan |
| Railway | Need Pro plan | $20/bulan |
| Vercel | Still Hobby OK | Free |
| **Total** | | **~$270-280/bulan** |

**Per-user cost:** ~$2.70/bulan. Kalau charge $10/bulan subscription, gross margin ~73%.

## Cost optimization strategies

Kalau perlu potong cost:

**1. Cache LLM response untuk query yang sering muncul.**
- Query "apa itu useState React" akan sama untuk banyak user
- Cache di Redis dengan TTL 1 hari
- Bisa cut 20-30% LLM cost

**2. Model tiering: pakai 8B untuk tugas simple.**
- Title generation, memory extraction, LLM judge → gunakan Llama 3.1 8B (10x cheaper)
- Main chat → Llama 3.3 70B
- Sekarang implementation aku sudah begini

**3. Truncate history yang panjang.**
- Setelah 20 turn, summarize old turns dengan 8B model, replace dengan summary
- Cut input token cost 30-50% untuk long conversation

**4. Aggressive chunking + smaller embedding dimension.**
- Chunk 500 token vs 1000 token → 2x lebih banyak chunks tapi lebih relevant retrieval
- voyage-3-lite 512d vs OpenAI 1536d → 3x storage saving, quality similar

**5. Rerank stage adaptive.**
- Skip rerank kalau top-5 by RRF score sudah confident (score gap besar)
- Save ~$0.0004/chat × 50% chat = potensial 25% Voyage cost saving

## Sensitivity analysis

**Faktor paling sensitif ke cost:** LLM output token. Karena Groq output $0.79/1M vs input $0.59/1M (33% lebih mahal).

**Optimization biggest ROI:**
- Output token control (max_tokens param, prompt engineering "respond in max 3 paragraphs")
- Model tiering (8B untuk tugas non-chat)

## Interview talking points

**Q: "Apakah proyek kamu sustainable secara cost?"**

> "Untuk personal use production, cost ~$3-5/bulan, mostly ditutup free tier Neon + Groq. Kalau scale ke 100 concurrent users, cost naik ke ~$280/bulan atau $2.70 per user. Dengan pricing model $10/bulan subscription, margin ~73%. Optimization gap masih terbuka: caching frequent query bisa cut 20-30% LLM cost, model tiering 8B vs 70B untuk task berbeda sudah aku implement, history summarization bisa cut long-context cost 30-50%."

**Q: "Kenapa Voyage AI, bukan OpenAI embeddings?"**

> "Voyage voyage-3-lite $0.02/1M vs OpenAI text-embedding-3-small $0.02/1M — harga sama tapi Voyage 512d vs OpenAI 1536d berarti 3x storage saving di pgvector. Quality benchmark comparable di MTEB. Voyage juga punya rerank-2 native, integration lebih smooth. Trade-off: Voyage ecosystem smaller, less mature docs."

**Q: "Kalau ada budget constraint ketat, apa yang kamu potong dulu?"**

> "Pertama, disable rerank stage — save Voyage rerank cost (~$0.0004/chat), quality drop dari recall@5 0.90 ke 0.78 (masih acceptable untuk most use case). Kedua, aggressive caching + shorter response. Ketiga, model downgrade ke Llama 3.1 8B untuk sebagian besar chat, escalate ke 70B hanya kalau user complain."
