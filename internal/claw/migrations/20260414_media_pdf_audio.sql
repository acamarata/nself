-- T-0701: np_claw_pdf_pages + np_claw_pdf_toc tables
-- T-0707: np_claw_transcripts table

-- Ensure ltree extension
CREATE EXTENSION IF NOT EXISTS ltree;

-- ==========================================================================
-- np_claw_pdf_pages
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_pdf_pages (
    attachment_id   UUID NOT NULL REFERENCES np_claw_attachments(id) ON DELETE CASCADE,
    page_number     INT NOT NULL,
    text            TEXT NOT NULL DEFAULT '',
    layout          JSONB NOT NULL DEFAULT '[]',
    tables          JSONB NOT NULL DEFAULT '[]',
    figure_ids      UUID[] DEFAULT '{}',
    embedding_id    UUID REFERENCES np_claw_media_embeddings(id) ON DELETE SET NULL,
    PRIMARY KEY (attachment_id, page_number)
);

CREATE INDEX IF NOT EXISTS idx_claw_pdf_pages_embedding
    ON np_claw_pdf_pages (embedding_id)
    WHERE embedding_id IS NOT NULL;

-- ==========================================================================
-- np_claw_pdf_toc
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_pdf_toc (
    attachment_id   UUID NOT NULL REFERENCES np_claw_attachments(id) ON DELETE CASCADE,
    path            LTREE NOT NULL,
    title           TEXT NOT NULL,
    page_number     INT NOT NULL,
    level           SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (attachment_id, path)
);

CREATE INDEX IF NOT EXISTS idx_claw_pdf_toc_path
    ON np_claw_pdf_toc USING gist (path);

-- ==========================================================================
-- np_claw_transcripts
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_transcripts (
    attachment_id   UUID PRIMARY KEY REFERENCES np_claw_attachments(id) ON DELETE CASCADE,
    language        TEXT NOT NULL DEFAULT 'en',
    provider        TEXT NOT NULL,
    full_text       TEXT NOT NULL DEFAULT '',
    words           JSONB NOT NULL DEFAULT '[]',
    segments        JSONB NOT NULL DEFAULT '[]',
    speakers        JSONB,
    confidence      REAL NOT NULL DEFAULT 0.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
