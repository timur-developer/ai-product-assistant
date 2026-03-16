CREATE TABLE draft_versions (
    id BIGSERIAL PRIMARY KEY,
    draft_id BIGINT NOT NULL REFERENCES drafts (id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version > 0),
    content JSONB NOT NULL,
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL CHECK (prompt_tokens >= 0),
    completion_tokens INTEGER NOT NULL CHECK (completion_tokens >= 0),
    total_tokens INTEGER NOT NULL CHECK (total_tokens >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (draft_id, version)
);

CREATE INDEX idx_draft_versions_draft_id_version_desc
    ON draft_versions (draft_id, version DESC);
