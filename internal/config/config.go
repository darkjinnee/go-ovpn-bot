package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken      string
	DatabasePath  string
	ScriptsPath   string
	ConfigsPath   string
	ConfigPrefix  string
	Debug         bool
}

func Load() (*Config, error) {
	// Загружаем .env файл если он существует
	if err := godotenv.Load(); err != nil {
		// Игнорируем ошибку если файл не найден
	}

	cfg := &Config{
		BotToken:     getEnv("BOT_TOKEN", ""),
		DatabasePath: getEnv("DATABASE_PATH", "./data/bot.db"),
		ScriptsPath:  getEnv("SCRIPTS_PATH", "./scripts"),
		ConfigsPath:  getEnv("CONFIGS_PATH", "./.ovpn"),
		ConfigPrefix: getEnv("CONFIG_PREFIX", "VPN"),
		Debug:        getBoolEnv("DEBUG", false),
	}

	if cfg.BotToken == "" {
		return nil, &ConfigError{Field: "BOT_TOKEN", Message: "Bot token is required"}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(strings.ToLower(value)); err == nil {
			return parsed
		}
	}
	return defaultValue
}


type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
