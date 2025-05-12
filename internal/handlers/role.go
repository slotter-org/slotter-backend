package handlers

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/services"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
  "github.com/slotter-org/slotter-backend/internal/sse"
  "github.com/slotter-org/slotter-backend/internal/errordata"
)

type RoleHandler struct {
  roleService         services.RoleService
  sseHub              *sse.SSEHub
}

func NewRoleHandler(roleService services.RoleService, hub *sse.SSEHub) *RoleHandler {
  return &RoleHandler{roleService: roleService, sseHub: hub}
}

type RoleCreateRequest struct {
  Name            string            `json:"name"`
  Description     string            `json:"description,omitempty"`
}

func (rh *RoleHandler) CreateRole(c *gin.Context) {
  ctx := c.Request.Context()
  var req RoleCreateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  _, nrErr := rh.roleService.CreateLoggedInWithEntity(ctx, nil, req.Name, req.Description)
  if nrErr != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": nrErr.Error()})
    return
  }
  errData := errordata.GetErrorData(ctx)
  if errData != nil && errData.HasMessage() {
    c.JSON(http.StatusBadRequest, gin.H{"error": errData.Message})
    return
  }
  ssd := ssedata.GetSSEData(ctx)
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      rh.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Role created successfully"})
}

type RoleUpdateRequest struct {
  RoleID          string              `json:"role_id"`
  Name            string              `json: "name,omitempty"`
  Description     string              `json:"description,omitempty"`
  Permissions     []types.Permission  `json:"permissions,omitempty"`
}

func (rh *RoleHandler) UpdateRole(c *gin.Context) {
  ctx := c.Request.Context()
  var req RoleUpdateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  roleUUID, err := uuid.Parse(req.RoleID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roleId parameter"})
    return
  }
  updatedRole, ruErr := rh.roleService.UpdateRole(ctx, nil, roleUUID, req.Name, req.Description, req.Permissions)
  if ruErr != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": ruErr.Error()})
    return
  }
  errData := errordata.GetErrorData(ctx)
  if errData != nil && errData.HasMessage() {
    c.JSON(http.StatusBadRequest, gin.H{"error": errData.Message})
    return
  }
  ssd := ssedata.GetSSEData(ctx)
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      rh.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}
