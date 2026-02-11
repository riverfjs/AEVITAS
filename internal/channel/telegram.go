package channel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdklogger "github.com/cexll/agentsdk-go/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	telegramify "github.com/riverfjs/telegramify-go"
	"github.com/stellarlinkco/myclaw/internal/bus"
	"github.com/stellarlinkco/myclaw/internal/config"
)

const telegramChannelName = "telegram"

// TelegramBot interface for mocking telegram bot API
type TelegramBot interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetSelf() tgbotapi.User
	GetFileDirectURL(fileID string) (string, error)
}

// tgBotWrapper wraps tgbotapi.BotAPI to implement TelegramBot interface
type tgBotWrapper struct {
	bot *tgbotapi.BotAPI
}

func (w *tgBotWrapper) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return w.bot.GetUpdatesChan(config)
}

func (w *tgBotWrapper) StopReceivingUpdates() {
	w.bot.StopReceivingUpdates()
}

func (w *tgBotWrapper) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return w.bot.Send(c)
}

func (w *tgBotWrapper) GetSelf() tgbotapi.User {
	return w.bot.Self
}

func (w *tgBotWrapper) GetFileDirectURL(fileID string) (string, error) {
	return w.bot.GetFileDirectURL(fileID)
}

// BotFactory creates TelegramBot instances (allows mocking)
type BotFactory func(token, apiEndpoint string, client *http.Client) (TelegramBot, error)

// defaultBotFactory creates real telegram bot
var defaultBotFactory BotFactory = func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPIWithClient(token, apiEndpoint, client)
	if err != nil {
		return nil, err
	}
	return &tgBotWrapper{bot: bot}, nil
}

type TelegramChannel struct {
	BaseChannel
	token      string
	bot        TelegramBot
	proxy      string
	cancel     context.CancelFunc
	botFactory BotFactory
}

func NewTelegramChannel(cfg config.TelegramConfig, b *bus.MessageBus, logger sdklogger.Logger) (*TelegramChannel, error) {
	return NewTelegramChannelWithFactory(cfg, b, defaultBotFactory, logger)
}

// NewTelegramChannelWithFactory creates a TelegramChannel with custom bot factory (for testing)
func NewTelegramChannelWithFactory(cfg config.TelegramConfig, b *bus.MessageBus, factory BotFactory, logger sdklogger.Logger) (*TelegramChannel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	ch := &TelegramChannel{
		BaseChannel: NewBaseChannel(telegramChannelName, b, cfg.AllowFrom, logger),
		token:       cfg.Token,
		proxy:       cfg.Proxy,
		botFactory:  factory,
	}
	return ch, nil
}

func (t *TelegramChannel) initBot() error {
	var client *http.Client
	if t.proxy != "" {
		proxyURL, err := url.Parse(t.proxy)
		if err != nil {
			return fmt.Errorf("parse proxy url: %w", err)
		}
		client = &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		}
	} else {
		client = http.DefaultClient
	}

	bot, err := t.botFactory(t.token, tgbotapi.APIEndpoint, client)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}
	t.bot = bot
	t.logger.Infof("[telegram] authorized as @%s", bot.GetSelf().UserName)
	return nil
}

func (t *TelegramChannel) Start(ctx context.Context) error {
	if err := t.initBot(); err != nil {
		return err
	}

	ctx, t.cancel = context.WithCancel(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := t.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case update := <-updates:
				if update.Message == nil {
					continue
				}
				t.handleMessage(update.Message)
			case <-ctx.Done():
				return
			}
		}
	}()

	t.logger.Infof("[telegram] polling started")
	return nil
}

func (t *TelegramChannel) handleMessage(msg *tgbotapi.Message) {
	senderID := strconv.FormatInt(msg.From.ID, 10)

	if !t.IsAllowed(senderID) {
		t.logger.Warnf("[telegram] rejected message from %s (%s)", senderID, msg.From.UserName)
		return
	}

	content := msg.Text
	if content == "" && msg.Caption != "" {
		content = msg.Caption
	}

	// Download photos if present
	var media []string
	if msg.Photo != nil && len(msg.Photo) > 0 {
		// Get largest photo
		photo := msg.Photo[len(msg.Photo)-1]
		localPath, err := t.downloadFile(photo.FileID, "photo")
		if err != nil {
			t.logger.Warnf("failed to download photo: %v", err)
		} else {
			media = append(media, localPath)
			t.logger.Debugf("downloaded photo to %s", localPath)
		}
		// If no text, provide default prompt
		if content == "" {
			content = "请分析这张图片"
		}
	}

	// Skip messages with no content and no media
	if content == "" && len(media) == 0 {
		return
	}

	chatID := strconv.FormatInt(msg.Chat.ID, 10)

	// Start continuous typing indicator (stops when message is received in Inbound channel)
	// Telegram typing indicator lasts 5 seconds, so we resend every 4 seconds
	stopTyping := make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		
		// Send first typing immediately
		typing := tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping)
		t.bot.Send(typing)
		
		for {
			select {
			case <-stopTyping:
				return
			case <-ticker.C:
				typing := tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping)
				t.bot.Send(typing)
			}
		}
	}()

	t.bus.Inbound <- bus.InboundMessage{
		Channel:   telegramChannelName,
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Media:     media,
		Timestamp: time.Unix(int64(msg.Date), 0),
		Metadata: map[string]any{
			"username":      msg.From.UserName,
			"first_name":    msg.From.FirstName,
			"message_id":    msg.MessageID,
			"stop_typing":   stopTyping, // Pass channel to gateway to stop typing
		},
	}
}

