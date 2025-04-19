package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"

  "github.com/yungbote/slotter/backend/internal/types"
  "github.com/yungbote/slotter/backend/internal/services"
)

type InvitationHandler struct {
  invitationService       services.InvitationService
}

func NewInvitationHandler(invitationService services.InvitationService) *InvitationHandler {
  return &InvitationHandler{invitationService: invitationService}
}

type InvitationSendRequest struct {
  Email             string                `json:"email,omitemtpy"`
  PhoneNumber       string                `json:"phone_number,omitempty"`
  InvitationType    types.InvitationType  `json:"invitation_type,omitempty"`
}

func (ih *InvitationHandler) SendInvitation(c *gin.Context) {
  var req InvitationSendRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  invitation := &types.Invitation{
    Email:              &req.Email,
    PhoneNumber:        &req.PhoneNumber,
    InvitationType:     req.InvitationType,
  }
  if err := ih.invitationService.SendInvitation(c.Request.Context(), invitation); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation sent successfully"})
}
