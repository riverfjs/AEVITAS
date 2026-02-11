package channel

import (
	"context"
	"fmt"
	"testing"

	sdklogger "github.com/cexll/agentsdk-go/pkg/logger"
	"github.com/stellarlinkco/myclaw/internal/bus"
	"github.com/stellarlinkco/myclaw/internal/config"
)

// ===== BaseChannel 测试 =====

func TestBaseChannel_Name(t *testing.T) {
	b := bus.NewMessageBus(10)
	logger := sdklogger.NewDefault()
	ch := NewBaseChannel("test", b, nil, logger)
	if ch.Name() != "test" {
		t.Errorf("Name = %q, want test", ch.Name())
	}
}

func TestBaseChannel_IsAllowed_NoFilter(t *testing.T) {
	b := bus.NewMessageBus(10)
	logger := sdklogger.NewDefault()
	ch := NewBaseChannel("test", b, nil, logger)
	if !ch.IsAllowed("anyone") {
		t.Error("should allow anyone when allowFrom is empty")
	}
}

func TestBaseChannel_IsAllowed_WithFilter(t *testing.T) {
	b := bus.NewMessageBus(10)
	logger := sdklogger.NewDefault()
	ch := NewBaseChannel("test", b, []string{"user1", "user2"}, logger)

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

// ===== ChannelManager 测试 =====

func TestChannelManager_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	logger := sdklogger.NewDefault()
	m, err := NewChannelManager(config.ChannelsConfig{}, b, logger)
	if err != nil {
		t.Fatalf("NewChannelManager error: %v", err)
	}
	if len(m.EnabledChannels()) != 0 {
		t.Errorf("expected 0 enabled channels, got %d", len(m.EnabledChannels()))
	}
}

func TestChannelManager_StartAll_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	m, _ := NewChannelManager(config.ChannelsConfig{}, b, sdklogger.NewDefault())

	ctx := context.Background()
	if err := m.StartAll(ctx); err != nil {
		t.Errorf("StartAll error: %v", err)
	}
}

func TestChannelManager_StopAll_Empty(t *testing.T) {
	b := bus.NewMessageBus(10)
	m, _ := NewChannelManager(config.ChannelsConfig{}, b, sdklogger.NewDefault())

	if err := m.StopAll(); err != nil {
		t.Errorf("StopAll error: %v", err)
	}
}

// ===== Mock Channel for Testing =====

type mockChannel struct {
	name     string
	started  bool
	stopped  bool
	startErr error
	stopErr  error
	sentMsgs []bus.OutboundMessage
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
		logger:   sdklogger.NewDefault(),
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
		logger:   sdklogger.NewDefault(),
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
		logger:   sdklogger.NewDefault(),
	}

	// Should not return error (errors are logged)
	if err := m.StopAll(); err != nil {
		t.Errorf("StopAll should not return error: %v", err)
	}
}
