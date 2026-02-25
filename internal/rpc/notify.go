package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/stellarlinkco/myclaw/internal/bus"
)

// RegisterNotifyHandlers registers the notify.* RPC methods on the WebSocket server.
//
// This allows external scripts (skills, cron commands) to push arbitrary text
// messages to a user's chat session without going through the agent runtime.
// The message is written directly to bus.Outbound and delivered by whichever
// channel adapter (Telegram, Feishu, …) subscribes to that channel name.
//
// Designed for use by long-running skill scripts that want to report progress
// while the agent is waiting for the tool result — e.g. "⏳ Fetching financials…"
//
// Wire protocol (WebSocket JSON-RPC):
//
//	Request:  { "type":"req", "id":"<uuid>", "method":"notify.send",
//	            "params": { "channel":"telegram", "chatId":"123456", "message":"..." } }
//	Response: { "type":"res", "id":"<uuid>", "ok":true, "payload":{"ok":true} }
//
// channel defaults to "telegram" if omitted.
func RegisterNotifyHandlers(s *Server, b *bus.MessageBus) {
	s.Register("notify.send", func(params json.RawMessage, respond RespondFn) {
		var p struct {
			Channel string `json:"channel"`
			ChatID  string `json:"chatId"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			respond(false, nil, fmt.Sprintf("invalid params: %v", err))
			return
		}
		if p.Message == "" {
			respond(false, nil, "missing message")
			return
		}
		if p.Channel == "" {
			p.Channel = "telegram"
		}
		// Write to outbound bus — channel adapter delivers to the target chat.
		b.Outbound <- bus.OutboundMessage{
			Channel: p.Channel,
			ChatID:  p.ChatID,
			Content: p.Message,
		}
		respond(true, map[string]interface{}{"ok": true}, "")
	})
}
