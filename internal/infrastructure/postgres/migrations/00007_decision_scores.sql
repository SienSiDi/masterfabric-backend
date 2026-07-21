-- +goose Up
CREATE TABLE decision_scores (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id       uuid NOT NULL UNIQUE REFERENCES inference_events(id) ON DELETE CASCADE,
    correctness    double precision NOT NULL DEFAULT 0,
    latency_score  double precision NOT NULL DEFAULT 0,
    safety_flag    boolean NOT NULL DEFAULT false,
    cost_score     double precision NOT NULL DEFAULT 0,
    user_signal    text NOT NULL DEFAULT '',
    composite      double precision NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_decision_scores_event_id ON decision_scores(event_id);
CREATE INDEX idx_decision_scores_created_at ON decision_scores(created_at);

-- +goose Down
DROP TABLE IF EXISTS decision_scores;
