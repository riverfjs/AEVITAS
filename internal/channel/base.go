package channel

import (
	"context"

	sdklogger "github.com/riverfjs/agentsdk-go/pkg/logger"
	"github.com/stellarlinkco/myclaw/internal/bus"
)

type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Send(msg bus.OutboundMessage) error
}

type BaseChannel struct {
	name      string
	bus       *bus.MessageBus
	allowFrom map[string]bool
	logger    sdklogger.Logger
}

func NewBaseChannel(name string, b *bus.MessageBus, allowFrom []string, logger sdklogger.Logger) BaseChannel {
	af := make(map[string]bool, len(allowFrom))
	for _, id := range allowFrom {
		af[id] = true
	}
	return BaseChannel{name: name, bus: b, allowFrom: af, logger: logger}
}

func (c *BaseChannel) Name() string {
	return c.name
}

func (c *BaseChannel) IsAllowed(senderID string) bool {
	if len(c.allowFrom) == 0 {
		return true
	}
	return c.allowFrom[senderID]
}
