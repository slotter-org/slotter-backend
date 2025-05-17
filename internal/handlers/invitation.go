package handlers

import (
  "net/http"
  
  "github.com/gin-gonic/gin"
  "github.com/google/uuid"
  
  "github.com/slotter-org/slotter-backend/internal/sse"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/services"
)

type InvitationHandler struct {
  invitationService       services.InvitationService
  sseHub                  *sse.SSEHub
}

func NewInvitationHandler(invitationService services.InvitationService, hub *sse.SSEHub) *InvitationHandler {
  return &InvitationHandler{invitationService: invitationService, sseHub: hub}
}

type InvitationSendRequest struct {
  Email             string                `json:"email,omitemtpy"`
  PhoneNumber       string                `json:"phone_number,omitempty"`
  InvitationType    types.InvitationType  `json:"invitation_type,omitempty"`
  Name              string                `json:"name,omitempty"`
  Message           string                `json:"message,omitempty"`
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
    Name:               &req.Name,
    Message:            &req.Message,
  }
  if err := ih.invitationService.SendInvitation(c.Request.Context(), nil, invitation); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
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
  if req.InvitationID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invitation_id is required"})
    return
  }
  invUUID, err := uuid.Parse(req.InvitationID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation_id format"})
    return
  }
  _, updateErr := ih.invitationService.UpdateInvitation(
    c.Request.Context(),
    nil,
    invUUID,
    req.Name,
    req.Message,
  )
  if updateErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": updateErr.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation updated successfully"})
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
  if req.InvitationID == "" || req.RoleID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invitation_id and role_id are required"})
  }
  invUUID, err := uuid.Parse(req.InvitationID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation_id format"})
  }
  roleUUID, rErr := uuid.Parse(req.RoleID)
  if rErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_id format"})
    return
  }
  _, updateErr := ih.invitationService.UpdateInvitationRole(
    c.Request.Context(),
    nil,
    invUUID,
    roleUUID,
  )
  if updateErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": updateErr.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation role updated successfully"})
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
  if req.InvitationID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invitation_id is required"})
    return
  }
  invUUID, err := uuid.Parse(req.InvitationID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation_id format"})
    return
  }
  _, cancelErr := ih.invitationService.CancelInvitation(
    c.Request.Context(),
    nil,
    invUUID,
  )
  if cancelErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": cancelErr.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation canceled successfully"})
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
  if req.InvitationID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invitation_id is required"})
    return
  }
  invUUID, err := uuid.Parse(req.InvitationID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation_id format"})
    return
  }
  _, reErr := ih.invitationService.ResendInvitation(
    c.Request.Context(),
    nil,
    invUUID,
  )
  if reErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": reErr.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
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
  if req.InvitationID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invitation_id is required"})
    return
  }
  invUUID, err := uuid.Parse(req.InvitationID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation_id format"})
    return
  }
  delErr := ih.invitationService.DeleteInvitation(
    c.Request.Context(),
    nil,
    invUUID,
  )
  if delErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": delErr.Error()})
    return
  }
  ssd := ssedata.GetSSEData(c.Request.Context())
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ih.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Invitation deleted successfully"})
}

type InvitationTokenValidateRequest struct {
  Token             string            `json:"token"`
}

func (ih *InvitationHandler) ValidateInvitationToken(c *gin.Context) {
  var req InvitationTokenValidateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  if req.Token == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
    return
  }
  invitation, err := ih.invitationService.ValidateInvitationToken(
    c.Request.Context(),
    nil,
    req.Token,
  )
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"invitation": invitation})
}
