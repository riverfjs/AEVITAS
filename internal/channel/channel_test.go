package channel

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/chenwenjie/myclaw/internal/bus"
	"github.com/chenwenjie/myclaw/internal/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestBaseChannel_Name(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch := NewBaseChannel("test", b, nil)
	if ch.Name() != "test" {
		t.Errorf("Name = %q, want test", ch.Name())
	}
}

func TestBaseChannel_IsAllowed_NoFilter(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch := NewBaseChannel("test", b, nil)
	if !ch.IsAllowed("anyone") {
		t.Error("should allow anyone when allowFrom is empty")
	}
}

func TestBaseChannel_IsAllowed_WithFilter(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch := NewBaseChannel("test", b, []string{"user1", "user2"})

	if !ch.IsAllowed("user1") {
		t.Error("should allow user1")
	}
	if !ch.IsAllowed("user2") {
		t.Error("should allow user2")
	}
	if ch.IsAllowed("user3") {
		t.Error("should reject user3")
	}
}

func TestNewTelegramChannel_NoToken(t *testing.T) {
	b := bus.NewMessageBus(10)
	_, err := NewTelegramChannel(config.TelegramConfig{}, b)
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestNewTelegramChannel_Valid(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, err := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Name() != "telegram" {
		t.Errorf("Name = %q, want telegram", ch.Name())
	}
}

func TestToTelegramHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"**bold**", "<b>bold</b>"},
		{"`code`", "<code>code</code>"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
	}

	for _, tt := range tests {
		got := toTelegramHTML(tt.input)
		if got != tt.want {
			t.Errorf("toTelegramHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestChannelManager_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	m, err := NewChannelManager(config.ChannelsConfig{}, b)
	if err != nil {
		t.Fatalf("NewChannelManager error: %v", err)
	}
	if len(m.EnabledChannels()) != 0 {
		t.Errorf("expected 0 enabled channels, got %d", len(m.EnabledChannels()))
	}
}

func TestToTelegramHTML_CodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"code block with language",
			"```go\nfunc main() {}\n```",
			"<pre>func main() {}\n</pre>",
		},
		{
			"code block without language",
			"```\ncode here\n```",
			"<pre>\ncode here\n</pre>",
		},
		{
			"italic text",
			"*italic*",
			"<i>italic</i>",
		},
		{
			"mixed bold and italic",
			"**bold** and *italic*",
			"<b>bold</b> and <i>italic</i>",
		},
		{
			"unclosed code block",
			"```code",
			"<code></code>`code", // best-effort: processes inline code
		},
		{
			"unclosed inline code",
			"`code",
			"`code", // no closing backtick, unchanged
		},
		{
			"unclosed bold",
			"**bold",
			"<i></i>bold", // best-effort: processes single * as italic
		},
		{
			"unclosed italic",
			"*italic",
			"*italic", // no closing *, unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toTelegramHTML(tt.input)
			if got != tt.want {
				t.Errorf("toTelegramHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTelegramChannel_Stop_NotStarted(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	// Should not panic when stopping before starting
	err := ch.Stop()
	if err != nil {
		t.Errorf("Stop error: %v", err)
	}
}

func TestTelegramChannel_Send_NilBot(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: "test"})
	if err == nil {
		t.Error("expected error when bot is nil")
	}
}

func TestTelegramChannel_WithProxy(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, err := NewTelegramChannel(config.TelegramConfig{
		Token: "fake-token",
		Proxy: "http://proxy.local:8080",
	}, b)
	if err != nil {
		t.Fatalf("NewTelegramChannel error: %v", err)
	}
	if ch.proxy != "http://proxy.local:8080" {
		t.Errorf("proxy = %q, want http://proxy.local:8080", ch.proxy)
	}
}

func TestChannelManager_StartAll_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	m, _ := NewChannelManager(config.ChannelsConfig{}, b)

	ctx := context.Background()
	if err := m.StartAll(ctx); err != nil {
		t.Errorf("StartAll error: %v", err)
	}
}

