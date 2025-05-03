package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/slotter-org/slotter-backend/internal/services"
    "github.com/slotter-org/slotter-backend/internal/socket"
)

type WarehouseHandler struct {
    warehouseService services.WarehouseService
    hub              *socket.Hub
}

func NewWarehouseHandler(warehouseService services.WarehouseService, hub *socket.Hub) *WarehouseHandler {
    return &WarehouseHandler{warehouseService: warehouseService, hub: hub}
}

func (wh *WarehouseHandler) CreateWarehouse(c *gin.Context) {
    var req struct {
        Name      string `json:"name"`
        CompanyID string `json:"company_id,omitempty"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    var companyUUID uuid.UUID
    if req.CompanyID != "" {
        parsed, parseErr := uuid.Parse(req.CompanyID)
        if parseErr != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid company_id UUID"})
            return
        }
        companyUUID = parsed
    }

    warehouse, err := wh.warehouseService.CreateWarehouse(
        c.Request.Context(),
        req.Name,
        companyUUID,
    )
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if warehouse.CompanyID != uuid.Nil {
        wh.hub.BroadcastGlobal(c.Request.Context(), socket.Message{
            Channel: "company:" + warehouse.CompanyID.String(),
            Data: map[string]interface{}{
                "action": "warehouse_created",
                "payload": warehouse,
            },
        })
    }
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "warehouse": warehouse,
    })
}
