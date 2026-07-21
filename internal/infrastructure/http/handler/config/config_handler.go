package config

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
	configusecase "github.com/masterfabric/masterfabric_backend/internal/application/config/usecase"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

type Handler struct {
	getUC    *configusecase.GetConfigUseCase
	updateUC *configusecase.UpdateConfigUseCase
}

func New(getUC *configusecase.GetConfigUseCase, updateUC *configusecase.UpdateConfigUseCase) *Handler {
	return &Handler{getUC: getUC, updateUC: updateUC}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.getUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req model.AppConfig
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "invalid JSON body", err))
		return
	}
	if req.Limits.MaxPromptChars <= 0 || req.Limits.RatePerMin <= 0 {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "limits.maxPromptChars and limits.ratePerMin must be positive", nil))
		return
	}
	cfg, err := h.updateUC.Execute(r.Context(), &req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}
