package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ llmrepo.MonitoringRepository = (*MonitoringRepository)(nil)

type MonitoringRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringRepository(pool *pgxpool.Pool) *MonitoringRepository {
	return &MonitoringRepository{pool: pool}
}

// GetReport runs the aggregation queries for the Raw LLM Monitoring payload.
// The time window is [from, to). Empty filter values mean "all time".
func (r *MonitoringRepository) GetReport(ctx context.Context, filter llmrepo.MonitoringFilter) (*llmrepo.MonitoringReport, error) {
	report := &llmrepo.MonitoringReport{}
	report.Window.From = filter.From
	report.Window.To = filter.To

	var fromArg, toArg any
	if !filter.From.IsZero() {
		fromArg = filter.From
	}
	if !filter.To.IsZero() {
		toArg = filter.To
	} else {
		toArg = time.Now().UTC()
		report.Window.To = toArg.(time.Time)
	}

	// 1. Totals + latency + tokens
	totalsQuery := `
		SELECT
			COUNT(DISTINCT e.session_id) AS sessions,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE s.id IS NOT NULL) AS scored_events,
			COUNT(*) FILTER (WHERE e.error <> '') AS errors,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY e.latency_ms), 0) AS p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY e.latency_ms), 0) AS p95,
			COALESCE(MAX(e.latency_ms), 0) AS max_latency,
			COALESCE(SUM(e.tokens_in), 0) AS tokens_in,
			COALESCE(SUM(e.tokens_out), 0) AS tokens_out
		FROM inference_events e
		JOIN llm_sessions sess ON sess.id = e.session_id
		LEFT JOIN decision_scores s ON s.event_id = e.id
		WHERE ($1::timestamptz IS NULL OR e.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR e.created_at < $2)
		  AND ($3::text = '' OR sess.model_id = $3)
	`
	err := r.pool.QueryRow(ctx, totalsQuery, fromArg, toArg, filter.ModelID).Scan(
		&report.Totals.Sessions,
		&report.Totals.Events,
		&report.Totals.ScoredEvents,
		&report.Totals.Errors,
		&report.Latency.P50Ms,
		&report.Latency.P95Ms,
		&report.Latency.MaxMs,
		&report.Tokens.InTotal,
		&report.Tokens.OutTotal,
	)
	if err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "monitoring totals", fmt.Errorf("totals query: %w", err))
	}

	// 2. Score aggregates
	scoresQuery := `
		SELECT
			COALESCE(AVG(s.correctness), 0),
			COALESCE(AVG(s.composite), 0),
			COALESCE(AVG(CASE WHEN s.safety_flag THEN 1.0 ELSE 0.0 END), 0),
			COALESCE(AVG(CASE WHEN s.user_signal = 'accept' THEN 1.0 ELSE 0.0 END), 0)
		FROM decision_scores s
		JOIN inference_events e ON e.id = s.event_id
		JOIN llm_sessions sess ON sess.id = e.session_id
		WHERE ($1::timestamptz IS NULL OR e.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR e.created_at < $2)
		  AND ($3::text = '' OR sess.model_id = $3)
	`
	err = r.pool.QueryRow(ctx, scoresQuery, fromArg, toArg, filter.ModelID).Scan(
		&report.Scores.AvgCorrectness,
		&report.Scores.AvgComposite,
		&report.Scores.SafetyFlagRate,
		&report.Scores.UserAcceptRate,
	)
	if err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "monitoring scores", fmt.Errorf("scores query: %w", err))
	}

	// 3. Per-model breakdown
	byModelQuery := `
		SELECT
			sess.model_id,
			COUNT(*) AS events,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY e.latency_ms), 0) AS p50,
			COALESCE(AVG(s.composite), 0) AS avg_composite
		FROM inference_events e
		JOIN llm_sessions sess ON sess.id = e.session_id
		LEFT JOIN decision_scores s ON s.event_id = e.id
		WHERE ($1::timestamptz IS NULL OR e.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR e.created_at < $2)
		GROUP BY sess.model_id
		ORDER BY events DESC
	`
	rows, err := r.pool.Query(ctx, byModelQuery, fromArg, toArg)
	if err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "monitoring by-model", fmt.Errorf("by-model query: %w", err))
	}
	defer rows.Close()
	for rows.Next() {
		var m llmrepo.MonitoringByModel
		if err := rows.Scan(&m.ModelID, &m.Events, &m.P50Ms, &m.AvgComposite); err != nil {
			return nil, domainerr.New(domainerr.CodeInternal, "scan by-model row", err)
		}
		report.ByModel = append(report.ByModel, m)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "iterate by-model", err)
	}
	if report.ByModel == nil {
		report.ByModel = []llmrepo.MonitoringByModel{}
	}

	return report, nil
}
