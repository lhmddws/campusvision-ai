package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// ConfigHandler handles HTTP requests for /api/configs.
type ConfigHandler struct {
	svc *service.ConfigService
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// ListConfigs    GET /api/configs
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	group := c.Query("group")

	configs, err := h.svc.GetAllConfigs(c.Request.Context(), group)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to list configs: "+err.Error())
		return
	}

	Success(c, configs)
}

// GetConfig    GET /api/configs/:key
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	key := c.Param("key")

	cfg, err := h.svc.GetConfigByKey(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Config not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to get config: "+err.Error())
		return
	}

	Success(c, cfg)
}

// UpdateConfig    PUT /api/configs/:key
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")

	var body struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body, expected {\"value\": \"...\"}")
		return
	}

	if err := h.svc.UpdateConfig(c.Request.Context(), key, body.Value); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Config not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to update config: "+err.Error())
		return
	}

	Success(c, nil)
}

// BatchUpdate    PUT /api/configs/batch
func (h *ConfigHandler) BatchUpdate(c *gin.Context) {
	var body []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body, expected array of {key, value}")
		return
	}

	if len(body) == 0 {
		Error(c, http.StatusBadRequest, "Request body must be a non-empty array")
		return
	}

	updates := make([]dto.ConfigUpdateDTO, 0, len(body))
	for _, item := range body {
		updates = append(updates, dto.ConfigUpdateDTO{
			ConfigKey:   item.Key,
			ConfigValue: item.Value,
		})
	}

	if err := h.svc.BatchUpdate(c.Request.Context(), updates); err != nil {
		Error(c, http.StatusInternalServerError, "Failed to batch update configs: "+err.Error())
		return
	}

	Success(c, nil)
}

// ResetConfig    POST /api/configs/:key/reset
func (h *ConfigHandler) ResetConfig(c *gin.Context) {
	key := c.Param("key")

	cfg, err := h.svc.ResetConfig(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Config not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to reset config: "+err.Error())
		return
	}

	Success(c, cfg)
}

// ListGroups    GET /api/configs/groups
func (h *ConfigHandler) ListGroups(c *gin.Context) {
	groups, err := h.svc.GetGroups(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to list groups: "+err.Error())
		return
	}

	Success(c, groups)
}
