package handlers

import (
  "encoding/json"
  "net/http"
  "sync"
  
  "github.com/gin-gonic/gin"
  "github.com/google/uuid"
  
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/sse"
)

type SSEHandler struct {
  Log           *logger.Logger
  Hub           *sse.SSEHub
  mu            sync.RWMutex
  userMap       map[uuid.UUID]*sse.SSEClient
}

func NewSSEHandler(log *logger.Logger, hub *sse.SSEHub) *SSEHandler {
  return &SSEHandler{
    Log:      log,
    Hub:      hub,
    userMap:  make(map[uuid.UUID]*sse.SSEClient),
  }
}

func (h *SSEHandler) SSEStream(c *gin.Context) {
  rd := requestdata.GetRequestData(c.Request.Context())
  if rd == nil || rd.UserID == uuid.Nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
    return
  }
  userID := rd.UserID

  h.mu.Lock()
  client, ok := h.userMap[userID]
  if ok {
    h.Hub.CloseClient(client)
    delete(h.userMap, userID)
  }
  client = h.Hub.NewSSEClient(userID)
  client.ID = uuid.New()
  client.Logger = h.Log.With("SSEClientID", client.ID)
  h.userMap[userID] = client
  h.mu.Unlock()

  h.Hub.ServeHTTP(c.Writer, c.Request, client)

  h.mu.Lock()
  delete(h.userMap, userID)
  h.mu.Unlock()
  h.Hub.CloseClient(client)
}

func (h *SSEHandler) SSESubscribe(c *gin.Context) {
  rd := requestdata.GetRequestData(c.Request.Context())
  if rd == nil || rd.UserID == uuid.Nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
    return
  }
  userID := rd.UserID

  var req struct {
    Channel       string      `json:"channel"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.Channel == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel"})
    return
  }
  h.mu.RLock()
  client, exists := h.userMap[userID]
  h.mu.RUnlock()
  if !exists {
    c.JSON(http.StatusConflict, gin.H{"error": "no active SSE connection for this user"})
    return
  }
  h.Hub.AddChannel(client, req.Channel)
  c.JSON(http.StatusOK, gin.H{"message": "subscribed", "channel": req.Channel})
}

func (h *SSEHandler) SSEUnsubscribe(c *gin.Context) {
  rd := requestdata.GetRequestData(c.Request.Context())
  if rd == nil || rd.UserID == uuid.Nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
    return
  }
  userID := rd.UserID

  var req struct {
    Channel         string      `json:"channel"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.Channel == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel"})
    return
  }
  h.mu.RLock()
  client, exists := h.userMap[userID]
  h.mu.RUnlock()
  if !exists {
    c.JSON(http.StatusConflict, gin.H{"error": "no active SSE connection for this user"})
    return
  }
  h.Hub.RemoveChannel(client, req.Channel)
  c.JSON(http.StatusOK, gin.H{"message": "unsubscribed", "channel": req.Channel})
}
