package message_processor

import (
	"context"
	"time"

	"go.uber.org/zap"

	"backend/internal/annotation_client"
	"backend/internal/collector_client"
	"backend/internal/crypto"
	"backend/internal/ml_client"
	"backend/internal/models"
	"backend/internal/repository"
)

// Processor handles fetching, processing, and saving messages.
type Processor struct {
	collectorClient  *collector_client.Client
	mlClient         *ml_client.Client
	annotationClient *annotation_client.Client
	messageRepo      repository.MessageRepository
	chatRepo         repository.ChatRepository
	mlDatasetRepo    repository.MLDatasetRepository
	keyManager       *crypto.KeyManager
	systemUserID     int64
	systemUserDKEnc  string
	logger           *zap.Logger
	pollInterval     int64
	chatProcessDelay int64
}

// NewProcessor creates a new message processor.
func NewProcessor(
	collectorClient *collector_client.Client,
	mlClient *ml_client.Client,
	annotationClient *annotation_client.Client,
	messageRepo repository.MessageRepository,
	chatRepo repository.ChatRepository,
	mlDatasetRepo repository.MLDatasetRepository,
	keyManager *crypto.KeyManager,
	systemUserID int64,
	systemUserDKEnc string,
	logger *zap.Logger,
	pollInterval int64,
	chatProcessDelay int64,
) *Processor {
	return &Processor{
		collectorClient:  collectorClient,
		mlClient:         mlClient,
		annotationClient: annotationClient,
		messageRepo:      messageRepo,
		chatRepo:         chatRepo,
		mlDatasetRepo:    mlDatasetRepo,
		keyManager:       keyManager,
		systemUserID:     systemUserID,
		systemUserDKEnc:  systemUserDKEnc,
		logger:           logger,
		pollInterval:     pollInterval,
		chatProcessDelay: chatProcessDelay,
	}
}

