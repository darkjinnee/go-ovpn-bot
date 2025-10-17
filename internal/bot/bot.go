package bot

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go-ovpn-bot/internal/config"
	"go-ovpn-bot/internal/database"
	"go-ovpn-bot/internal/ovpn"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	config      *config.Config
	db          *database.DB
	ovpnService *ovpn.Service
	// Состояние ожидания кода активации для пользователей
	waitingForCode map[int64]bool
}

func New(cfg *config.Config, db *database.DB, ovpnService *ovpn.Service) *Bot {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Debug = cfg.Debug

	return &Bot{
		api:            bot,
		config:         cfg,
		db:             db,
		ovpnService:    ovpnService,
		waitingForCode: make(map[int64]bool),
	}
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	if b.config.Debug {
		log.Println("Bot started successfully in DEBUG mode")
	} else {
		log.Println("Bot started successfully")
	}

	for update := range updates {
		if b.config.Debug {
			log.Printf("Received update: %+v", update)
		}
		
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}

	return nil
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if b.config.Debug {
		log.Printf("Handling message from user %d (%s): %s", 
			message.From.ID, message.From.UserName, message.Text)
	}
	
	// Получаем или создаем пользователя
	user, err := b.db.GetOrCreateUser(
		int64(message.From.ID),
		message.From.UserName,
	)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Произошла ошибка при обработке запроса")
		return
	}

	// Проверяем, ожидает ли пользователь ввод кода активации
	if b.waitingForCode[user.ID] {
		b.handleActivationCode(message, user)
		return
	}

	// Обрабатываем команды
	switch {
	case strings.HasPrefix(message.Text, "/start"):
		b.handleStartCommand(message, user)
	case strings.HasPrefix(message.Text, "/add"):
		b.handleAddCommand(message, user)
	case strings.HasPrefix(message.Text, "/remove"):
		b.handleRemoveCommand(message, user)
	case strings.HasPrefix(message.Text, "/code"):
		b.handleCodeCommand(message, user)
	default:
		b.sendMessage(message.Chat.ID, "❓ Неизвестная команда. Используйте /start для просмотра доступных команд.")
	}
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Получаем пользователя
	user, err := b.db.GetOrCreateUser(
		int64(query.From.ID),
		query.From.UserName,
	)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		b.answerCallbackQuery(query.ID, "❌ Произошла ошибка")
		return
	}

	// Обрабатываем callback данные
	data := query.Data
	if strings.HasPrefix(data, "remove_") {
		configIDStr := strings.TrimPrefix(data, "remove_")
		configID, err := strconv.ParseInt(configIDStr, 10, 64)
		if err != nil {
			b.answerCallbackQuery(query.ID, "❌ Неверный ID конфигурации")
			return
		}

		b.handleRemoveConfigCallback(query, user, configID)
	} else if data == "cancel_remove" {
		b.handleCancelRemoveCallback(query)
	}

	// Отвечаем на callback query
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleStartCommand(message *tgbotapi.Message, user *database.User) {
	text := `🔐 *Добро пожаловать в OpenVPN Bot!*

Этот бот поможет вам управлять VPN конфигурациями.

*Доступные команды:*
• /add - Создать новую VPN конфигурацию
• /remove - Удалить существующую конфигурацию
• /code - Активировать код для увеличения лимита

*Ваши конфигурации:* ` + fmt.Sprintf("%d", len(user.Configs)) + `
*Ваш лимит:* ` + fmt.Sprintf("%d", user.Limit)

	b.sendMessage(message.Chat.ID, text)
}

func (b *Bot) handleAddCommand(message *tgbotapi.Message, user *database.User) {
	// Проверяем лимит пользователя
	if user.Limit <= len(user.Configs) {
		b.sendMessage(message.Chat.ID, 
			"❌ У вас исчерпан лимит конфигураций!\n\n"+
			"*Текущий лимит:* "+fmt.Sprintf("%d", user.Limit)+"\n"+
			"*Использовано:* "+fmt.Sprintf("%d", len(user.Configs))+"\n\n"+
			"Используйте команду /code для активации кода и увеличения лимита.")
		return
	}

	// Создаем клиента
	b.sendMessage(message.Chat.ID, "⏳ Создаю новую VPN конфигурацию...")

	clientName, configPath, err := b.ovpnService.CreateClient()
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка при создании конфигурации. Попробуйте позже.")
		return
	}

	// Сохраняем информацию о конфигурации в базу данных
	config, err := b.db.CreateConfig(user.ID, clientName, configPath)
	if err != nil {
		log.Printf("Failed to save config to database: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка при сохранении конфигурации в базу данных.")
		return
	}

	// Читаем содержимое конфигурационного файла
	configData, err := b.ovpnService.ReadConfigFile(configPath)
	if err != nil {
		log.Printf("Failed to read config file: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка при чтении конфигурационного файла.")
		return
	}

	// Отправляем конфигурационный файл
	file := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FileBytes{
		Name:  clientName + ".ovpn",
		Bytes: configData,
	})
	file.Caption = fmt.Sprintf("✅ Конфигурация *%s* успешно создана!", clientName)

	if _, err := b.api.Send(file); err != nil {
		log.Printf("Failed to send config file: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка при отправке конфигурационного файла.")
		return
	}

	// Обновляем информацию о пользователе
	user.Configs = append(user.Configs, *config)
}

