package channel

import (
	"errors"
	"testing"

	"github.com/riverfjs/aevitas/internal/bus"
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

func TestCommandHandler_HandleStart(t *testing.T) {
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	
	handler := NewCommandHandler(resetter, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	
	handler := NewCommandHandler(resetter, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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
	handler := NewCommandHandler(nil, "")
	
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

