package handlers

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "github.com/slotter-org/slotter-backend/internal/services"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
  "github.com/slotter-org/slotter-backend/internal/sse"
)

type RoleHandler struct {
  roleService         services.RoleService
  sseHub              *sse.SSEHub
}

func NewRoleHandler(roleService services.RoleService, hub *sse.SSEHub) *RoleHandler {
  return &RoleHander{roleService: roleService, sseHub: hub}
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
  newRole, nrErr := rh.roleService.CreatedLoggedInWithEntity(ctx, nil, req.Name, req.Description)
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
