package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"

  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/services"
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

type InvitationUpdateRequest struct {
  InvitationID            string              `json:"invitation_id"`
  Message                 string              `json:"message,omitempty"`
  Name                    string              `json:"name,omitempty"`
}

func (ih *InvitationHandler) UpdateInvitationMsgName(c *gin.Context) {
  var req InvitationUpdateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }

}

type InvitationUpdateRoleRequest struct {
  InvitationID            string              `json:"invitation_id"`
  RoleID                  string              `json:"role_id"`
}

func (ih *InvitationHandler) UpdateInvitationRole(c *gin.Context) {
  var req InvitationUpdateRoleRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
}

type InvitationCancelRequest struct {
  InvitationID          string                `json:"invitation_id"`
}

func (ih *InvitationHandler) CancelInvitation(c *gin.Context) {
  var req InvitationCancelRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
}

type InvitationResendRequest struct {
  InvitationID          string                `json:"invitation_id"`
}

func (ih *InvitationHandler) ResendInvitation(c *gin.Context) {
  var req InvitationResendRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
}

type InvitationDeleteRequest struct {
  InvitationID          string              `json:"invitation_id"`
}

func (ih *InvitationHandler) DeleteInvitation(c *gin.Context) {
  var req InvitationDeleteRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
}
