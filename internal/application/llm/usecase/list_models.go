package usecase

import (
	"context"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type ListModelsUseCase struct{}

func NewListModelsUseCase() *ListModelsUseCase { return &ListModelsUseCase{} }

// Execute returns the server-side allow-list of WebLLM models.
// For MVP this is a static list — future: load from app_config.
func (uc *ListModelsUseCase) Execute(_ context.Context) (dto.ListModelsResponse, error) {
	models := []llmmodel.ModelManifest{
		{ID: "gemma-2-2b-it-q4f32_1-MLC", EstimatedBytes: 2_508_000_000, Recommended: true},
		{ID: "gemma-2-2b-it-q4f16_1-MLC", EstimatedBytes: 2_700_000_000, Recommended: false},
	}
	out := make([]dto.ModelDTO, 0, len(models))
	for _, m := range models {
		out = append(out, dto.ModelDTO{
			ModelID:        m.ID,
			EstimatedBytes: m.EstimatedBytes,
			Recommended:    m.Recommended,
		})
	}
	return dto.ListModelsResponse{Models: out}, nil
}
