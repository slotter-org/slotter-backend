package socket

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	
	"github.com/slotter-org/slotter-backend/internal/logger"
)

type InboundMessage struct {
	Action			string					`json:"action,omitempty"`
	Channel			string					`json:"channel,omitempty"`
}

const (
	OutboundChanBuffer = 256
	WriteWait = 10 * time.Second
	PongWait = 60 * time.Second
	PingInterval = 25 * time.Second
)

type Client struct {
	ID						uuid.UUID
	Conn					*websocket.Conn
	Hub						*Hub
	logger				*logger.Logger
	cancelFn			context.CancelFunc
	Outbound			chan Message
}

func NewClient(conn *websocket.Conn, hub *Hub, log *logger.Logger) *Client {
	return &Client{
		ID:					uuid.New(),
		Conn:				conn,
		Hub:				hub,
		logger:			log,
		Outbound:		make(chan Message, OutboundChanBuffer),
	}
}

func (c *Client) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFn = cancel
	go c.writeLoop(ctx)
	c.readLoop(ctx)
}

func (c *Client) readLoop(ctx context.Context) {
	defer c.Close()
	c.Conn.SetReadDeadline(time.Now().Add(PongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			netErr, ok := err.(net.Error)
			if !ok || !netErr.Temporary() {
				c.logger.Debug("Websocket read error => closing client", "error", err)
				break
			}
		}
		var inbound InboundMessage
		if err := json.Unmarshal(data, &inbound); err != nil {
			c.logger.Debug("Failed to unmarshal inbound message", "error", err, "raw", string(data))
			continue
		}
		switch inbound.Action {
		case "subscribe":
			if inbound.Channel != "" {
				c.Hub.Subscribe(c, []string{inbound.Channel})
				c.logger.Debug("Client requested subscribe", "channel", inbound.Channel, "client", c.ID)
			}
		case "unsubscribe":
			if inbound.Channel != "" {
				c.Hub.UnsubscribeFromChannel(c, inbound.Channel)
				c.logger.Debug("Client requested unsubscribe", "channel", inbound.Channel, "client", c.ID)
			}
		default:
			c.logger.Debug("Inbound WebSocket message unhandled", "client", c.ID, "message", inbound)
		}
	}
}

func (c *Client) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.Outbound:
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				c.logger.Debug("Outbound channel closed, shutting down writeLoop", "client", c.ID)
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.writeJSON(msg); err != nil {
				c.logger.Warn("Failed writing JSON to client", "client", c.ID, "error", err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Debug("Ping error => shutting down client", "client", c.ID, "error", err)
				return
			}
		case <-ctx.Done():
			c.logger.Debug("writeLoop context done => shutting down", "client", c.ID)
			return
		}
	}
}

func (c *Client) writeJSON(v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	_, writeErr := w.Write(payload)
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func (c *Client) Close() {
	c.logger.Debug("Closing client connection", "client", c.ID)
	if c.cancelFn != nil {
		c.cancelFn()
	}
	_ = c.Conn.Close()
	close(c.Outbound)
	c.Hub.Unsubscribe(c)
}