// Run starts the periodic message collection and processing.
func (p *Processor) Run(ctx context.Context) {
	p.logger.Info("Message processor started.")

	ticker := time.NewTicker(time.Duration(p.pollInterval) * time.Second)
	defer ticker.Stop()

	// Initial chat discovery on startup
	p.discoverAndManageChats(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Message processor stopped.")
			return
		case <-ticker.C:
			p.logger.Info("Polling collector for new messages...")

			// Periodically discover and manage chats
			p.discoverAndManageChats(ctx)

			chats, err := p.chatRepo.GetAllChats()
			if err != nil {
				p.logger.Error("Failed to get all chats from DB", zap.Error(err))
				continue
			}

			if len(chats) == 0 {
				p.logger.Info("No chats configured for monitoring.")
				continue
			}

			for i, chat := range chats {
				if !chat.MonitoringActive {
					p.logger.Debug("Skipping inactive chat", zap.Int64("chat_id", chat.ID), zap.String("source", chat.Source))
					continue
				}

				// Fetch messages based on source (Telegram or VK)
				collectorCtx, collectorCancel := context.WithTimeout(ctx, 15*time.Second)
				var messages []collector_client.Message
				var err error

				if chat.Source == "vk" && chat.VKPeerID != nil {
					p.logger.Info("Fetching VK messages for chat", zap.Int64("chat_id", chat.ID), zap.Int64("vk_peer_id", *chat.VKPeerID), zap.Int64("last_collected_message_id", chat.LastCollectedMessageID))
					messages, err = p.collectorClient.GetVKMessages(collectorCtx, *chat.VKPeerID, chat.LastCollectedMessageID)
				} else if chat.Source == "telegram" && chat.TelegramID != nil {
					p.logger.Info("Fetching Telegram messages for chat", zap.Int64("chat_id", chat.ID), zap.Int64("telegram_id", *chat.TelegramID), zap.Int64("last_collected_message_id", chat.LastCollectedMessageID))
					messages, err = p.collectorClient.GetMessages(collectorCtx, *chat.TelegramID, chat.LastCollectedMessageID)
				} else {
					p.logger.Warn("Chat has invalid source configuration", zap.Int64("chat_id", chat.ID), zap.String("source", chat.Source))
					collectorCancel()
					continue
				}

				collectorCancel()
				if err != nil {
					p.logger.Error("Failed to get messages from collector", zap.Error(err), zap.Int64("chat_id", chat.ID), zap.String("source", chat.Source))
					continue
				}

				if len(messages) == 0 {
					p.logger.Info("No new messages from collector for chat", zap.Int64("chat_id", chat.ID), zap.String("source", chat.Source))
					continue
				}

				p.logger.Info("Received messages from collector for chat", zap.Int64("chat_id", chat.ID), zap.String("source", chat.Source), zap.Int("count", len(messages)))

				var maxMessageID int64 = chat.LastCollectedMessageID
				for _, msg := range messages {
					// Encrypt message content with system user's data key
					encryptedContent, err := p.keyManager.EncryptMessage(msg.Text, p.systemUserID, p.systemUserDKEnc)
					if err != nil {
						p.logger.Error("Failed to encrypt message content", zap.Error(err), zap.Int64("message_id", msg.ID))
						continue
					}

					// Save the raw message with source-specific fields
					messageToSave := &models.Message{
						ChatID:           chat.ID,
						SenderUsername:   msg.SenderUsername,
						Timestamp:        msg.Timestamp,
						ContentEncrypted: encryptedContent,
						Source:           msg.Source,
						MessageType:      msg.Type,
					}

					// Set source-specific message IDs
					if msg.Source == "telegram" {
						messageToSave.TelegramMessageID = &msg.ID
					} else if msg.Source == "vk" {
						messageToSave.VKMessageID = &msg.ID
					}
					err = p.messageRepo.SaveMessage(messageToSave)
					if err != nil {
						p.logger.Error("Failed to save message", zap.Error(err), zap.Int64("telegram_message_id", msg.ID))
						continue
					}

					if msg.ID > maxMessageID {
						maxMessageID = msg.ID
					}

					// If annotation service is enabled, use it for dataset collection
					if p.annotationClient != nil {
						annotationCtx, annotationCancel := context.WithTimeout(ctx, 30*time.Second)
						annotation, err := p.annotationClient.AnnotateSingle(annotationCtx, msg.Text)
						annotationCancel()

						if err != nil {
							p.logger.Error("Failed to annotate message with Annotation Service", zap.Error(err), zap.Int64("message_id", msg.ID))
						} else {
							p.logger.Info("Message annotated",
								zap.Int64("message_id", msg.ID),
								zap.Int("category_id", annotation.CategoryID),
								zap.String("category_name", annotation.CategoryName))

							// Save ALL annotations to ML dataset (both threats and neutral)
							if p.mlDatasetRepo != nil {
								mlEntry := &models.MLDatasetEntry{
									MessageText:       msg.Text, // Plain text, NOT encrypted
									CategoryID:        annotation.CategoryID,
									CategoryName:      annotation.CategoryName,
									Justification:     annotation.Justification,
									Provider:          annotation.Provider,
									ModelVersion:      annotation.ModelVersion,
									AnnotatedAt:       annotation.AnnotatedAt,
									OriginalMessageID: &messageToSave.ID,
									IsValidated:       false,
									Source:            msg.Source, // Use actual source (telegram or vk)
								}
								err := p.mlDatasetRepo.SaveEntry(mlEntry)
								if err != nil {
									p.logger.Error("Failed to save ML dataset entry", zap.Error(err), zap.Int64("message_id", msg.ID))
								} else {
									p.logger.Debug("ML dataset entry saved",
										zap.Int64("dataset_id", mlEntry.ID),
										zap.Int("category_id", annotation.CategoryID))
								}
							}

							// Save as incident if it's a threat (category 1-8, not 9 which is neutral)
							if annotation.CategoryID != 9 {
								// Encrypt incident summary with system user's data key
								encryptedSummary, encErr := p.keyManager.EncryptMessage(msg.Text, p.systemUserID, p.systemUserDKEnc)
								if encErr != nil {
									p.logger.Error("Failed to encrypt incident summary", zap.Error(encErr), zap.Int64("message_id", msg.ID))
									encryptedSummary = "" // Use empty string if encryption fails
								}

								incidentToSave := &models.Incident{
									MessageID:        messageToSave.ID,
									ThreatType:       annotation.CategoryName,
									ModelConfidence:  1.0, // LLM annotation
									Status:           "new",
									SummaryEncrypted: encryptedSummary,
								}
								err := p.messageRepo.SaveIncident(incidentToSave)
								if err != nil {
									p.logger.Error("Failed to save annotated incident", zap.Error(err), zap.Int64("message_id", msg.ID))
								}
							}
						}
					} else {
						// Use ML service for production classification
						mlCtx, mlCancel := context.WithTimeout(ctx, 5*time.Second)
						classification, err := p.mlClient.ClassifySingle(mlCtx, msg.Text)
						mlCancel()

						if err != nil {
							p.logger.Error("Failed to classify message with ML service", zap.Error(err), zap.Int64("message_id", msg.ID))
							// Fallback to mock service if ML service is unavailable
							isSocialEngineering := p.mockAIService(msg.Text)
							if isSocialEngineering {
								classification = &ml_client.ClassifyResponse{
									PrimaryCategory:   "social_engineering",
									PrimaryCategoryID: 1,
									PrimaryConfidence: 0.5,
									IsAttack:          true,
								}
							}
						}

						if classification != nil && classification.IsAttack {
							// Use category from single model or fallback to primary category
							category := classification.Category
							categoryID := classification.CategoryID
							confidence := classification.Confidence

							// Fallback to legacy dual model fields if present
							if category == "" && classification.PrimaryCategory != "" {
								category = classification.PrimaryCategory
								categoryID = classification.PrimaryCategoryID
								confidence = classification.PrimaryConfidence
							}

							p.logger.Info("Social engineering message detected.",
								zap.Int64("message_id", msg.ID),
								zap.String("category", category),
								zap.Int("category_id", categoryID),
								zap.Float64("confidence", confidence))

							// Encrypt incident summary with system user's data key
							encryptedSummary, encErr := p.keyManager.EncryptMessage(msg.Text, p.systemUserID, p.systemUserDKEnc)
							if encErr != nil {
								p.logger.Error("Failed to encrypt incident summary", zap.Error(encErr), zap.Int64("message_id", msg.ID))
								encryptedSummary = "" // Use empty string if encryption fails
							}

							// Save the message as an incident
							incidentToSave := &models.Incident{
								MessageID:        messageToSave.ID,
								ThreatType:       category,
								ModelConfidence:  confidence,
								Status:           "new",
								SummaryEncrypted: encryptedSummary,
							}
							err := p.messageRepo.SaveIncident(incidentToSave)
							if err != nil {
								p.logger.Error("Failed to save social engineering incident", zap.Error(err), zap.Int64("message_id", msg.ID))
							}
						}
					}
				}

				// Update LastCollectedMessageID for the chat
				if maxMessageID > chat.LastCollectedMessageID {
					err := p.chatRepo.UpdateLastCollectedMessageID(chat.ID, maxMessageID)
					if err != nil {
						p.logger.Error("Failed to update last collected message ID for chat", zap.Error(err), zap.Int64("chat_id", chat.ID), zap.Int64("new_max_message_id", maxMessageID))
					}
				}

				// Add a delay after processing each chat to avoid FLOOD_WAIT errors
				if i < len(chats)-1 && p.chatProcessDelay > 0 {
					p.logger.Debug("Waiting before processing next chat", zap.Int64("delay_seconds", p.chatProcessDelay))
					time.Sleep(time.Duration(p.chatProcessDelay) * time.Second)
				}
			}
		}
	}
}

