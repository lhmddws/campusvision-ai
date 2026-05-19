package dto

// ConfigUpdateDTO is the request body for updating a configuration entry.
type ConfigUpdateDTO struct {
	ConfigKey   string `json:"config_key" binding:"required"`
	ConfigValue string `json:"config_value" binding:"required"`
}
