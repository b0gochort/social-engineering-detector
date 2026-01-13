package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config структура для всей конфигурации приложения.
type Config struct {
	Telegram         TelegramConfig `yaml:"telegram"`
	VK               VKConfig       `yaml:"vk"`
	Database         DatabaseConfig `yaml:"database"`
	API              APIConfig      `yaml:"api"`
	CollectorInterval string         `yaml:"collector_interval"`
}

// APIConfig contains configuration for the API server.
type APIConfig struct {
	Port string `yaml:"port"`
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

// VKConfig содержит конфигурацию для клиента VK API.
type VKConfig struct {
	Enabled      bool   `yaml:"enabled"`
	AccessToken  string `yaml:"access_token"`  // User Access Token (получается через OAuth)
	AppID        int    `yaml:"app_id"`        // VK Application ID
	ClientSecret string `yaml:"client_secret"` // VK Application Secret
	RedirectURI  string `yaml:"redirect_uri"`  // OAuth redirect URI
}

// LoadConfig читает конфигурацию из указанного пути.
// Environment переменные имеют приоритет над значениями из файла.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Override with environment variables if they exist
	// Telegram configuration
	if apiID := os.Getenv("TELEGRAM_API_ID"); apiID != "" {
		if id, err := strconv.Atoi(apiID); err == nil {
			cfg.Telegram.APIID = id
		}
	}
	if apiHash := os.Getenv("TELEGRAM_API_HASH"); apiHash != "" {
		cfg.Telegram.APIHash = apiHash
	}
	if phone := os.Getenv("TELEGRAM_PHONE"); phone != "" {
		cfg.Telegram.Phone = phone
	}

	// VK configuration
	if appID := os.Getenv("VK_APP_ID"); appID != "" {
		if id, err := strconv.Atoi(appID); err == nil {
			cfg.VK.AppID = id
		}
	}
	if accessToken := os.Getenv("VK_ACCESS_TOKEN"); accessToken != "" {
		cfg.VK.AccessToken = accessToken
	}
	if enabled := os.Getenv("VK_ENABLED"); enabled != "" {
		cfg.VK.Enabled = enabled == "true" || enabled == "1"
	}

	return &cfg, nil
}