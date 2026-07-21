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
	b, err := json.Marshal(body)
	if err != nil {
		return
	}
	_, _ = w.Write(b)
}

func Created(w http.ResponseWriter, body any) { JSON(w, http.StatusCreated, body) }
func NoContent(w http.ResponseWriter)        { w.WriteHeader(http.StatusNoContent) }

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func Error(w http.ResponseWriter, err error) {
	de, ok := domainerr.As(err)
	if !ok {
		slog.Error("internal error", "error", err)
		eb := errorBody{}
		eb.Error.Code = string(domainerr.CodeInternal)
		eb.Error.Message = "an internal error occurred"
		JSON(w, http.StatusInternalServerError, eb)
		return
	}
	status := mapStatus(de.Code)
	if status >= 500 {
		slog.Error("server error", "code", de.Code, "message", de.Message, "cause", de.Cause)
	} else {
		slog.Debug("client error", "code", de.Code, "message", de.Message)
	}
	eb := errorBody{}
	eb.Error.Code = string(de.Code)
	eb.Error.Message = de.Message
	JSON(w, status, eb)
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
