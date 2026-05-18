package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cameras  []CameraConfig  `yaml:"cameras"`
	Frame    FrameConfig     `yaml:"frame"`
	Kafka    KafkaConfig     `yaml:"kafka"`
	RTSP     RTSPConfig      `yaml:"rtsp"`
	Health     HealthConfig     `yaml:"health"`
	Management ManagementConfig `yaml:"management"`
	Database   DatabaseConfig   `yaml:"database"`
	Log        LogConfig        `yaml:"log"`
}

type RTSPComponents struct {
	Protocol    string `yaml:"protocol" json:"protocol"`
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Path        string `yaml:"path" json:"path"`
	Username    string `yaml:"username" json:"username"`
	PasswordEnc string `yaml:"password_enc" json:"password_enc"`
	Nonce       string `yaml:"nonce" json:"nonce"`
	KeyID       string `yaml:"key_id" json:"key_id"`
}

type CameraConfig struct {
	ID         string         `yaml:"id"`
	Building   string         `yaml:"building"`
	Type       string         `yaml:"type"`
	RTSPURL    string         `yaml:"rtsp_url"`
	Components RTSPComponents `yaml:"components"`
	Enabled    bool           `yaml:"enabled"`
}

// BuildRTSPURL constructs the full RTSP URL from Components when password_enc is set,
// or falls back to RTSPURL for backward compatibility.
// masterKey is reserved for future decryption use (see Task 4).
func (c *CameraConfig) BuildRTSPURL(masterKey []byte) (string, error) {
	_ = masterKey // TODO: Use CryptoService to decrypt password_enc when CryptoService is available (Task 4)

	if c.Components.PasswordEnc != "" {
		password := c.Components.PasswordEnc

		// TODO: Decrypt password using CryptoService.Decrypt when available (Task 4)
		// For now, password_enc is passed through as-is.

		userinfo := url.UserPassword(c.Components.Username, password)
		u := &url.URL{
			Scheme: c.Components.Protocol,
			User:   userinfo,
			Host:   fmt.Sprintf("%s:%d", c.Components.Host, c.Components.Port),
			Path:   c.Components.Path,
		}
		return u.String(), nil
	}

	if c.RTSPURL != "" {
		return c.RTSPURL, nil
	}

	return "", fmt.Errorf("no RTSP URL configured for camera %s", c.ID)
}

type FrameConfig struct {
	FPSDay            int     `yaml:"fps_day"`
	FPSNight          int     `yaml:"fps_night"`
	JPEGQuality       int     `yaml:"jpeg_quality"`
	Width             int     `yaml:"width"`
	Height            int     `yaml:"height"`
	DynamicExtraction bool    `yaml:"dynamic_extraction"`
	MotionThreshold   float64 `yaml:"motion_threshold"`
}

type KafkaConfig struct {
	Brokers     []string `yaml:"brokers"`
	Topic       string   `yaml:"topic"`
	Compression string   `yaml:"compression"`
	BatchSize   int      `yaml:"batch_size"`
}

type RTSPConfig struct {
	ReconnectInterval    time.Duration `yaml:"reconnect_interval"`
	ReadTimeout          time.Duration `yaml:"read_timeout"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts"`
}

type HealthConfig struct {
	Port int `yaml:"port"`
}

type ManagementConfig struct {
	Port          int    `yaml:"port"`
	BindAddress   string `yaml:"bind_address"`
	ManagementKey string `yaml:"management_key"`
}

type DatabaseConfig struct {
	DSN          string        `yaml:"dsn"`
	Driver       string        `yaml:"driver"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.Management.Port == 0 {
		cfg.Management.Port = 8081
	}
	if cfg.Management.BindAddress == "" {
		cfg.Management.BindAddress = "127.0.0.1"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mysql"
	}
	if cfg.Database.PollInterval == 0 {
		cfg.Database.PollInterval = 30 * time.Second
	}

	return cfg, nil
}
