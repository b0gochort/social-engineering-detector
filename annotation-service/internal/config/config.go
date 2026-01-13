package config

import (
	"fmt"
	"os"

	"annotation-service/internal/llm"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	// Multiple providers configuration
	Providers []llm.ProviderConfig `yaml:"providers"`

	// Legacy single provider config (fallback)
	Gemini struct {
		APIKey     string `yaml:"api_key"`
		ModelName  string `yaml:"model_name"`
		MaxRetries int    `yaml:"max_retries"`
	} `yaml:"gemini"`

	Database struct {
		Path string `yaml:"path"` // SQLite path or PostgreSQL URL
		Type string `yaml:"type"` // "sqlite" or "postgres"
	} `yaml:"database"`

	MaxFailuresBeforeSwitch int `yaml:"max_failures_before_switch"`
}

// LoadConfig loads configuration from YAML file
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

	// Set defaults
	if config.Server.Port == "" {
		config.Server.Port = "8002"
	}

	if config.Gemini.ModelName == "" {
		config.Gemini.ModelName = "gemini-2.0-flash-exp"
	}

	if config.Gemini.MaxRetries == 0 {
		config.Gemini.MaxRetries = 3
	}

	if config.Database.Type == "" {
		config.Database.Type = "sqlite"
	}

	if config.Database.Path == "" {
		config.Database.Path = "./data/annotations.db"
	}

	if config.MaxFailuresBeforeSwitch == 0 {
		config.MaxFailuresBeforeSwitch = 3
	}

	// Expand environment variables in provider API keys
	for i := range config.Providers {
		config.Providers[i].APIKey = os.ExpandEnv(config.Providers[i].APIKey)
	}
	config.Gemini.APIKey = os.ExpandEnv(config.Gemini.APIKey)

	return config, nil
}
