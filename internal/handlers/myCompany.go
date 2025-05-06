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

func (mch *MyCompanyHandler) GetMyUsers(c *gin.Context) {
  myUsers, err := mch.myCompanyService.GetMyUsers(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myUsers": myUsers})
}

func (mch *MyCompanyHandler) GetMyRoles(c *gin.Context) {
  myRoles, err := mch.myCompanyService.GetMyRoles(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myRoles": myRoles})
}

func (mch *MyCompanyHandler) GetMyInvitations(c *gin.Context) {
  myInvs, err := mch.myCompanyService.GetMyInvitations(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myInvitations": myInvs})
}

