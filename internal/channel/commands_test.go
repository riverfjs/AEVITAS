package channel

import (
	"errors"
	"strings"
	"testing"

	"github.com/riverfjs/aevitas/internal/bus"
	"github.com/riverfjs/agentsdk-go/pkg/api"
)

// mockSessionResetter implements SessionResetter interface
type mockSessionResetter struct {
	clearFunc func(sessionID string) error
}

func (m *mockSessionResetter) ClearSession(sessionID string) error {
	if m.clearFunc != nil {
		return m.clearFunc(sessionID)
	}
	return nil
}

type mockUsageReporter struct {
	mockSessionResetter
	session *api.SessionTokenStats
	total   *api.SessionTokenStats
}

func (m *mockUsageReporter) GetSessionStats(sessionID string) *api.SessionTokenStats {
	return m.session
}

func (m *mockUsageReporter) GetTotalStats() *api.SessionTokenStats {
	return m.total
}

func TestCommandHandler_HandleStart(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/start",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /start to be handled")
	}
	
	if result.Response == "" {
		t.Error("Expected non-empty response")
	}
	
	if !contains(result.Response, "Aevitas") {
		t.Errorf("Expected 'Aevitas' in response, got: %s", result.Response)
	}
}

func TestCommandHandler_HandleHelp(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/help",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /help to be handled")
	}
	
	if result.Response == "" {
		t.Error("Expected non-empty response")
	}
	
	if !contains(result.Response, "/start") || !contains(result.Response, "/reset") {
		t.Errorf("Expected command list in response, got: %s", result.Response)
	}
}

func TestCommandHandler_HandleReset_Success(t *testing.T) {
	resetCalled := false
	var capturedSessionKey string
	
	resetter := &mockSessionResetter{
		clearFunc: func(sessionID string) error {
			resetCalled = true
			capturedSessionKey = sessionID
			return nil
		},
	}
	
	handler := NewCommandHandler(resetter, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/reset",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /reset to be handled")
	}
	
	if !resetCalled {
		t.Error("Expected reset function to be called")
	}
	
	expectedSessionKey := "telegram:123"
	if capturedSessionKey != expectedSessionKey {
		t.Errorf("Expected session key %s, got %s", expectedSessionKey, capturedSessionKey)
	}
	
	if !contains(result.Response, "✅") || !contains(result.Response, "Reset") {
		t.Errorf("Expected success message, got: %s", result.Response)
	}
}

func TestCommandHandler_HandleReset_NoFunction(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/reset",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /reset to be handled")
	}
	
	if !contains(result.Response, "not available") {
		t.Errorf("Expected unavailable message, got: %s", result.Response)
	}
}

func TestCommandHandler_HandleReset_Error(t *testing.T) {
	testErr := errors.New("reset failed")
	
	resetter := &mockSessionResetter{
		clearFunc: func(sessionID string) error {
			return testErr
		},
	}
	
	handler := NewCommandHandler(resetter, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/reset",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /reset to be handled")
	}
	
	if !contains(result.Response, "❌") || !contains(result.Response, "Failed") {
		t.Errorf("Expected error message, got: %s", result.Response)
	}
}

func TestCommandHandler_UnknownCommand(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/unknown",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected unknown command to be handled with error message")
	}
	
	if !contains(result.Response, "Unknown command") || !contains(result.Response, "/unknown") {
		t.Errorf("Expected unknown command error message, got: %s", result.Response)
	}
}

func TestCommandHandler_NotACommand(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "hello world",
	}
	
	result := handler.HandleCommand(msg)
	
	if result.Handled {
		t.Error("Expected regular message not to be handled as command")
	}
	
	if result.Response != "" {
		t.Errorf("Expected empty response for regular message, got: %s", result.Response)
	}
}

func TestCommandHandler_EmptyContent(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "",
	}
	
	result := handler.HandleCommand(msg)
	
	if result.Handled {
		t.Error("Expected empty content not to be handled")
	}
}

