package model

type ModelManifest struct {
	ID             string `json:"modelId"`
	EstimatedBytes int64  `json:"estimatedBytes"`
	Recommended    bool   `json:"recommended"`
}
