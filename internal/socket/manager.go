package socket

import (
    "context"
    "sync"

    "github.com/google/uuid"
    "github.com/slotter-org/slotter-backend/internal/logger"
)

// Message is unchanged
type Message struct {
    Channel string      `json:"channel"`
    Data    interface{} `json:"data"`
}

type Hub struct {
    log       *logger.Logger
    mu        sync.RWMutex
    channels  map[string]map[uuid.UUID]*Client

    // Add a pointer to the RedisPubSub, optional
    redisPubSub *RedisPubSub
}

// Modify NewHub to accept an optional redisPubSub:
func NewHub(logger *logger.Logger) *Hub {
    return &Hub{
        log:       logger,
        channels:  make(map[string]map[uuid.UUID]*Client),
    }
}

// If you want to store it later:
func (h *Hub) SetRedisPubSub(rp *RedisPubSub) {
    h.redisPubSub = rp
}

// Subscribe is unchanged
func (h *Hub) Subscribe(client *Client, channels []string) {
    h.mu.Lock()
    defer h.mu.Unlock()

    for _, ch := range channels {
        if h.channels[ch] == nil {
            h.channels[ch] = make(map[uuid.UUID]*Client)
        }
        h.channels[ch][client.ID] = client
    }
    h.log.Debug("Client subscribed", "client", client.ID, "channels", channels)
}

// Unsubscribe is unchanged
func (h *Hub) Unsubscribe(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    for ch, clientsMap := range h.channels {
        if _, ok := clientsMap[client.ID]; ok {
            delete(clientsMap, client.ID)
            if len(clientsMap) == 0 {
                delete(h.channels, ch)
            }
        }
    }
    h.log.Debug("Client unsubscribed from all channels", "client", client.ID)
}

func (h *Hub) UnsubscribeFromChannel(client *Client, channel string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if clientsMap, ok := h.channels[channel]; ok {
        delete(clientsMap, client.ID)
        if len(clientsMap) == 0 {
            delete(h.channels, channel)
        }
    }
}

// localBroadcast is your old broadcast logic, now internal:
func (h *Hub) localBroadcast(msg Message) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    clientsMap, ok := h.channels[msg.Channel]
    if !ok {
        return
    }
    for _, client := range clientsMap {
        select {
        case client.Outbound <- msg:
        default:
            h.log.Warn("Dropping message to client; outbound buffer full", "client", client.ID, "channel", msg.Channel)
        }
    }
}

// BroadcastGlobal is the main entry point for sending a message to local + remote
func (h *Hub) BroadcastGlobal(ctx context.Context, msg Message) {
    // 1) always do local broadcast
    h.localBroadcast(msg)

    // 2) if we have Redis, also publish so other nodes can broadcast
    if h.redisPubSub != nil {
        if err := h.redisPubSub.Publish(msg); err != nil {
            h.log.Warn("Failed to publish to Redis", "error", err)
        }
    }
}
