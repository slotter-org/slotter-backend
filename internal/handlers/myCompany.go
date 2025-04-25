package handlers

import (
  "net/http"
  "github.com/gin-gonic/gin"
  
  "github.com/slotter-org/slotter-backend/internal/services"
)

type MyCompanyHandler struct {
  myCompanyService services.MyCompanyService
}

func NewMyCompanyHandler(myCompanyService services.MyCompanyService) *MyCompanyHandler {
  return &MyCompanyHandler{myCompanyService: myCompanyService}
}

func (mch *MyCompanyHandler) GetMyWarehouses(c *gin.Context) {
  myWarehouses, err := mch.myCompanyService.GetMyWarehouses(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myWarehouses": myWarehouses})
}

