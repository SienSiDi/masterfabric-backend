package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
)

type ScoreRepository interface {
	Create(ctx context.Context, s *model.DecisionScore) error
	FindByEventID(ctx context.Context, eventID uuid.UUID) (*model.DecisionScore, error)
}

type MonitoringFilter struct {
	From     time.Time
	To       time.Time
	ModelID  string
}

type MonitoringTotals struct {
	Sessions     int `json:"sessions"`
	Events       int `json:"events"`
	ScoredEvents int `json:"scoredEvents"`
	Errors       int `json:"errors"`
}

type MonitoringLatency struct {
	P50Ms int `json:"p50Ms"`
	P95Ms int `json:"p95Ms"`
	MaxMs int `json:"maxMs"`
}

type MonitoringTokens struct {
	InTotal  int `json:"inTotal"`
	OutTotal int `json:"outTotal"`
}

type MonitoringScores struct {
	AvgCorrectness float64 `json:"avgCorrectness"`
	AvgComposite   float64 `json:"avgComposite"`
	SafetyFlagRate float64 `json:"safetyFlagRate"`
	UserAcceptRate float64 `json:"userAcceptRate"`
}

type MonitoringByModel struct {
	ModelID      string  `json:"modelId"`
	Events       int     `json:"events"`
	P50Ms        int     `json:"p50Ms"`
	AvgComposite float64 `json:"avgComposite"`
}

type MonitoringReport struct {
	Window  struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"window"`
	Totals   MonitoringTotals   `json:"totals"`
	Latency  MonitoringLatency  `json:"latency"`
	Tokens   MonitoringTokens   `json:"tokens"`
	Scores   MonitoringScores   `json:"scores"`
	ByModel  []MonitoringByModel `json:"byModel"`
}

type MonitoringRepository interface {
	GetReport(ctx context.Context, filter MonitoringFilter) (*MonitoringReport, error)
}
