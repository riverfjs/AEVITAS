package main

// gateway.go — WebSocket RPC client for myclaw gateway.
// Wire protocol mirrors openclaw:
//   req: { "type":"req", "id":"<id>", "method":"<name>", "params":<any> }
//   res: { "type":"res", "id":"<id>", "ok":<bool>, "payload":<any>, "error":{...} }

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const gatewayURL = "ws://127.0.0.1:18790"

type rpcRequest struct {
	Type   string      `json:"type"`
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Ok      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// callGateway opens a WS connection, sends one RPC request, waits for the
// matching response, and returns the raw payload bytes.
func callGateway(method string, params interface{}) (json.RawMessage, error) {
	conn, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to gateway (%s): %w", gatewayURL, err)
	}
	defer func() {
		// Send proper close frame before closing
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}()

	reqID := fmt.Sprintf("todoist-%d", time.Now().UnixNano())
	req := rpcRequest{Type: "req", ID: reqID, Method: method, Params: params}
	if err := conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("send rpc request: %w", err)
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read rpc response: %w", err)
		}
		var res rpcResponse
		if err := json.Unmarshal(data, &res); err != nil {
			continue
		}
		if res.Type != "res" || res.ID != reqID {
			continue
		}
		if !res.Ok {
			msg := "rpc error"
			if res.Error != nil {
				msg = res.Error.Message
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return res.Payload, nil
	}
}
