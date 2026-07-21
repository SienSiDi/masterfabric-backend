-- +goose Up
CREATE TABLE llm_sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id    text NOT NULL,
    model_hash  text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    ended_at    timestamptz
);

CREATE INDEX idx_llm_sessions_user_id ON llm_sessions(user_id);
CREATE INDEX idx_llm_sessions_created_at ON llm_sessions(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS llm_sessions;
