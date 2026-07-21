-- +goose Up
CREATE TABLE app_config (
    key text PRIMARY KEY,
    value jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO app_config (key, value) VALUES (
    'app',
    '{"webllm":{"modelId":"gemma-2b-q4f32_1-MLC","modelUrl":"https://huggingface.co/mlc-ai/Gemma-2B-q4f32_1-MLC/resolve/main/","estimatedBytes":1640000000},"features":{"scoring":true,"monitoring":true},"limits":{"maxPromptChars":4000,"ratePerMin":30}}'::jsonb
);

-- +goose Down
DROP TABLE IF EXISTS app_config;
