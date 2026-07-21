-- Update WebLLM model from gemma-2b-q4f32_1-MLC to gemma-2-2b-it-q4f32_1-MLC
-- (gemma-2b was removed from @mlc-ai/web-llm 0.2.84 prebuiltAppConfig)

UPDATE app_config
SET config = jsonb_set(
  config,
  '{webllm,modelId}',
  '"gemma-2-2b-it-q4f32_1-MLC"'
)
WHERE config->'webllm'->>'modelId' = 'gemma-2b-q4f32_1-MLC';

UPDATE app_config
SET config = jsonb_set(
  config,
  '{webllm,modelUrl}',
  '"https://huggingface.co/mlc-ai/gemma-2-2b-it-q4f32_1-MLC/resolve/main/"'
)
WHERE config->'webllm'->>'modelId' = 'gemma-2-2b-it-q4f32_1-MLC';

UPDATE app_config
SET config = jsonb_set(
  config,
  '{webllm,estimatedBytes}',
  '2508000000'
)
WHERE config->'webllm'->>'modelId' = 'gemma-2-2b-it-q4f32_1-MLC';
