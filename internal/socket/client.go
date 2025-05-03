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

//---------------------------------------------------------------------
// Public message formats  (unchanged)
//---------------------------------------------------------------------
type InboundMessage struct {
	Action  string `json:"action,omitempty"`  // "subscribe" | "unsubscribe" | …
	Channel string `json:"channel,omitempty"` // channel name, etc.
}

type Message struct {
	Channel string      `json:"channel,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

//---------------------------------------------------------------------
// Tunables  (unchanged)
//---------------------------------------------------------------------
const (
	OutboundChanBuffer = 256

	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

//---------------------------------------------------------------------
// Client
//---------------------------------------------------------------------
type Client struct {
	ID        uuid.UUID
	Conn      *websocket.Conn
	Hub       *Hub
	Log       *logger.Logger
	cancelFn  context.CancelFunc
	Outbound  chan Message
}

// NewClient constructs a fully-initialised Client.  The cancel function comes
// from the handler so the HTTP context can finish while the WS lives on.
func NewClient(conn *websocket.Conn, hub *Hub, uid uuid.UUID,
	cancel context.CancelFunc, log *logger.Logger) *Client {

	return &Client{
		ID:       uid,
		Conn:     conn,
		Hub:      hub,
		Log:      log,
		cancelFn: cancel,
		Outbound: make(chan Message, OutboundChanBuffer),
	}
}

//---------------------------------------------------------------------
// Public entry-points – invoked from handlers
//---------------------------------------------------------------------
func (c *Client) ReadLoop(ctx context.Context) { c.readLoop(ctx) }
func (c *Client) WriteLoop(ctx context.Context) { c.writeLoop(ctx) }

//---------------------------------------------------------------------
// readLoop – inbound → Hub
//---------------------------------------------------------------------
func (c *Client) readLoop(ctx context.Context) {
	defer c.close()

	c.Conn.SetReadLimit(1 << 20)                           // 1 MiB
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))   // initial deadline
	c.Conn.SetPongHandler(func(string) error {             // keep-alive handler
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return

		default:
			_, data, err := c.Conn.ReadMessage()
			if err != nil {
				if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
					c.Log.Debug("websocket read error → closing client", "error", err)
					return
				}
				continue
			}

			var inbound InboundMessage
			if err := json.Unmarshal(data, &inbound); err != nil {
				c.Log.Debug("failed to unmarshal inbound message",
					"error", err, "raw", string(data))
				continue
			}

			switch inbound.Action {
			case "subscribe":
				if inbound.Channel != "" {
					c.Hub.Subscribe(c, []string{inbound.Channel})
					c.Log.Debug("client subscribed",
						"channel", inbound.Channel, "client", c.ID)
				}
			case "unsubscribe":
				if inbound.Channel != "" {
					c.Hub.UnsubscribeFromChannel(c, inbound.Channel)
					c.Log.Debug("client unsubscribed",
						"channel", inbound.Channel, "client", c.ID)
				}
			default:
				c.Log.Debug("inbound WS message unhandled",
					"client", c.ID, "message", inbound)
			}
		}
	}
}

//---------------------------------------------------------------------
// writeLoop – Hub → outbound
//---------------------------------------------------------------------
func (c *Client) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case <-ctx.Done():
			c.Log.Debug("writeLoop ctx done → shutdown", "client", c.ID)
			return

		case msg, ok := <-c.Outbound:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Log.Debug("outbound channel closed → shutdown", "client", c.ID)
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.writeJSON(msg); err != nil {
				c.Log.Warn("failed writing JSON", "client", c.ID, "error", err)
				return
			}

		case <-ticker.C: // keep-alive ping
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Log.Debug("ping error → shutdown", "client", c.ID, "error", err)
				return
			}
		}
	}
}

//---------------------------------------------------------------------
// utilities
//---------------------------------------------------------------------
func (c *Client) writeJSON(v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err = w.Write(payload); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (c *Client) close() {
	c.Log.Debug("closing client connection", "client", c.ID)
	if c.cancelFn != nil {
		c.cancelFn() // stop the sibling pump
	}
	_ = c.Conn.Close()
	close(c.Outbound)
	c.Hub.Unsubscribe(c)
}
