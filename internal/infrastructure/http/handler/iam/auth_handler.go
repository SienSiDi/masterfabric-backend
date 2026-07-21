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

type AuthHandler struct {
	registerUC    *iamusecase.RegisterUseCase
	loginUC       *iamusecase.LoginUseCase
	refreshUC     *iamusecase.RefreshUseCase
	logoutUC      *iamusecase.LogoutUseCase
	listSessUC    *iamusecase.ListSessionsUseCase
}

func NewAuthHandler(
	registerUC *iamusecase.RegisterUseCase,
	loginUC *iamusecase.LoginUseCase,
	refreshUC *iamusecase.RefreshUseCase,
	logoutUC *iamusecase.LogoutUseCase,
	listSessUC *iamusecase.ListSessionsUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		refreshUC:  refreshUC,
		logoutUC:   logoutUC,
		listSessUC: listSessUC,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.registerUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.loginUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.refreshUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.logoutUC.Execute(r.Context(), req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *AuthHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	userIDVal, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	resp, err := h.listSessUC.Execute(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}
