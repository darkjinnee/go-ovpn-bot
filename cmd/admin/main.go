package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go-ovpn-bot/internal/config"
	"go-ovpn-bot/internal/database"
)

func main() {
	var (
		limit = flag.Int("limit", 1, "Лимит конфигураций для кода")
		count = flag.Int("count", 1, "Количество кодов для генерации")
	)
	flag.Parse()

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

	// Генерируем коды
	rand.Seed(time.Now().UnixNano())
	
	fmt.Printf("Генерируем %d кодов с лимитом %d...\n\n", *count, *limit)
	
	for i := 0; i < *count; i++ {
		code := generateActivationCode()
		
		// Создаем код в базе данных
		activationCode, err := db.CreateActivationCode(code, *limit)
		if err != nil {
			log.Printf("Failed to create activation code %s: %v", code, err)
			continue
		}
		
		fmt.Printf("Код %d: %s (ID: %d, Лимит: %d)\n", 
			i+1, activationCode.Code, activationCode.ID, activationCode.Limit)
	}
	
	fmt.Printf("\n✅ Успешно создано %d кодов активации!\n", *count)
}

// generateActivationCode генерирует случайный код активации
func generateActivationCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 10)
	
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	
	return string(code)
}
