// Package testkit provides a deterministic Universal WebSocket client for bot
// protocol tests. It never embeds credentials in URLs or logs request payloads.
package testkit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"purr-chat-server/internal/onebot"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
}

func Dial(ctx context.Context, endpoint, token string) (*Client, error) {
	header := make(http.Header)
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, header)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendAction(action string, params any, echo any) error {
	payload, err := json.Marshal(map[string]any{"action": action, "params": params, "echo": echo})
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *Client) Ack(eventID string, seq int64, echo any) error {
	return c.SendAction("ack_event", map[string]any{"event_id": eventID, "seq": seq}, echo)
}

func (c *Client) ReadActionResponse(timeout time.Duration) (onebot.ActionResponse, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	_, payload, err := c.conn.ReadMessage()
	if err != nil {
		return onebot.ActionResponse{}, err
	}
	var response onebot.ActionResponse
	err = json.Unmarshal(payload, &response)
	return response, err
}

func (c *Client) ReadEvent(timeout time.Duration) (onebot.Event, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	_, payload, err := c.conn.ReadMessage()
	if err != nil {
		return onebot.Event{}, err
	}
	var event onebot.Event
	err = json.Unmarshal(payload, &event)
	return event, err
}
