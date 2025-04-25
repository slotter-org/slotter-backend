package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"
  "github.com/gorilla/websocket"
  
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/socket"
)

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func WsHandler(hub *socket.Hub, log *logger.Logger) gin.HandlerFunc {
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
