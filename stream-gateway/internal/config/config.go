package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cameras  []CameraConfig  `yaml:"cameras"`
	Frame    FrameConfig     `yaml:"frame"`
	Kafka    KafkaConfig     `yaml:"kafka"`
	RTSP     RTSPConfig      `yaml:"rtsp"`
	Health   HealthConfig    `yaml:"health"`
	Database DatabaseConfig  `yaml:"database"`
	Log      LogConfig       `yaml:"log"`
}

type CameraConfig struct {
	ID       string `yaml:"id"`
	Building string `yaml:"building"`
	RTSPURL  string `yaml:"rtsp_url"`
	Enabled  bool   `yaml:"enabled"`
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
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mysql"
	}
	if cfg.Database.PollInterval == 0 {
		cfg.Database.PollInterval = 30 * time.Second
	}

	return cfg, nil
}