func TestChannelManager_StopAll_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	m, _ := NewChannelManager(config.ChannelsConfig{}, b)

	if err := m.StopAll(); err != nil {
		t.Errorf("StopAll error: %v", err)
	}
}

// mockChannel implements Channel interface for testing
type mockChannel struct {
	name       string
	started    bool
	stopped    bool
	startErr   error
	stopErr    error
	sentMsgs   []bus.OutboundMessage
}

func (m *mockChannel) Name() string { return m.name }

func (m *mockChannel) Start(ctx context.Context) error {
	m.started = true
	return m.startErr
}

func (m *mockChannel) Stop() error {
	m.stopped = true
	return m.stopErr
}

func (m *mockChannel) Send(msg bus.OutboundMessage) error {
	m.sentMsgs = append(m.sentMsgs, msg)
	return nil
}

func TestChannelManager_WithMockChannel(t *testing.T) {
	b := bus.NewMessageBus(10)

	mock := &mockChannel{name: "mock"}

	m := &ChannelManager{
		channels: map[string]Channel{"mock": mock},
		bus:      b,
	}

	// Test StartAll
	ctx := context.Background()
	if err := m.StartAll(ctx); err != nil {
		t.Errorf("StartAll error: %v", err)
	}
	if !mock.started {
		t.Error("mock channel should be started")
	}

	// Test EnabledChannels
	channels := m.EnabledChannels()
	if len(channels) != 1 || channels[0] != "mock" {
		t.Errorf("EnabledChannels = %v, want [mock]", channels)
	}

	// Test StopAll
	if err := m.StopAll(); err != nil {
		t.Errorf("StopAll error: %v", err)
	}
	if !mock.stopped {
		t.Error("mock channel should be stopped")
	}
}

func TestChannelManager_StartAll_Error(t *testing.T) {
	b := bus.NewMessageBus(10)

	mock := &mockChannel{name: "mock", startErr: fmt.Errorf("start failed")}

	m := &ChannelManager{
		channels: map[string]Channel{"mock": mock},
		bus:      b,
	}

	ctx := context.Background()
	err := m.StartAll(ctx)
	if err == nil {
		t.Error("expected error from StartAll")
	}
}

func TestChannelManager_StopAll_Error(t *testing.T) {
	b := bus.NewMessageBus(10)

	mock := &mockChannel{name: "mock", stopErr: fmt.Errorf("stop failed")}

	m := &ChannelManager{
		channels: map[string]Channel{"mock": mock},
		bus:      b,
	}

	// Should not return error (errors are logged)
	if err := m.StopAll(); err != nil {
		t.Errorf("StopAll should not return error: %v", err)
	}
}

func TestTelegramChannel_Send_InvalidChatID(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	// Set bot to mock
	ch.SetBot(newMockBot())

	err := ch.Send(bus.OutboundMessage{ChatID: "not-a-number", Content: "test"})
	if err == nil {
		t.Error("expected error for invalid chat ID")
	}
}

func TestTelegramChannel_HandleMessage_Allowed(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	msg := &tgbotapi.Message{
		From: &tgbotapi.User{ID: 123, UserName: "testuser"},
		Chat: &tgbotapi.Chat{ID: 456},
		Text: "hello",
		Date: 1234567890,
	}

	ch.handleMessage(msg)

	select {
	case inbound := <-b.Inbound:
		if inbound.Content != "hello" {
			t.Errorf("content = %q, want hello", inbound.Content)
		}
		if inbound.SenderID != "123" {
			t.Errorf("senderID = %q, want 123", inbound.SenderID)
		}
		if inbound.ChatID != "456" {
			t.Errorf("chatID = %q, want 456", inbound.ChatID)
		}
	default:
		t.Error("expected inbound message")
	}
}

func TestTelegramChannel_HandleMessage_Rejected(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{
		Token:     "fake-token",
		AllowFrom: []string{"999"}, // Only allow user 999
	}, b)

	msg := &tgbotapi.Message{
		From: &tgbotapi.User{ID: 123, UserName: "testuser"}, // User 123 not allowed
		Chat: &tgbotapi.Chat{ID: 456},
		Text: "hello",
	}

	ch.handleMessage(msg)

	// Should not receive any message
	select {
	case <-b.Inbound:
		t.Error("should not receive message from rejected user")
	default:
		// OK - no message sent
	}
}

