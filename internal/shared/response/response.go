package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Created(w http.ResponseWriter, body any) { JSON(w, http.StatusCreated, body) }
func NoContent(w http.ResponseWriter)        { w.WriteHeader(http.StatusNoContent) }

func Error(w http.ResponseWriter, err error) {
	de, ok := domainerr.As(err)
	if !ok {
		slog.Error("internal error", "error", err)
		JSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{"code": string(domainerr.CodeInternal), "message": "an internal error occurred"},
		})
		return
	}
	status := mapStatus(de.Code)
	if status >= 500 {
		slog.Error("server error", "code", de.Code, "message", de.Message, "cause", de.Cause)
	} else {
		slog.Debug("client error", "code", de.Code, "message", de.Message)
	}
	JSON(w, status, map[string]any{
		"error": map[string]any{"code": string(de.Code), "message": de.Message},
	})
}

func mapStatus(c domainerr.Code) int {
	switch c {
	case domainerr.CodeNotFound:
		return http.StatusNotFound
	case domainerr.CodeBadRequest:
		return http.StatusBadRequest
	case domainerr.CodeUnauthorized:
		return http.StatusUnauthorized
	case domainerr.CodeForbidden:
		return http.StatusForbidden
	case domainerr.CodeConflict:
		return http.StatusConflict
	case domainerr.CodeTooMany:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
