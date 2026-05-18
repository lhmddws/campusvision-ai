package config

import (
	"encoding/json"
	"log"
)

// CameraConfigJSON represents the per-camera config_json content.
// V1: FPS override + type params (reserved for future use).
type CameraConfigJSON struct {
	FPSOverride *int                    `json:"fps_override,omitempty"`
	TypeParams  map[string]interface{}  `json:"type_params,omitempty"`
}

// ParseCameraConfig parses config_json string and returns effective FrameConfig.
// If config_json is empty or invalid, returns default with a warning.
// FPSOverride overrides both FPSDay and FPSNight when set.
func ParseCameraConfig(jsonStr string, defaultFrameCfg FrameConfig) FrameConfig {
	if jsonStr == "" {
		return defaultFrameCfg
	}

	var cfg CameraConfigJSON
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		log.Printf("[WARN] config: invalid config_json '%s': %v — using defaults", jsonStr, err)
		return defaultFrameCfg
	}

	result := defaultFrameCfg
	if cfg.FPSOverride != nil {
		result.FPSDay = *cfg.FPSOverride
		result.FPSNight = *cfg.FPSOverride
	}

	return result
}