func (t *TelegramChannel) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.bot != nil {
		t.bot.StopReceivingUpdates()
	}
	return nil
}

// downloadFile downloads a Telegram file to local temp directory
func (t *TelegramChannel) downloadFile(fileID, prefix string) (string, error) {
	fileURL, err := t.bot.GetFileDirectURL(fileID)
	if err != nil {
		return "", fmt.Errorf("get file url: %w", err)
	}

	// Create temp directory
	tempDir := filepath.Join(os.TempDir(), "myclaw-telegram-media")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Download file
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	// Generate unique filename
	filename := fmt.Sprintf("%s-%d-%s", prefix, time.Now().Unix(), filepath.Base(fileURL))
	localPath := filepath.Join(tempDir, filename)

	// Save to disk
	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}

	return localPath, nil
}

// SetBot sets the bot (for testing)
func (t *TelegramChannel) SetBot(bot TelegramBot) {
	t.bot = bot
}

func (t *TelegramChannel) Send(msg bus.OutboundMessage) error {
	if t.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	chatID, err := strconv.ParseInt(msg.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}

	// Check if content contains image paths (detect screenshot paths)
	if imagePath := extractImagePath(msg.Content); imagePath != "" {
		// Send image first
		if err := t.sendPhoto(chatID, imagePath); err != nil {
			t.logger.Warnf("failed to send photo, falling back to text: %v", err)
			// Continue to send text message as fallback
		}
		// Remove image path from content
		msg.Content = strings.ReplaceAll(msg.Content, imagePath, "")
	}

	return t.sendNewMessage(chatID, msg.Content)
}

// extractImagePath extracts image file path from content
func extractImagePath(content string) string {
	// Match screenshot paths like /var/folders/.../screenshot-*.png
	re := regexp.MustCompile(`(/[^\s]+/screenshot-[0-9]+\.png)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// sendPhoto sends a photo to Telegram
func (t *TelegramChannel) sendPhoto(chatID int64, imagePath string) error {
	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file not found: %s", imagePath)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(imagePath))
	_, err := t.bot.Send(photo)
	if err != nil {
		return fmt.Errorf("send telegram photo: %w", err)
	}
	
	t.logger.Infof("sent photo to telegram chat_id=%d path=%s", chatID, imagePath)
	return nil
}

// sendNewMessage sends a new message (potentially split into multiple parts)
// Uses Telegram MessageEntity format instead of HTML/Markdown parse modes
func (t *TelegramChannel) sendNewMessage(chatID int64, content string) error {
	// Convert Markdown to plain text + entities using telegramify
	text, entities := telegramify.Convert(content, false, nil)
	
	// Split into chunks if needed (Telegram limit is 4096 UTF-16 code units)
	const maxUTF16Len = 4090 // Leave some margin
	chunks := telegramify.SplitEntities(text, entities, maxUTF16Len)
	
	// Send each chunk
	for _, chunk := range chunks {
		tgMsg := tgbotapi.NewMessage(chatID, chunk.Text)
		
		// Convert our MessageEntity to Telegram's format
		if len(chunk.Entities) > 0 {
			tgEntities := make([]tgbotapi.MessageEntity, 0, len(chunk.Entities))
			for _, ent := range chunk.Entities {
				tgEnt := tgbotapi.MessageEntity{
					Type:   ent.Type,
					Offset: ent.Offset,
					Length: ent.Length,
				}
				if ent.URL != "" {
					tgEnt.URL = ent.URL
				}
				if ent.Language != "" {
					tgEnt.Language = ent.Language
				}
				// Note: CustomEmojiID not supported by tgbotapi v5
				tgEntities = append(tgEntities, tgEnt)
			}
			tgMsg.Entities = tgEntities
		}
		
		// Send the message (do NOT set ParseMode - entities are used directly)
		if _, err := t.bot.Send(tgMsg); err != nil {
			// Fallback to plain text if entity parsing fails
			t.logger.Warnf("[telegram] Failed to send with entities, falling back to plain text: %v", err)
			fallbackMsg := tgbotapi.NewMessage(chatID, text) // Use original full text as fallback
			if _, err2 := t.bot.Send(fallbackMsg); err2 != nil {
				return fmt.Errorf("send telegram message: %w", err2)
			}
			return nil
		}
	}
	return nil
}

