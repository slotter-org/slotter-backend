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

func (mwh *MyWmsHandler) GetMyUsers(c *gin.Context) {
  myUsers, err := mwh.myWmsService.GetMyUsers(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myUsers": myUsers})
}

func (mwh *MyWmsHandler) GetMyRoles(c *gin.Context) {
  myRoles, err := mwh.myWmsService.GetMyRoles(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myRoles": myRoles})
}

func (mwh *MyWmsHandler) GetMyInvitations(c *gin.Context) {
  myInvs, err := mwh.myWmsService.GetMyInvitations(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myInvitations": myInvs})
}

func (mwh *MyWmsHandler) GetMyPermissions(c *gin.Context) {
  myPerms, err := mwh.myWmsService.GetAllPermissions(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myPermissions": myPerms})
}
