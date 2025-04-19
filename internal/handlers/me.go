package handlers

import (
  "net/http"
  "github.com/gin-gonic/gin"

  "github.com/yungbote/slotter/backend/internal/services"
)

type MeHandler struct {
  meService services.MeService
}

func NewMeHandler(meService services.MeService) *MeHandler {
  return &MeHandler{meService: meService}
}

func (mh *MeHandler) GetMe(c *gin.Context) {
  me, err := mh.meService.GetMe(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"me": me})
}

func (mh *MeHandler) GetMyWms(c *gin.Context) {
  myWms, err := mh.meService.GetMyWms(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myWms": myWms})
}

func (mh *MeHandler) GetMyCompany(c *gin.Context) {
  myCompany, err := mh.meService.GetMyCompany(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myCompany": myCompany})
}

func (mh *MeHandler) GetMyRole(c *gin.Context) {
  myRole, err := mh.meService.GetMyRole(c.Request.Context(), nil)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"myRole": myRole})
}

