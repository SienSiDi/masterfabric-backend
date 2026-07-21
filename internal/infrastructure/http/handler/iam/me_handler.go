package iam

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	iamusecase "github.com/masterfabric/masterfabric_backend/internal/application/iam/usecase"
	"github.com/masterfabric/masterfabric_backend/internal/shared/middleware"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
	"github.com/masterfabric/masterfabric_backend/internal/shared/validator"
)

type MeHandler struct {
	meUC         *iamusecase.MeUseCase
	updateMeUC   *iamusecase.UpdateMeUseCase
	changePwdUC  *iamusecase.ChangePasswordUseCase
}

func NewMeHandler(meUC *iamusecase.MeUseCase, updateMeUC *iamusecase.UpdateMeUseCase, changePwdUC *iamusecase.ChangePasswordUseCase) *MeHandler {
	return &MeHandler{meUC: meUC, updateMeUC: updateMeUC, changePwdUC: changePwdUC}
}

func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	resp, err := h.meUC.Execute(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *MeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	var req dto.UpdateMeRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.updateMeUC.Execute(r.Context(), userID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *MeHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	var req dto.ChangePasswordRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.changePwdUC.Execute(r.Context(), userID, req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	uid, ok := v.(uuid.UUID)
	return uid, ok
}
