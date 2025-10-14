package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config структура для всей конфигурации приложения.
type Config struct {
	Telegram         TelegramConfig `yaml:"telegram"`
	Database         DatabaseConfig `yaml:"database"`
	CollectorInterval string         `yaml:"collector_interval"`
}

// DatabaseConfig contains configuration for the database connection.
type DatabaseConfig struct {
	URL string `yaml:"url"`
}

// TelegramConfig содержит конфигурацию для клиента Telegram.
type TelegramConfig struct {
	APIID   int    `yaml:"api_id"`
	APIHash string `yaml:"api_hash"`
	Phone   string `yaml:"phone"`
}

// LoadConfig читает конфигурацию из указанного пути.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}