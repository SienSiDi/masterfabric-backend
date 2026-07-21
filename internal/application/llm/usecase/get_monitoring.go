package usecase

import (
	"context"
	"time"

	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type GetMonitoringUseCase struct {
	monitoringRepo llmrepo.MonitoringRepository
}

func NewGetMonitoringUseCase(monitoringRepo llmrepo.MonitoringRepository) *GetMonitoringUseCase {
	return &GetMonitoringUseCase{monitoringRepo: monitoringRepo}
}

// Execute returns the aggregated monitoring report. Admin-only at the route level.
func (uc *GetMonitoringUseCase) Execute(ctx context.Context, from, to time.Time, modelID string) (*llmrepo.MonitoringReport, error) {
	filter := llmrepo.MonitoringFilter{
		From:    from,
		To:      to,
		ModelID: modelID,
	}
	return uc.monitoringRepo.GetReport(ctx, filter)
}

// ToDTO converts the domain MonitoringReport into the DTO shape that matches the
// spec api_endpoints.md. Used by the handler.
func ToDTO(r *llmrepo.MonitoringReport) dto.MonitoringResponse {
	resp := dto.MonitoringResponse{
		Totals: map[string]int{
			"sessions":      r.Totals.Sessions,
			"events":        r.Totals.Events,
			"scoredEvents":  r.Totals.ScoredEvents,
			"errors":        r.Totals.Errors,
		},
		Latency: map[string]int{
			"p50Ms": r.Latency.P50Ms,
			"p95Ms": r.Latency.P95Ms,
			"maxMs": r.Latency.MaxMs,
		},
		Tokens: map[string]int{
			"inTotal":  r.Tokens.InTotal,
			"outTotal": r.Tokens.OutTotal,
		},
		Scores: map[string]float64{
			"avgCorrectness": r.Scores.AvgCorrectness,
			"avgComposite":   r.Scores.AvgComposite,
			"safetyFlagRate": r.Scores.SafetyFlagRate,
			"userAcceptRate": r.Scores.UserAcceptRate,
		},
	}
	resp.Window.From = r.Window.From
	resp.Window.To = r.Window.To
	for _, m := range r.ByModel {
		resp.ByModel = append(resp.ByModel, map[string]any{
			"modelId":      m.ModelID,
			"events":       m.Events,
			"p50Ms":        m.P50Ms,
			"avgComposite": m.AvgComposite,
		})
	}
	if resp.ByModel == nil {
		resp.ByModel = []map[string]any{}
	}
	return resp
}
