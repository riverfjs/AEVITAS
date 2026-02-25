package channel

import (
	"context"
	"fmt"
	"sync"

	sdklogger "github.com/riverfjs/agentsdk-go/pkg/logger"
	"github.com/stellarlinkco/myclaw/internal/bus"
	"github.com/stellarlinkco/myclaw/internal/config"
)

type ChannelManager struct {
	channels map[string]Channel
	bus      *bus.MessageBus
	logger   sdklogger.Logger
}

func NewChannelManager(cfg config.ChannelsConfig, b *bus.MessageBus, logger sdklogger.Logger) (*ChannelManager, error) {
	m := &ChannelManager{
		channels: make(map[string]Channel),
		bus:      b,
		logger:   logger,
	}

	if cfg.Telegram.Enabled {
		ch, err := NewTelegramChannel(cfg.Telegram, b, logger)
		if err != nil {
			return nil, fmt.Errorf("init telegram channel: %w", err)
		}
		m.channels[ch.Name()] = ch
		b.SubscribeOutbound(ch.Name(), func(msg bus.OutboundMessage) {
			if err := ch.Send(msg); err != nil {
				logger.Errorf("[channel-mgr] send to %s failed: %v", ch.Name(), err)
			}
		})
	}

	if cfg.Feishu.Enabled {
		ch, err := NewFeishuChannel(cfg.Feishu, b, logger)
		if err != nil {
			return nil, fmt.Errorf("init feishu channel: %w", err)
		}
		m.channels[ch.Name()] = ch
		b.SubscribeOutbound(ch.Name(), func(msg bus.OutboundMessage) {
			if err := ch.Send(msg); err != nil {
				logger.Errorf("[channel-mgr] send to %s failed: %v", ch.Name(), err)
			}
		})
	}

	if cfg.WeCom.Enabled {
		ch, err := NewWeComChannel(cfg.WeCom, b, logger)
		if err != nil {
			return nil, fmt.Errorf("init wecom channel: %w", err)
		}
		m.channels[ch.Name()] = ch
		b.SubscribeOutbound(ch.Name(), func(msg bus.OutboundMessage) {
			if err := ch.Send(msg); err != nil {
				logger.Errorf("[channel-mgr] send to %s failed: %v", ch.Name(), err)
			}
		})
	}

	return m, nil
}

func (m *ChannelManager) StartAll(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(m.channels))

	for name, ch := range m.channels {
		wg.Add(1)
		go func(name string, ch Channel) {
			defer wg.Done()
			m.logger.Infof("[channel-mgr] starting %s", name)
			if err := ch.Start(ctx); err != nil {
				errCh <- fmt.Errorf("%s: %w", name, err)
			}
		}(name, ch)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

func (m *ChannelManager) StopAll() error {
	for name, ch := range m.channels {
		m.logger.Infof("[channel-mgr] stopping %s", name)
		if err := ch.Stop(); err != nil {
			m.logger.Errorf("[channel-mgr] error stopping %s: %v", name, err)
		}
	}
	return nil
}

func (m *ChannelManager) EnabledChannels() []string {
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}
