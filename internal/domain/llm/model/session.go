package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ModelID    string
	ModelHash  string
	CreatedAt  time.Time
	EndedAt    *time.Time
}

type InferenceEvent struct {
	ID          uuid.UUID
	SessionID   uuid.UUID
	UserID      uuid.UUID
	Prompt      string
	Completion  string
	TokensIn    int
	TokensOut   int
	LatencyMs   int
	Error       string
	CreatedAt   time.Time
}

type DecisionScore struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	Correctness   float64
	LatencyScore  float64
	SafetyFlag    bool
	CostScore     float64
	UserSignal    string
	Composite     float64
	CreatedAt     time.Time
}
