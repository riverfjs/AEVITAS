// Package rpc provides a WebSocket RPC server with the same wire protocol as openclaw:
//
//	Request:  { "type":"req",   "id":"<uuid>", "method":"<name>", "params":<any> }
//	Response: { "type":"res",   "id":"<uuid>", "ok":<bool>, "payload":<any>, "error":<ErrorShape> }
//	Event:    { "type":"event", "event":"<name>", "payload":<any> }
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ── Wire types ────────────────────────────────────────────────────────────────

type RequestFrame struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type ErrorShape struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseFrame struct {
	Type    string      `json:"type"`
	ID      string      `json:"id"`
	Ok      bool        `json:"ok"`
	Payload interface{} `json:"payload,omitempty"`
	Error   *ErrorShape `json:"error,omitempty"`
}

type EventFrame struct {
	Type    string      `json:"type"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload,omitempty"`
}

// ── Handler types ─────────────────────────────────────────────────────────────

// RespondFn sends a response back to the caller.
// Pass errMsg="" and ok=true for success; ok=false + errMsg for error.
type RespondFn func(ok bool, payload interface{}, errMsg string)

// Handler processes a single RPC method call.
type Handler func(params json.RawMessage, respond RespondFn)

// ── Server ────────────────────────────────────────────────────────────────────

type Server struct {
	handlers map[string]Handler
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	logger   Logger
}

// Logger is a minimal interface so the server doesn't import a concrete logger.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func NewServer(logger Logger) *Server {
	return &Server{
		handlers: make(map[string]Handler),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // local only
		},
		logger: logger,
	}
}

// Register adds a handler for the given method name.
func (s *Server) Register(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// Start listens on addr and serves WebSocket connections until ctx is done.
// addr is in the form "host:port", e.g. "127.0.0.1:18790".
func (s *Server) Start(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("rpc listen %s: %w", addr, err)
	}
	s.logger.Infof("[rpc] listening on ws://%s", addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWS)
	srv := &http.Server{Handler: mux}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("[rpc] server error: %v", err)
		}
	}()

	return nil
}

// handleWS upgrades the HTTP connection to WebSocket and reads frames.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorf("[rpc] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	s.logger.Infof("[rpc] client connected from %s", r.RemoteAddr)

	send := func(v interface{}) {
		data, err := json.Marshal(v)
		if err != nil {
			s.logger.Errorf("[rpc] marshal error: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			s.logger.Errorf("[rpc] write error: %v", err)
		}
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// 1006 (abnormal closure) is expected from one-shot clients that
			// close the connection without a proper WebSocket close handshake.
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				s.logger.Infof("[rpc] client disconnected: %s", r.RemoteAddr)
			} else {
				s.logger.Errorf("[rpc] read error: %v", err)
			}
			return
		}

		var req RequestFrame
		if err := json.Unmarshal(data, &req); err != nil || req.Type != "req" || req.ID == "" || req.Method == "" {
			send(ResponseFrame{
				Type:  "res",
				ID:    req.ID,
				Ok:    false,
				Error: &ErrorShape{Code: "INVALID_REQUEST", Message: "invalid request frame"},
			})
			continue
		}

		s.mu.RLock()
		h, ok := s.handlers[req.Method]
		s.mu.RUnlock()

		if !ok {
			send(ResponseFrame{
				Type:  "res",
				ID:    req.ID,
				Ok:    false,
				Error: &ErrorShape{Code: "METHOD_NOT_FOUND", Message: fmt.Sprintf("unknown method: %s", req.Method)},
			})
			continue
		}

		// Call handler inline (handlers are expected to be fast / non-blocking for our use case)
		h(req.Params, func(okFlag bool, payload interface{}, errMsg string) {
			res := ResponseFrame{Type: "res", ID: req.ID, Ok: okFlag}
			if okFlag {
				res.Payload = payload
			} else {
				res.Error = &ErrorShape{Code: "INTERNAL_ERROR", Message: errMsg}
			}
			send(res)
		})
	}
}

// Broadcast sends an event frame to all connected clients.
// (Reserved for future use — presence, job-completion notifications, etc.)
func (s *Server) Broadcast(event string, payload interface{}) {
	// Currently no-op. Add client registry here when push events are needed.
}
