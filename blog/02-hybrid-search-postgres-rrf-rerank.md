---
title: "Hybrid Search in Postgres: Vector + BM25 + Reranking (with Actual Code)"
published: false
description: "Kenapa pure vector search nggak cukup, dan cara kombinasiin dengan BM25 dan reranker pakai Postgres + pgvector. Include SQL RRF snippet + Go orchestrator code + eval numbers dari production."
tags: [rag, postgres, pgvector, ai]
canonical_url:
cover_image:
---

Setahun terakhir, "cukup pakai vector search" jadi default advice buat RAG. Kenyataannya, ini nggak cukup untuk banyak use case production. Aku ketemu ini pas ngebangun chat app pribadi — vector search bagus untuk semantic recall, tapi jelek untuk exact-match queries kayak API key, product code, atau technical term yang jarang. Fix-nya: hybrid search — vector + BM25 + reranking.

Tulisan ini step-by-step gimana aku implement pattern itu pakai Postgres saja (pgvector + tsvector), plus code Go actual dari production stack, plus eval numbers yang aku dapet. Kalau kamu lagi ngebangun RAG dan pertimbangin pattern ini, semoga membantu.

## Kenapa pure vector search jatuh

Vector search kerja dengan cari dokumen yang embedding-nya paling deket cosine similarity ke query embedding. Bagus untuk:

- Sinonim: "cara login" match dokumen yang bilang "authentication flow"
- Konsep abstrak: "how to reduce latency" match diskusi caching, DB indexing, CDN.

Jelek untuk:

- Exact terms: query `Bearer YXXK7B` sering nggak return dokumen yang punya string itu, karena embedding "smooth out" token spesifik.
- Product codes, error codes, acronym: `ERR_INVALID_TOKEN` mungkin nggak dekat semantic dengan chunk yang sebutin string ini persis.
- Nama orang, tempat, technical terms yang tidak umum: model embedding underrepresent proper nouns.

Pattern klasik BM25 (best-match algorithm dari information retrieval era 90-an) unggul di area ini karena bekerja di level token frequency, bukan semantic. Untuk RAG production, kamu butuh dua-duanya.

## Setup: pgvector + tsvector di satu tabel

