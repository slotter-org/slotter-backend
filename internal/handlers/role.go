package handlers

import (
  "net/http"

  "github.com/gin-gonic/gin"
  "github.com/google/uuid"
  
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/services"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
  "github.com/slotter-org/slotter-backend/internal/sse"
  "github.com/slotter-org/slotter-backend/internal/errordata"
)

type RoleHandler struct {
  roleService     services.RoleService
  sseHub          *sse.SSEHub
}

func NewRoleHandler(roleService services.RoleService, hub *sse.SSEHub) *RoleHandler {
  return &RoleHandler{roleService: roleService, sseHub: hub}
}

//-------------------------------------------------------
// CREATE
//-------------------------------------------------------

type RoleCreateRequest struct {
  Name            string          `json:"name"`
  Description     string          `json:"description"`
}

func (rh *RoleHandler) CreateRole(c *gin.Context) {
  ctx := c.Request.Context()
  var req RoleCreateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  _, createErr := rh.roleService.CreateLoggedIn(ctx, nil, req.Name, req.Description)
  if createErr != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": createErr.Error()})
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

//----------------------------------------------------------
// UPDATE ROLE (name/description)
//----------------------------------------------------------

type RoleNameDescUpdateRequest struct {
  RoleID          string          `json:"role_id"`
  Name            string          `json:"name,omitempty"`
  Description     string          `json:"description,omitempty"`
}

func (rh *RoleHandler) UpdateRoleNameDesc(c *gin.Context) {
  ctx := c.Request.Context()
  var req RoleNameDescUpdateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  if req.RoleID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "role_id is required"})
    return
  }
  roleUUID, err := uuid.Parse(req.RoleID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
    return
  }
  _, upErr := rh.roleService.UpdateRole(ctx, nil, roleUUID, req.Name, req.Description)
  if upErr != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": upErr.Error()})
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
  c.JSON(http.StatusOK, gin.H{"message": "Role name/description updated successfully"})
}

//-----------------------------------------------------------------------------
// UPDATE PERMISSIONS
//-----------------------------------------------------------------------------

type RolePermissionsUpdateRequest struct {
  RoleID            string              `json:"role_id"`
  Permissions       []types.Permission  `json:"permissions,omitempty"`
}

func (rh *RoleHandler) UpdateRolePermissions(c *gin.Context) {
  ctx := c.Request.Context()
  var req RolePermissionsUpdateRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  if req.RoleID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "role_id is required"})
    return
  }
  roleUUID, err := uuid.Parse(req.RoleID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
    return
  }
  _, upErr := rh.roleService.UpdatePermissions(ctx, nil, roleUUID, req.Permissions)
  if upErr != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": upErr.Error()})
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
  c.JSON(http.StatusOK, gin.H{"message": "Role permissions updated successfully"})
}

//--------------------------------------------------------------------------------------
// DELETE
//--------------------------------------------------------------------------------------

type RoleDeleteRequest struct {
  RoleID        string          `json:"role_id"`
}

func (rh *RoleHandler) DeleteRole(c *gin.Context) {
  ctx := c.Request.Context()
  var req RoleDeleteRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
    return
  }
  if req.RoleID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "role_id is required"})
    return
  }
  roleUUID, err := uuid.Parse(req.RoleID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
    return
  }
  delErr := rh.roleService.DeleteRole(ctx, nil, roleUUID)
  if delErr != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": delErr.Error()})
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
  c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}
