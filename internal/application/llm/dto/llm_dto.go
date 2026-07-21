package dto

import (
	"time"

	"github.com/google/uuid"
)

// Models

type ModelDTO struct {
	ModelID        string `json:"modelId"`
	EstimatedBytes int64  `json:"estimatedBytes"`
	Recommended    bool   `json:"recommended"`
}

type ListModelsResponse struct {
	Models []ModelDTO `json:"models"`
}

// Sessions

type CreateSessionRequest struct {
	ModelID   string `json:"modelId" validate:"required"`
	ModelHash string `json:"modelHash"`
}

type SessionDTO struct {
	ID         uuid.UUID  `json:"sessionId"`
	ModelID    string     `json:"modelId"`
	ModelHash  string     `json:"modelHash"`
	CreatedAt  time.Time  `json:"createdAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
}

type ListSessionsByUserResponse struct {
	Sessions []SessionDTO `json:"sessions"`
}

// Events (Day 06)

type RecordEventRequest struct {
	Prompt     string `json:"prompt" validate:"required,max=4000"`
	Completion string `json:"completion"`
	TokensIn   int    `json:"tokensIn" validate:"min=0"`
	TokensOut  int    `json:"tokensOut" validate:"min=0"`
	LatencyMs  int    `json:"latencyMs" validate:"min=0"`
	Error      string `json:"error"`
}

type EventDTO struct {
	ID         uuid.UUID `json:"eventId"`
	Prompt     string    `json:"prompt"`
	Completion string    `json:"completion"`
	TokensIn   int       `json:"tokensIn"`
	TokensOut  int       `json:"tokensOut"`
	LatencyMs  int       `json:"latencyMs"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ListEventsResponse struct {
	Events []EventDTO `json:"events"`
	Page   int        `json:"page"`
	Limit  int        `json:"limit"`
	Total  int        `json:"total"`
}

// Score (Day 07)

type RecordScoreRequest struct {
	EventID       uuid.UUID `json:"eventId" validate:"required"`
	Correctness   float64   `json:"correctness" validate:"min=0,max=1"`
	LatencyScore  float64   `json:"latencyScore" validate:"min=0,max=1"`
	SafetyFlag    bool      `json:"safetyFlag"`
	CostScore     float64   `json:"costScore" validate:"min=0,max=1"`
	UserSignal    string    `json:"userSignal" validate:"omitempty,oneof=accept reject edit"`
	Composite     float64   `json:"composite" validate:"min=0,max=1"`
}

// Monitoring (Day 07) — mirrors the MonitoringReport struct from the domain.
type MonitoringResponse struct {
	Window  struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"window"`
	Totals  map[string]int     `json:"totals"`
	Latency map[string]int     `json:"latency"`
	Tokens  map[string]int     `json:"tokens"`
	Scores  map[string]float64 `json:"scores"`
	ByModel []map[string]any   `json:"byModel"`
}
