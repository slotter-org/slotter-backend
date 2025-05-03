package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"
  "github.com/gorilla/websocket"
  
  "github.com/slotter-org/slotter-backend/internal/middleware"
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/socket"
)

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func WsHandlerr(hub *socket.Hub, log *logger.Logger) gin.HandlerFunc {
  return func(c *gin.Context) {
    ctx := c.Request.Context()
    rd := requestdata.GetRequestData(ctx)
    if rd == nil || rd.UserID == [16]byte{} {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
      return
    }
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
      log.Warn("Failed to upgrade to websocket", "error", err)
      return
    }
    client := socket.NewClient(conn, hub, log)
    go client.Run(ctx)
  }
}

func WsHandler(hub *socket.Hub, log *logger.Logger) gin.HandlerFunc {
  return func(c *gin.Context) {
    if err := middleware.requireAuth(c); err != nil {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
      return
    }
    reqData := requestdata.GetRequestData(c.Request.Context())
    if reqData == nil || reqData.UserID == uuid.Nil {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
      return
    }
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
      log.Warn("websocket upgrade failed", "error", err)
      return
    }
    ctx, cancel := context.WithCancel(context.Background())
    client := socket.NewClient(
      hub,
      reqData.UserID,
      conn,
      cancel,
      log,
    )
    hub.Register <- client

    go client.ReadPump(ctx)
    go client.WritePump(ctx)
  }
}
