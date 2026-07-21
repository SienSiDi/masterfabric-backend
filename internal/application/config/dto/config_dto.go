package dto

import "github.com/masterfabric/masterfabric_backend/internal/domain/config/model"

// GetConfigResponse is the JSON shape returned by GET /api/v1/config.
type GetConfigResponse = model.AppConfig

// UpdateConfigRequest is the JSON body for PUT /api/v1/admin/config (full replace).
type UpdateConfigRequest = model.AppConfig
