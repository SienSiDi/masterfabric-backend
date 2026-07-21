package validator

import (
	"encoding/json"
	"net/http"

	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/go-playground/validator/v10"
)

var v = validator.New()

func DecodeAndValidate(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return domainerr.New(domainerr.CodeBadRequest, "invalid JSON body", err)
	}
	if err := v.Struct(dst); err != nil {
		return domainerr.New(domainerr.CodeBadRequest, err.Error(), err)
	}
	return nil
}
