package usecase

import (
	"context"
	"testing"

	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

func TestListModelsUseCase_Execute(t *testing.T) {
	uc := NewListModelsUseCase()
	resp, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(resp.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(resp.Models))
	}
	// Verify the recommended Gemma is present
	found := false
	for _, m := range resp.Models {
		if m.ModelID == "gemma-2-2b-it-q4f32_1-MLC" && m.Recommended {
			found = true
		}
	}
	if !found {
		t.Error("expected gemma-2-2b-it-q4f32_1-MLC with recommended=true")
	}
	// keep dto referenced
	_ = dto.ListModelsResponse{}
}
