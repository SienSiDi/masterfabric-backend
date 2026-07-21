-- +goose Up
CREATE TABLE inference_events (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   uuid NOT NULL REFERENCES llm_sessions(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prompt       text NOT NULL,
    completion   text NOT NULL DEFAULT '',
    tokens_in    integer NOT NULL DEFAULT 0,
    tokens_out   integer NOT NULL DEFAULT 0,
    latency_ms   integer NOT NULL DEFAULT 0,
    error        text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_inference_events_session_id ON inference_events(session_id, created_at);
CREATE INDEX idx_inference_events_user_id ON inference_events(user_id);
CREATE INDEX idx_inference_events_created_at ON inference_events(created_at);

-- +goose Down
DROP TABLE IF EXISTS inference_events;