func TestCommandHandler_CaseInsensitive(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	testCases := []string{"/START", "/Start", "/StArT", "/HELP", "/Help", "/RESET", "/Reset"}
	
	for _, cmd := range testCases {
		msg := bus.InboundMessage{
			Channel:  "test",
			ChatID:   "123",
			SenderID: "user1",
			Content:  cmd,
		}
		
		result := handler.HandleCommand(msg)
		
		if !result.Handled {
			t.Errorf("Expected %s to be handled (case insensitive)", cmd)
		}
	}
}

func TestCommandHandler_WithWhitespace(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "  /start  ",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /start with whitespace to be handled")
	}
}

func TestCommandHandler_CommandWithArgs(t *testing.T) {
	handler := NewCommandHandler(nil, "", 200000)
	
	msg := bus.InboundMessage{
		Channel:  "test",
		ChatID:   "123",
		SenderID: "user1",
		Content:  "/start some extra args",
	}
	
	result := handler.HandleCommand(msg)
	
	if !result.Handled {
		t.Error("Expected /start with args to be handled")
	}
}

func TestCommandHandler_HandleUsage_Default(t *testing.T) {
	reporter := &mockUsageReporter{
		session: &api.SessionTokenStats{
			TotalInput:          100,
			TotalOutput:         50,
			TotalTokens:         150,
			CacheCreated:        10,
			CacheRead:           20,
		},
	}
	handler := NewCommandHandler(reporter, "", 200000)
	msg := bus.InboundMessage{Channel: "telegram", ChatID: "1", SenderID: "u", Content: "/usage"}
	result := handler.HandleCommand(msg)
	if !result.Handled {
		t.Fatal("expected /usage handled")
	}
	if result.Event != "usage_hud" {
		t.Fatalf("expected usage_hud event, got %q", result.Event)
	}
	if !strings.Contains(result.Response, "Usage (Current Session)") ||
		!strings.Contains(result.Response, "Context window:") ||
		!strings.Contains(result.Response, "⬜") ||
		!strings.Contains(result.Response, "(100/200000)") ||
		!strings.Contains(result.Response, "Total billed tokens: 150") ||
		!strings.Contains(result.Response, "Input: 100 | Output: 50 | Cache: 30 | Total: 150") {
		t.Fatalf("unexpected response: %s", result.Response)
	}
}

func TestCommandHandler_HandleUsage_Total(t *testing.T) {
	reporter := &mockUsageReporter{
		total: &api.SessionTokenStats{
			TotalInput:   500,
			TotalOutput:  200,
			TotalTokens:  700,
		},
	}
	handler := NewCommandHandler(reporter, "", 200000)
	msg := bus.InboundMessage{Channel: "telegram", ChatID: "1", SenderID: "u", Content: "/usage total"}
	result := handler.HandleCommand(msg)
	if !result.Handled {
		t.Fatal("expected /usage total handled")
	}
	if result.Event != "usage_hud" {
		t.Fatalf("expected usage_hud event, got %q", result.Event)
	}
	if !strings.Contains(result.Response, "Usage (Total)") ||
		!strings.Contains(result.Response, "⬜") ||
		!strings.Contains(result.Response, "(500/200000)") ||
		!strings.Contains(result.Response, "Total billed tokens: 700") ||
		!strings.Contains(result.Response, "Input: 500 | Output: 200 | Cache: 0 | Total: 700") {
		t.Fatalf("unexpected response: %s", result.Response)
	}
}

func TestCommandHandler_HandleUsage_NoReporter(t *testing.T) {
	handler := NewCommandHandler(&mockSessionResetter{}, "", 200000)
	msg := bus.InboundMessage{Channel: "telegram", ChatID: "1", SenderID: "u", Content: "/usage"}
	result := handler.HandleCommand(msg)
	if !result.Handled {
		t.Fatal("expected /usage handled")
	}
	if !strings.Contains(result.Response, "not available") {
		t.Fatalf("unexpected response: %s", result.Response)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

