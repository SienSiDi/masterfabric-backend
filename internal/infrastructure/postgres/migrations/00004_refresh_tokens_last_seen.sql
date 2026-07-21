-- +goose Up
ALTER TABLE refresh_tokens ADD COLUMN last_seen_at timestamptz NOT NULL DEFAULT now();
UPDATE refresh_tokens SET last_seen_at = created_at WHERE last_seen_at IS NULL;

-- +goose Down
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS last_seen_at;
