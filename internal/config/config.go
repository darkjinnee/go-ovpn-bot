package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken      string
	DatabasePath  string
	ScriptsPath   string
	ConfigsPath   string
	MaxConfigs    int
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
		MaxConfigs:   getEnvAsInt("MAX_CONFIGS", 5),
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

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
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
