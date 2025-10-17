package main

import (
	"log"

	"go-ovpn-bot/internal/bot"
	"go-ovpn-bot/internal/config"
	"go-ovpn-bot/internal/database"
	"go-ovpn-bot/internal/ovpn"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем базу данных
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Инициализируем OpenVPN сервис
	ovpnService := ovpn.New(cfg.ScriptsPath, cfg.ConfigsPath, cfg.ConfigPrefix)

	// Создаем и запускаем бота
	botInstance := bot.New(cfg, db, ovpnService)

	if err := botInstance.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
}
