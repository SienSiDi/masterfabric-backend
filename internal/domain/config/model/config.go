package model

type AppConfig struct {
	WebLLM   WebLLMConfig `json:"webllm"`
	Features Features     `json:"features"`
	Limits   Limits       `json:"limits"`
}

type WebLLMConfig struct {
	ModelID        string `json:"modelId"`
	ModelURL       string `json:"modelUrl"`
	EstimatedBytes int64  `json:"estimatedBytes"`
}

type Features struct {
	Scoring    bool `json:"scoring"`
	Monitoring bool `json:"monitoring"`
}

type Limits struct {
	MaxPromptChars int `json:"maxPromptChars"`
	RatePerMin     int `json:"ratePerMin"`
}

func Default() AppConfig {
	return AppConfig{
		WebLLM: WebLLMConfig{
			ModelID:        "gemma-2b-q4f32_1-MLC",
			ModelURL:       "https://huggingface.co/mlc-ai/Gemma-2B-q4f32_1-MLC/resolve/main/",
			EstimatedBytes: 1_640_000_000,
		},
		Features: Features{Scoring: true, Monitoring: true},
		Limits:   Limits{MaxPromptChars: 4000, RatePerMin: 30},
	}
}
