CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS posts (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    title text NOT NULL,
    content text NOT NULL,
    author_id varchar(255) NOT NULL,
    comments_enabled boolean NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS comments (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    post_id uuid NOT NULL,
    parent_id uuid,
    content text NOT NULL CHECK (length(content) <= 2000),
    author_id varchar(255) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_post FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES comments (id) ON DELETE CASCADE
);

-- Индексы для получения комментов и детей коммента
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments (post_id);

CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments (parent_id);