Aku pakai Neon Postgres. Satu tabel `document_chunks` menyimpan semuanya:

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE document_chunks (
    id          BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    chunk_index INT NOT NULL,
    content     TEXT NOT NULL,
    embedding   VECTOR(512),
    tsv         TSVECTOR,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- HNSW index untuk vector search (fast approximate nearest neighbor)
CREATE INDEX ON document_chunks
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- GIN index untuk fulltext search
CREATE INDEX ON document_chunks USING GIN (tsv);

-- Trigger auto-update tsvector dari content
CREATE TRIGGER document_chunks_tsv_update
BEFORE INSERT OR UPDATE ON document_chunks
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(tsv, 'pg_catalog.simple', content);
```

Kenapa `simple` config, bukan `english` atau `indonesian`? Karena user upload dokumen mixed language (bahasa Inggris dan Indonesia). `simple` config nggak lakukan stemming, jadi query exact match work untuk semua bahasa. Trade-off: recall lebih rendah dari `english` config untuk query bahasa Inggris, tapi jauh lebih konsisten.

## Vector search query

```sql
SELECT
    id,
    document_id,
    content,
    1 - (embedding <=> $1::vector) AS score
FROM document_chunks
WHERE user_id = $2
ORDER BY embedding <=> $1::vector
LIMIT 20;
```

`<=>` operator pgvector = cosine distance. `1 - distance` = cosine similarity. HNSW index bikin ini sub-100ms untuk table 100k rows.

## BM25 search query

Postgres nggak native BM25, tapi punya `ts_rank_cd` yang cukup approximate untuk use case ini. Aku pakai:

```sql
SELECT
    id,
    document_id,
    content,
    ts_rank_cd(tsv, plainto_tsquery('simple', $1)) AS score
FROM document_chunks
WHERE user_id = $2
    AND tsv @@ plainto_tsquery('simple', $1)
ORDER BY score DESC
LIMIT 20;
```

Kalau kamu butuh BM25 asli, ada extension `pg_search` atau bisa combine ts_rank dengan idf sendiri. Untuk portfolio scale, `ts_rank_cd` cukup.

## Fusion: Reciprocal Rank Fusion (RRF)

Vector search return top-20 dengan score cosine (0-1). BM25 return top-20 dengan score ts_rank (unbounded). Score ini nggak bisa langsung dijumlahkan — beda scale.

Solusi klasik: RRF. Rumusnya sederhana:

```
RRF_score(d) = Σ 1 / (k + rank_i(d))
```

Di mana `rank_i(d)` = rank dokumen `d` di ranking `i` (vector atau BM25), dan `k` konstanta smoothing (default 60).

Aku implement fusion di Go, bukan SQL, karena rank per dokumen lebih clear di app layer:

```go
// internal/service/retriever.go

type Candidate struct {
    ID         int64
    DocumentID int64
    Content    string
    VecScore   float64
    BM25Score  float64
    VecRank    int  // 0 = not present
    BM25Rank   int  // 0 = not present
    FusedScore float64
}

const rrfK = 60.0

func (r *Retriever) Fuse(vecHits, bm25Hits []Candidate) []Candidate {
    merged := make(map[int64]*Candidate)

    // Index vector hits dengan rank
    for i, h := range vecHits {
        h.VecRank = i + 1
        merged[h.ID] = &h
    }

    // Index bm25 hits, merge kalau sudah ada
    for i, h := range bm25Hits {
        if existing, ok := merged[h.ID]; ok {
            existing.BM25Score = h.BM25Score
            existing.BM25Rank = i + 1
        } else {
            h.BM25Rank = i + 1
            merged[h.ID] = &h
        }
    }

    // Compute RRF score untuk semua candidate
    out := make([]Candidate, 0, len(merged))
    for _, c := range merged {
        score := 0.0
        if c.VecRank > 0 {
            score += 1.0 / (rrfK + float64(c.VecRank))
        }
        if c.BM25Rank > 0 {
            score += 1.0 / (rrfK + float64(c.BM25Rank))
        }
        c.FusedScore = score
        out = append(out, *c)
    }

    sort.Slice(out, func(i, j int) bool {
        return out[i].FusedScore > out[j].FusedScore
    })
    return out
}
```

Setelah fusion, aku take top-40 candidates.

## Reranking: kenapa perlu, dan gimana

Fusion sudah bagus, tapi masih relatively noisy. Rerank stage — model khusus yang score `(query, passage)` pair — jauh lebih akurat dari embedding similarity karena bisa attention cross terms.

Voyage AI punya `rerank-2` API yang cocok. Aku panggil dengan top-40 candidates dari fusion, minta top-5 final:

```go
// internal/service/rerank.go

type RerankRequest struct {
    Query     string   `json:"query"`
    Documents []string `json:"documents"`
    Model     string   `json:"model"`
    TopK      int      `json:"top_k"`
}

type RerankResult struct {
    Index int     `json:"index"`
    Score float64 `json:"relevance_score"`
}

func (r *Reranker) Rerank(ctx context.Context, query string, candidates []Candidate, topK int) ([]Candidate, error) {
    if len(candidates) == 0 {
        return candidates, nil
    }
    if len(candidates) <= topK {
        return candidates, nil // No need to rerank
    }

    docs := make([]string, len(candidates))
    for i, c := range candidates {
        docs[i] = c.Content
    }

    body, _ := json.Marshal(RerankRequest{
        Query:     query,
        Documents: docs,
        Model:     "rerank-2",
        TopK:      topK,
    })

    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://api.voyageai.com/v1/rerank", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+r.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := r.client.Do(req)
    if err != nil {
        // Graceful degradation: kalau rerank gagal, return top-K by RRF score
        return candidates[:topK], nil
    }
    defer resp.Body.Close()

    var out struct {
        Data []RerankResult `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return candidates[:topK], nil
    }

    reranked := make([]Candidate, 0, len(out.Data))
    for _, r := range out.Data {
        c := candidates[r.Index]
        c.FusedScore = r.Score // Replace with rerank score
        reranked = append(reranked, c)
    }
    return reranked, nil
}
```

**Perhatikan graceful degradation.** Kalau Voyage down atau rate-limited, aku fall back ke top-K by fusion score, bukan return error. User dapat degraded quality, bukan broken chat.

## Orchestrator: putting it together

```go
// internal/service/retriever.go

func (r *Retriever) Retrieve(ctx context.Context, userID, query string) ([]Candidate, error) {
    // 1. Embed query
    embedding, err := r.embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    // 2. Parallel vector + BM25 search
    var (
        vecHits, bm25Hits []Candidate
        vecErr, bm25Err   error
        wg                sync.WaitGroup
    )
    wg.Add(2)
    go func() {
        defer wg.Done()
        vecHits, vecErr = r.vectorSearch(ctx, userID, embedding, 20)
    }()
    go func() {
        defer wg.Done()
        bm25Hits, bm25Err = r.bm25Search(ctx, userID, query, 20)
    }()
    wg.Wait()

    // Graceful degradation: kalau satu fail, pakai yang lain saja
    if vecErr != nil && bm25Err != nil {
        return nil, fmt.Errorf("both searches failed: vec=%v bm25=%v", vecErr, bm25Err)
    }

    // 3. Fuse
    fused := r.Fuse(vecHits, bm25Hits)

    // 4. Rerank top-40 to top-5
    top40 := fused
    if len(top40) > 40 {
        top40 = top40[:40]
    }
    reranked, err := r.reranker.Rerank(ctx, query, top40, 5)
    if err != nil {
        return top40[:min(5, len(top40))], nil // Fallback
    }

    return reranked, nil
}
```

Semua step diinstrument dengan tracer:

```go
retSpan := tr.Start("retrieval")
defer retSpan.Finish()
retSpan.SetMeta("query_len", len(query))
retSpan.SetMeta("vec_hits", len(vecHits))
retSpan.SetMeta("bm25_hits", len(bm25Hits))
retSpan.SetMeta("fused", len(fused))
retSpan.SetMeta("rerank_topk", 5)
```

## Eval: does it actually work?

Aku bikin eval sederhana dengan 50 query-answer pairs manual. Compute recall@5 dan MRR untuk 3 configuration:

| Config | Recall@5 | MRR | Notes |
|---|---|---|---|
| Pure vector search | 0.62 | 0.41 | Baseline |
| Vector + BM25 (RRF fusion) | 0.78 | 0.55 | +26% recall |
| Vector + BM25 + Voyage rerank | 0.90 | 0.71 | +45% recall dari baseline |

45% recall improvement dari baseline. Latency additional ~300ms untuk rerank stage (Voyage API call). Trade-off yang worth it untuk user experience.

Untuk query "singkat" (1-2 kata, exact match), improvement lebih dramatic. Untuk query "panjang semantic" (paraphrase konsep), improvement lebih kecil karena vector sudah bagus. Konsisten sama intuisi.

## Trade-off dan alternative

**Kenapa Postgres, bukan Pinecone / Weaviate / Qdrant?**

Untuk portfolio scale (10k-100k chunks), Postgres pgvector cukup dan biaya lebih murah. Kalau scale ke 10M+ chunks atau butuh dedicated ANN performance, dedicated vector DB better. Tapi migration path clear: interface `Retriever` bisa di-swap tanpa ubah caller.

**Kenapa Voyage rerank-2, bukan Cohere Rerank atau ColBERT?**

Voyage cheaper dan integration mudah. Cohere kualitas comparable tapi harga 2x. ColBERT self-hosted lebih murah untuk high volume tapi butuh GPU dan lebih complex. Untuk single-user portfolio, Voyage sweet spot.

**Kenapa RRF `k=60`?**

Standard value dari paper original. Sudah bagus untuk most use cases. Kalau tuning: lower `k` (misal 10) skew ke top-ranked items, higher `k` (misal 100) lebih egalitarian antara ranks. Aku test range 30-100 dan 60 memberikan hasil paling konsisten.

**Kenapa embed dimension 512, bukan 1536?**

Voyage voyage-3-lite output 512d dan quality comparable ke OpenAI text-embedding-3-small 1536d. Ukuran storage kecil, HNSW index build lebih cepat, search sub-linear tetap sub-100ms. Kalau butuh quality lebih tinggi, upgrade ke voyage-3-large (1024d).

## Bottom line

Pattern hybrid search + reranking bukan cutting-edge, tapi kombinasi tepat untuk RAG production yang jarang ditulis end-to-end. Postgres + pgvector + tsvector cukup untuk portfolio scale, dan pattern-nya scale sampai puluhan juta chunks kalau infrastructure di-upgrade.

Code lengkap ada di [repo Personal Chat AI](https://github.com/aulia/personal-chat-ai). Kalau kamu bangun RAG serupa, silakan copy pattern-nya. Kalau kamu punya feedback atau alternative approach yang aku bisa test, mention aku di comment atau DM di [LinkedIn](https://linkedin.com/in/aulia-afriza).

---

*Post lain di series: [How I built a personal AI assistant in 12 weeks](./01-building-personal-ai-assistant-12-weeks.md). Portfolio live di [personalchat.aulia.dev](https://personalchat.aulia.dev).*
