package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application's configuration.
type Config struct {
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
	Collector struct {
		URL                string `yaml:"url"`
		PollInterval       int64  `yaml:"poll_interval_seconds"`
		ChatProcessDelay   int64  `yaml:"chat_process_delay_seconds"`
	} `yaml:"collector"`
	MLService struct {
		URL string `yaml:"url"`
	} `yaml:"ml_service"`
	AnnotationService struct {
		URL     string `yaml:"url"`
		Enabled bool   `yaml:"enabled"`
	} `yaml:"annotation_service"`
	AccessControl struct {
		Enabled          bool   `yaml:"enabled"`
		TelegramBotToken string `yaml:"telegram_bot_token"`
	} `yaml:"access_control"`
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
}

// LoadConfig reads configuration from the specified YAML file.
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}