func (b *Bot) handleRemoveCommand(message *tgbotapi.Message, user *database.User) {
	if len(user.Configs) == 0 {
		b.sendMessage(message.Chat.ID, "📭 У вас нет созданных конфигураций.")
		return
	}

	// Создаем inline клавиатуру с конфигурациями для удаления
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, config := range user.Configs {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🗑️ %s", config.Name),
			fmt.Sprintf("remove_%d", config.ID),
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	// Добавляем кнопку отмены
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_remove")
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{cancelButton})

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	text := "🗑️ *Выберите конфигурацию для удаления:*"
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = inlineKeyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send remove menu: %v", err)
	}
}

func (b *Bot) handleRemoveConfigCallback(query *tgbotapi.CallbackQuery, user *database.User, configID int64) {
	// Получаем информацию о конфигурации
	config, err := b.db.GetConfigByID(configID)
	if err != nil {
		log.Printf("Failed to get config: %v", err)
		b.answerCallbackQuery(query.ID, "❌ Конфигурация не найдена")
		return
	}

	// Проверяем что конфигурация принадлежит пользователю
	if config.UserID != user.ID {
		b.answerCallbackQuery(query.ID, "❌ У вас нет прав на удаление этой конфигурации")
		return
	}

	// Удаляем клиента через OpenVPN скрипт
	if err := b.ovpnService.RemoveClient(config.Name, config.FilePath); err != nil {
		log.Printf("Failed to remove client: %v", err)
		b.answerCallbackQuery(query.ID, "❌ Ошибка при удалении конфигурации")
		return
	}

	// Удаляем конфигурацию из базы данных
	if err := b.db.DeleteConfig(configID); err != nil {
		log.Printf("Failed to delete config from database: %v", err)
		b.answerCallbackQuery(query.ID, "❌ Ошибка при удалении из базы данных")
		return
	}

	// Отправляем подтверждение
	text := fmt.Sprintf("✅ Конфигурация *%s* успешно удалена!", config.Name)
	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send confirmation: %v", err)
	}

	b.answerCallbackQuery(query.ID, "✅ Конфигурация удалена")
}

func (b *Bot) handleCancelRemoveCallback(query *tgbotapi.CallbackQuery) {
	text := "❌ Удаление отменено."
	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send cancel message: %v", err)
	}

	b.answerCallbackQuery(query.ID, "❌ Отменено")
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func (b *Bot) answerCallbackQuery(callbackQueryID, text string) {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	if _, err := b.api.Request(callback); err != nil {
		log.Printf("Failed to answer callback query: %v", err)
	}
}

// handleCodeCommand обрабатывает команду /code
func (b *Bot) handleCodeCommand(message *tgbotapi.Message, user *database.User) {
	b.waitingForCode[user.ID] = true
	b.sendMessage(message.Chat.ID, 
		"🔑 *Активация кода*\n\n"+
		"Введите код активации для увеличения лимита конфигураций.\n\n"+
		"Код должен состоять из 10 символов (латинские буквы и цифры).")
}

// handleActivationCode обрабатывает введенный код активации
func (b *Bot) handleActivationCode(message *tgbotapi.Message, user *database.User) {
	code := strings.TrimSpace(message.Text)
	
	// Сбрасываем состояние ожидания
	delete(b.waitingForCode, user.ID)
	
	// Проверяем формат кода
	if len(code) != 10 {
		b.sendMessage(message.Chat.ID, 
			"❌ Неверный формат кода!\n\n"+
			"Код должен содержать ровно 10 символов (латинские буквы и цифры).")
		return
	}
	
	// Проверяем что код содержит только латинские буквы и цифры
	if !isValidCode(code) {
		b.sendMessage(message.Chat.ID, 
			"❌ Неверный формат кода!\n\n"+
			"Код должен содержать только латинские буквы (a-z, A-Z) и цифры (0-9).")
		return
	}
	
	// Получаем код из базы данных
	activationCode, err := b.db.GetActivationCodeByCode(code)
	if err != nil {
		b.sendMessage(message.Chat.ID, 
			"❌ Код не найден или неверный!\n\n"+
			"Проверьте правильность введенного кода.")
		return
	}
	
	// Проверяем статус кода
	if activationCode.Status != "active" {
		b.sendMessage(message.Chat.ID, 
			"❌ Код уже использован!\n\n"+
			"Этот код активации уже был использован ранее.")
		return
	}
	
	// Обновляем лимит пользователя
	newLimit := user.Limit + activationCode.Limit
	if err := b.db.UpdateUserLimit(user.ID, newLimit); err != nil {
		log.Printf("Failed to update user limit: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка при обновлении лимита. Попробуйте позже.")
		return
	}
	
	// Помечаем код как использованный
	if err := b.db.UseActivationCode(activationCode.ID); err != nil {
		log.Printf("Failed to mark code as used: %v", err)
		// Не прерываем выполнение, так как лимит уже обновлен
	}
	
	// Обновляем лимит в объекте пользователя
	user.Limit = newLimit
	
	b.sendMessage(message.Chat.ID, 
		fmt.Sprintf("✅ *Код успешно активирован!*\n\n"+
		"*Добавлено к лимиту:* %d\n"+
		"*Новый лимит:* %d\n"+
		"*Использовано:* %d\n\n"+
		"Теперь вы можете создавать VPN конфигурации!",
		activationCode.Limit, newLimit, len(user.Configs)))
}

// isValidCode проверяет что код содержит только латинские буквы и цифры
func isValidCode(code string) bool {
	for _, char := range code {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}

// generateActivationCode генерирует случайный код активации
func generateActivationCode() string {
	rand.Seed(time.Now().UnixNano())
	
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 10)
	
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	
	return string(code)
}
