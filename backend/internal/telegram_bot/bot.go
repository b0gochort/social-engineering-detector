package telegram_bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"backend/internal/config"
	"backend/internal/repository"
)

// Bot represents the Telegram bot for access request notifications
type Bot struct {
	api              *tgbotapi.BotAPI
	logger           *zap.Logger
	accessRequestRepo repository.AccessRequestRepository
	messageRepo      repository.MessageRepository
	cfg              *config.Config
}

// NewBot creates a new Telegram bot instance
func NewBot(cfg *config.Config, accessRequestRepo repository.AccessRequestRepository, messageRepo repository.MessageRepository, logger *zap.Logger) (*Bot, error) {
	if !cfg.AccessControl.Enabled || cfg.AccessControl.TelegramBotToken == "" {
		logger.Info("Telegram bot is disabled (access_control.enabled=false or token is empty)")
		return nil, nil
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.AccessControl.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot API: %w", err)
	}

	logger.Info("Telegram bot authorized", zap.String("username", botAPI.Self.UserName))

	return &Bot{
		api:              botAPI,
		logger:           logger,
		accessRequestRepo: accessRequestRepo,
		messageRepo:      messageRepo,
		cfg:              cfg,
	}, nil
}

// Start begins listening for updates from Telegram
func (b *Bot) Start(ctx context.Context) error {
	if b == nil {
		return nil // Bot is disabled
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.logger.Info("Telegram bot started, waiting for updates...")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Telegram bot shutting down...")
			b.api.StopReceivingUpdates()
			return nil
		case update := <-updates:
			if update.CallbackQuery != nil {
				b.handleCallbackQuery(update.CallbackQuery)
			} else if update.Message != nil {
				b.handleMessage(update.Message)
			}
		}
	}
}

// handleCallbackQuery processes callback queries from inline buttons
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	b.logger.Info("Received callback query",
		zap.String("data", query.Data),
		zap.Int64("user_id", query.From.ID),
	)

	// Acknowledge the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.api.Request(callback); err != nil {
		b.logger.Error("Failed to send callback response", zap.Error(err))
	}

	// Parse callback data: "approve:<request_id>" or "reject:<request_id>"
	parts := strings.SplitN(query.Data, ":", 2)
	if len(parts) != 2 {
		b.logger.Error("Failed to parse callback data: invalid format", zap.String("data", query.Data))
		b.sendMessage(query.From.ID, "❌ Ошибка обработки запроса")
		return
	}
	action := parts[0]
	requestIDStr := parts[1]

	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		b.logger.Error("Failed to parse request ID", zap.String("id", requestIDStr), zap.Error(err))
		b.sendMessage(query.From.ID, "❌ Ошибка обработки запроса")
		return
	}

	// Get the access request
	accessRequest, err := b.accessRequestRepo.GetByID(requestID)
	if err != nil {
		b.logger.Error("Failed to get access request", zap.Int64("request_id", requestID), zap.Error(err))
		b.sendMessage(query.From.ID, "❌ Запрос не найден")
		return
	}

	// Check if already responded
	if accessRequest.Status != "pending" {
		b.sendMessage(query.From.ID, fmt.Sprintf("ℹ️ Запрос уже обработан (статус: %s)", accessRequest.Status))
		return
	}

	// Update the request status
	var newStatus string
	var responseMessage string

	switch action {
	case "approve":
		newStatus = "approved"
		responseMessage = "✅ Доступ предоставлен"
		// Update incident access_granted
		if err := b.messageRepo.UpdateIncidentAccessGranted(accessRequest.IncidentID, true, &requestID); err != nil {
			b.logger.Error("Failed to update incident access_granted", zap.Error(err))
			b.sendMessage(query.From.ID, "❌ Ошибка обновления инцидента")
			return
		}
	case "reject":
		newStatus = "rejected"
		responseMessage = "❌ Доступ отклонен"
	default:
		b.logger.Error("Unknown action", zap.String("action", action))
		b.sendMessage(query.From.ID, "❌ Неизвестное действие")
		return
	}

	// Update the access request
	if err := b.accessRequestRepo.UpdateStatus(requestID, newStatus, time.Now()); err != nil {
		b.logger.Error("Failed to update access request status", zap.Error(err))
		b.sendMessage(query.From.ID, "❌ Ошибка обновления статуса")
		return
	}

	b.logger.Info("Access request processed",
		zap.Int64("request_id", requestID),
		zap.String("action", action),
		zap.String("new_status", newStatus),
	)

	// Send confirmation message
	b.sendMessage(query.From.ID, responseMessage)

	// Edit the original message to remove buttons
	edit := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		query.Message.Text+"\n\n"+responseMessage,
	)
	if _, err := b.api.Send(edit); err != nil {
		b.logger.Error("Failed to edit message", zap.Error(err))
	}
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			b.handleStartCommand(message)
		case "help":
			b.handleHelpCommand(message)
		default:
			b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для помощи.")
		}
	}
}

// handleStartCommand handles the /start command
func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	welcomeText := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Я бот для управления доступом к сообщениям в системе мониторинга киберугроз.\n\n"+
			"Когда ваш родитель запросит доступ к инциденту, я отправлю вам уведомление с кнопками для одобрения или отклонения запроса.\n\n"+
			"Используйте /help для получения дополнительной информации.",
		message.From.FirstName,
	)
	b.sendMessage(message.Chat.ID, welcomeText)
}

// handleHelpCommand handles the /help command
func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	helpText := "📚 Помощь:\n\n" +
		"/start - Приветственное сообщение\n" +
		"/help - Эта справка\n\n" +
		"Когда родитель запрашивает доступ к сообщению инцидента, вы получите уведомление с двумя кнопками:\n" +
		"✅ Одобрить - предоставить доступ\n" +
		"❌ Отклонить - отказать в доступе\n\n" +
		"Ваш Telegram ID: " + strconv.FormatInt(message.From.ID, 10)
	b.sendMessage(message.Chat.ID, helpText)
}

// SendAccessRequestNotification sends a notification to the child about a new access request
func (b *Bot) SendAccessRequestNotification(childTelegramID int64, requestID int64, incidentID int64, threatType string, messageText string) error {
	if b == nil {
		return fmt.Errorf("bot is disabled")
	}

	b.logger.Info("Preparing notification",
		zap.Int64("incident_id", incidentID),
		zap.String("threat_type", threatType),
		zap.Int("message_length", len(messageText)),
		zap.String("message_text", messageText),
	)

	// Create a preview of the message (first 150 characters)
	messagePreview := messageText
	if len(messagePreview) > 150 {
		messagePreview = messagePreview[:150] + "..."
	}

	notificationText := fmt.Sprintf(
		"🔔 Новый запрос на доступ к сообщению\n\n"+
			"📋 Инцидент ID: %d\n"+
			"⚠️ Тип угрозы: %s\n\n"+
			"📝 Превью сообщения:\n%s\n\n"+
			"Ваш родитель запросил доступ к полному тексту сообщения. Вы хотите предоставить доступ?",
		incidentID,
		threatType,
		messagePreview,
	)

	// Create inline keyboard with Approve/Reject buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve:%d", requestID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject:%d", requestID)),
		),
	)

	msg := tgbotapi.NewMessage(childTelegramID, notificationText)
	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.Error("Failed to send access request notification",
			zap.Int64("child_telegram_id", childTelegramID),
			zap.Int64("request_id", requestID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	b.logger.Info("Access request notification sent",
		zap.Int64("child_telegram_id", childTelegramID),
		zap.Int64("request_id", requestID),
	)

	return nil
}

// sendMessage is a helper to send a simple text message
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}
