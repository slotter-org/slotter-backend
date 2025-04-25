package handlers

import (
  "net/http"
  "github.com/gin-gonic/gin"

  "github.com/slotter-org/slotter-backend/internal/services"
)

type MyWmsHandler struct {
  myWmsService services.MyWmsService
}

func NewMyWmsHandler(myWmsService services.MyWmsService) *MyWmsHandler {
  return &MyWmsHandler{myWmsService: myWmsService}
}

func (mwh *MyWmsHandler) GetMyCompanies(c *gin.Context) {
  myCompanies, err := mwh.myWmsService.GetMyCompanies(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myCompanies": myCompanies})
}


