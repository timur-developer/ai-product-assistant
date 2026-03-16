CREATE TABLE drafts (
    id BIGSERIAL PRIMARY KEY,
    raw_idea TEXT NOT NULL,
    language TEXT NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0 CHECK (latest_version >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_drafts_created_at_id_desc
    ON drafts (created_at DESC, id DESC);