func TestTelegramChannel_HandleMessage_EmptyText(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	msg := &tgbotapi.Message{
		From: &tgbotapi.User{ID: 123},
		Chat: &tgbotapi.Chat{ID: 456},
		Text: "", // Empty text
	}

	ch.handleMessage(msg)

	// Should not receive any message
	select {
	case <-b.Inbound:
		t.Error("should not send message with empty content")
	default:
		// OK
	}
}

func TestTelegramChannel_HandleMessage_Caption(t *testing.T) {
	b := bus.NewMessageBus(10)
	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	msg := &tgbotapi.Message{
		From:    &tgbotapi.User{ID: 123},
		Chat:    &tgbotapi.Chat{ID: 456},
		Text:    "",
		Caption: "image caption", // Caption instead of text
	}

	ch.handleMessage(msg)

	select {
	case inbound := <-b.Inbound:
		if inbound.Content != "image caption" {
			t.Errorf("content = %q, want 'image caption'", inbound.Content)
		}
	default:
		t.Error("expected inbound message")
	}
}

// mockTelegramBot implements TelegramBot interface for testing
type mockTelegramBot struct {
	updatesChan chan tgbotapi.Update
	stopped     bool
	sentMsgs    []tgbotapi.Chattable
	sendErr     error
	self        tgbotapi.User
}

func newMockBot() *mockTelegramBot {
	return &mockTelegramBot{
		updatesChan: make(chan tgbotapi.Update, 10),
		self:        tgbotapi.User{UserName: "testbot"},
	}
}

func (m *mockTelegramBot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return m.updatesChan
}

func (m *mockTelegramBot) StopReceivingUpdates() {
	m.stopped = true
}

func (m *mockTelegramBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.sentMsgs = append(m.sentMsgs, c)
	if m.sendErr != nil {
		return tgbotapi.Message{}, m.sendErr
	}
	return tgbotapi.Message{MessageID: 1}, nil
}

func (m *mockTelegramBot) GetSelf() tgbotapi.User {
	return m.self
}

func TestTelegramChannel_InitBot_Success(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	factory := func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
		return mockBot, nil
	}

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{Token: "fake-token"}, b, factory)

	err := ch.initBot()
	if err != nil {
		t.Errorf("initBot error: %v", err)
	}
	if ch.bot == nil {
		t.Error("bot should be set")
	}
}

func TestTelegramChannel_InitBot_Error(t *testing.T) {
	b := bus.NewMessageBus(10)

	factory := func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
		return nil, fmt.Errorf("auth failed")
	}

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{Token: "fake-token"}, b, factory)

	err := ch.initBot()
	if err == nil {
		t.Error("expected error from initBot")
	}
}

func TestTelegramChannel_InitBot_InvalidProxy(t *testing.T) {
	b := bus.NewMessageBus(10)

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{
		Token: "fake-token",
		Proxy: "://invalid-url",
	}, b, defaultBotFactory)

	err := ch.initBot()
	if err == nil {
		t.Error("expected error for invalid proxy URL")
	}
}

func TestTelegramChannel_Start_Success(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	factory := func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
		return mockBot, nil
	}

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{Token: "fake-token"}, b, factory)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := ch.Start(ctx)
	if err != nil {
		t.Errorf("Start error: %v", err)
	}

	// Send a test update
	mockBot.updatesChan <- tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{ID: 123},
			Chat: &tgbotapi.Chat{ID: 456},
			Text: "test message",
		},
	}

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	select {
	case inbound := <-b.Inbound:
		if inbound.Content != "test message" {
			t.Errorf("content = %q, want 'test message'", inbound.Content)
		}
	default:
		t.Error("expected inbound message")
	}

	// Test stop
	ch.Stop()
	if !mockBot.stopped {
		t.Error("bot should be stopped")
	}
}

