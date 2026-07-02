-- Minggu 11 — Observability + Evals.
--
-- 3 tabel:
--   chat_traces : satu row per chat request, spans inline sebagai JSONB
--   eval_sets   : golden query sets untuk retrieval eval
--   eval_runs   : hasil eval per set (recall@k, MRR, judge scores, dll)

CREATE TABLE chat_traces (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id   TEXT,  -- nullable: chat baru tanpa conversation persisted
    model             TEXT NOT NULL,
    total_duration_ms INTEGER NOT NULL,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    memory_count      INTEGER NOT NULL DEFAULT 0,
    sources_count     INTEGER NOT NULL DEFAULT 0,
    tool_calls_count  INTEGER NOT NULL DEFAULT 0,
    error             TEXT,  -- nullable
    -- spans: array of {stage, duration_ms, metadata}. Contoh:
    --   [{"stage":"memory_retrieve","duration_ms":45,"metadata":{"count":3}},
    --    {"stage":"rag_retrieve","duration_ms":210,"metadata":{"count":5}},
    --    {"stage":"llm_stream","duration_ms":3200,"metadata":{"iter":0}},
    --    {"stage":"tool_exec","duration_ms":180,"metadata":{"tool":"web_search"}}]
    spans             JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX traces_user_created_idx ON chat_traces (user_id, created_at DESC);

-- Golden query set untuk retrieval eval.
CREATE TABLE eval_sets (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- queries: array of {query, expected_document_ids[], notes?}
    queries     JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX eval_sets_user_idx ON eval_sets (user_id, created_at DESC);

-- Hasil eval run (retrieval atau judge).
CREATE TABLE eval_runs (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL,  -- 'retrieval' | 'judge'
    eval_set_id     TEXT REFERENCES eval_sets(id) ON DELETE SET NULL,  -- untuk retrieval kind
    subject_message_id TEXT,        -- untuk judge kind
    -- results: shape tergantung kind
    --   retrieval: {topK, avgRecallAtK, avgMRR, perQuery:[{query, expected, actual, recall, rr}]}
    --   judge:     {model, faithfulness, helpfulness, reasoning}
    results         JSONB NOT NULL DEFAULT '{}'::jsonb,
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX eval_runs_user_kind_idx ON eval_runs (user_id, kind, created_at DESC);