func (p *Processor) discoverAndManageChats(ctx context.Context) {
	p.logger.Info("Discovering and managing chats...")
	collectorCtx, collectorCancel := context.WithTimeout(ctx, 10*time.Second)
	defer collectorCancel()

	collectorChats, err := p.collectorClient.GetChats(collectorCtx)
	if err != nil {
		p.logger.Error("Failed to get chats from collector for discovery", zap.Error(err))
		return
	}

	if len(collectorChats) == 0 {
		p.logger.Info("No chats found by collector.")
		return
	}

	for _, cChat := range collectorChats {
		dbChat, err := p.chatRepo.GetChatByTelegramID(cChat.ID)
		if err != nil {
			p.logger.Error("Failed to check chat existence in DB", zap.Error(err), zap.Int64("telegram_id", cChat.ID))
			continue
		}

		if dbChat == nil {
			// Chat does not exist in DB, create it
			p.logger.Info("New chat discovered, adding to DB", zap.Int64("telegram_id", cChat.ID), zap.String("name", cChat.Name))
			telegramID := cChat.ID
			newChat := &models.Chat{
				TelegramID:             &telegramID,
				Source:                 "telegram",
				Name:                   cChat.Name,
				IsGroup:                cChat.IsGroup,
				MonitoringActive:       true, // Default to active monitoring for new chats
				LastCollectedMessageID: 0,
			}
			err = p.chatRepo.CreateChat(newChat)
			if err != nil {
				p.logger.Error("Failed to create new chat in DB", zap.Error(err), zap.Int64("telegram_id", cChat.ID))
			}
		} else {
			// Chat already exists, ensure its name is up-to-date (optional, but good practice)
			if dbChat.Name != cChat.Name || dbChat.IsGroup != cChat.IsGroup {
				p.logger.Info("Updating existing chat info",
					zap.Int64("telegram_id", cChat.ID), zap.String("old_name", dbChat.Name), zap.String("new_name", cChat.Name))
				// TODO: Implement UpdateChat method in ChatRepository if needed
			}
		}
	}
}

// mockAIService is a placeholder for the AI service.
// It simulates social engineering detection.
func (p *Processor) mockAIService(text string) bool {
	// Simple heuristic for demonstration: if text contains "urgent" or "click here"
	if contains(text, "urgent") || contains(text, "click here") {
		return true
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}
