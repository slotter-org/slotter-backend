package handlers

import (
  "net/http"
  "github.com/gin-gonic/gin"
)

func Healthz(c *gin.Context) {
  c.String(http.StatusOK, "ok")
}