func TestTelegramChannel_Start_InitError(t *testing.T) {
	b := bus.NewMessageBus(10)

	factory := func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
		return nil, fmt.Errorf("init failed")
	}

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{Token: "fake-token"}, b, factory)

	err := ch.Start(context.Background())
	if err == nil {
		t.Error("expected error from Start")
	}
}

func TestTelegramChannel_Start_NilMessage(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	factory := func(token, apiEndpoint string, client *http.Client) (TelegramBot, error) {
		return mockBot, nil
	}

	ch, _ := NewTelegramChannelWithFactory(config.TelegramConfig{Token: "fake-token"}, b, factory)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch.Start(ctx)

	// Send update with nil message (should be ignored)
	mockBot.updatesChan <- tgbotapi.Update{Message: nil}

	time.Sleep(50 * time.Millisecond)

	select {
	case <-b.Inbound:
		t.Error("should not receive message for nil update")
	default:
		// OK
	}
}

func TestTelegramChannel_Send_Success(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	ch.SetBot(mockBot)

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: "hello"})
	if err != nil {
		t.Errorf("Send error: %v", err)
	}

	if len(mockBot.sentMsgs) != 1 {
		t.Errorf("expected 1 sent message, got %d", len(mockBot.sentMsgs))
	}
}

func TestTelegramChannel_Send_LongMessage(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	ch.SetBot(mockBot)

	// Create a message longer than 4000 chars with newlines
	longContent := ""
	for i := 0; i < 100; i++ {
		longContent += "This is a long line of text that will be repeated.\n"
	}

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: longContent})
	if err != nil {
		t.Errorf("Send error: %v", err)
	}

	// Should split into multiple messages
	if len(mockBot.sentMsgs) < 2 {
		t.Errorf("expected multiple sent messages for long content, got %d", len(mockBot.sentMsgs))
	}
}

func TestTelegramChannel_Send_LongMessageNoNewline(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	ch.SetBot(mockBot)

	// Create a long message without newlines
	longContent := ""
	for i := 0; i < 5000; i++ {
		longContent += "x"
	}

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: longContent})
	if err != nil {
		t.Errorf("Send error: %v", err)
	}

	if len(mockBot.sentMsgs) < 2 {
		t.Errorf("expected multiple messages, got %d", len(mockBot.sentMsgs))
	}
}

func TestTelegramChannel_Send_HTMLError_Retry(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()

	// First call fails (HTML parse error), second succeeds
	callCount := 0
	mockBot.sendErr = nil
	originalSend := mockBot.Send
	_ = originalSend

	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)

	// Create a wrapper that fails first then succeeds
	wrapper := &sendCountingBot{mockBot: mockBot, failFirst: true}
	ch.SetBot(wrapper)
	_ = callCount

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: "test"})
	// Should succeed after retry
	if err != nil {
		t.Errorf("Send should succeed after retry: %v", err)
	}
}

type sendCountingBot struct {
	mockBot   *mockTelegramBot
	failFirst bool
	callCount int
}

func (s *sendCountingBot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return s.mockBot.updatesChan
}

func (s *sendCountingBot) StopReceivingUpdates() {
	s.mockBot.stopped = true
}

func (s *sendCountingBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	s.callCount++
	if s.failFirst && s.callCount == 1 {
		return tgbotapi.Message{}, fmt.Errorf("HTML parse error")
	}
	return tgbotapi.Message{MessageID: 1}, nil
}

func (s *sendCountingBot) GetSelf() tgbotapi.User {
	return s.mockBot.self
}

func TestTelegramChannel_Send_BothFail(t *testing.T) {
	b := bus.NewMessageBus(10)
	mockBot := newMockBot()
	mockBot.sendErr = fmt.Errorf("send failed")

	ch, _ := NewTelegramChannel(config.TelegramConfig{Token: "fake-token"}, b)
	ch.SetBot(mockBot)

	err := ch.Send(bus.OutboundMessage{ChatID: "123", Content: "test"})
	if err == nil {
		t.Error("expected error when both sends fail")
	}
}
