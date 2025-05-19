package handlers

import (
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/google/uuid"

  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/services"
  "github.com/slotter-org/slotter-backend/internal/sse"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
)

type AuthHandler struct {
  authService     services.AuthService
  sseHub          *sse.SSEHub
}

func NewAuthHandler(authService services.AuthService, hub *sse.SSEHub) *AuthHandler {
  return &AuthHandler{authService: authService, sseHub: hub}
}

func (ah *AuthHandler) Register(c *gin.Context) {
  var req struct {
    Email           string              `json:"email"`
    PhoneNumber     string              `json:"phone_number,omitempty"`
    FirstName       string              `json:"first_name"`
    LastName        string              `json:"last_name"`
    Password        string              `json:"password"`
    NewWmsName      string              `json:"new_wms_name,omitempty"`
    NewCompanyName  string              `json:"new_company_name,omitempty"`
    CompanyID       string              `json:"company_id,omitempty"`
    WmsID           string              `json:"wms_id,omitempty"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
    return
  }
  user := types.User{
    Email:        req.Email,
    PhoneNumber:  &req.PhoneNumber,
    FirstName:    req.FirstName,
    LastName:     req.LastName,
    Password:     req.Password,
  }
  var newWmsName string
  var newCompanyName string
  if req.NewWmsName != "" {
    newWmsName = req.NewWmsName
    user.UserType = "wms"
  }
  if req.NewCompanyName != "" {
    newCompanyName = req.NewCompanyName
    user.UserType = "company"
  }
  if req.WmsID != "" {
    wmsID, _ := uuid.Parse(req.WmsID)
    user.WmsID = &wmsID
  }
  if req.CompanyID != "" {
    companyID, _ := uuid.Parse(req.CompanyID)
    user.CompanyID = &companyID
  }
  err := ah.authService.RegisterUser(c.Request.Context(), &user, newCompanyName, newWmsName)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"success": true})
}

func (ah *AuthHandler) RegisterWithInvitation(c *gin.Context) {
  var req struct {
    Token           string          `json:"token"`
    Email           string          `json:"email,omitempty"`
    PhoneNumber     string          `json:"phone_number,omitempty"`
    FirstName       string          `json:"first_name"`
    LastName        string          `json:"last_name"`
    Password        string          `json:"password"`
    NewCompanyName  string          `json:"new_company_name,omitempty"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
    return
  }
  if strings.TrimSpace(req.Token) == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "missing invitation token"})
    return
  }
  user := types.User{
    Email: req.Email,
    PhoneNumber: &"",
    FirstName: req.FirstName,
    LastName: req.LastName,
    Password: req.Password,
  }
  ctx := c.Request.Context()
  err := ah.authService.RegisterUserWithInvitationToken(ctx, &user, req.Token, req.NewCompanyName)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  ssd := ssedata.GetSSEData(ctx)
  if ssd != nil && len(ssd.Messages) > 0 {
    for _, msg := range ssd.Messages {
      ah.sseHub.Broadcast(msg)
    }
    ssd.Messages = nil
  }
  c.JSON(http.StatusOK, gin.H{"message": "User successfully registered via invitation"})
}

func (ah *AuthHandler) Login(c *gin.Context) {
  var req struct {
    Email           string          `json:"email"`
    Password        string          `json:"password"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
    return
  }
  accessToken, refreshToken, err := ah.authService.Login(c.Request.Context(), req.Email, req.Password)
  if err != nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
    return
  }
  accessTTL := ah.authService.GetAccessTTL()
  expiresIn := int(accessTTL.Seconds())

  c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken, "expires_in": expiresIn})
}

func (ah *AuthHandler) Refresh(c *gin.Context) {
  accessToken, refreshToken, err := ah.authService.Refresh(c.Request.Context())
  if err != nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
    return
  }
  accessTTL := ah.authService.GetAccessTTL()
  expiresIn := int(accessTTL.Seconds())

  c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken, "expires_in": expiresIn})
}

func (ah *AuthHandler) Logout(c *gin.Context) {
  err :=  ah.authService.Logout(c.Request.Context())
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

