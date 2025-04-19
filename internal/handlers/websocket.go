package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"
  "github.com/gorilla/websocket"
  
  "github.com/yungbote/slotter/backend/internal/logger"
  "github.com/yungbote/slotter/backend/internal/requestdata"
  "github.com/yungbote/slotter/backend/internal/socket"
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

    var channels []string

    userChan := "user:" + rd.UserID.String()
    channels = append(channels, userChan)

    if rd.CompanyID != [16]byte{} {
      compChan := "company:" + rd.CompanyID.String()
      channels = append(channels, compChan)
    }
    if rd.WmsID != [16]byte{} {
      wmsChan := "wms:" + rd.WmsID.String()
      channels = append(channels, wmsChan)
    }
    hub.Subscribe(client, channels)
    
    go client.Run(ctx)
  }
}
