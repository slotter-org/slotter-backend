package middleware

import (
  "encoding/json"
  "net/http"
  "strings"
  
  "github.com/gin-gonic/gin"
  "github.com/google/uuid"

  "github.com/slotter-org/slotter-backend/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/services"
)

type AuthMiddleware struct {
  log               *logger.Logger
  authService       services.AuthService
  roleRepo          repos.RoleRepo
}

func NewAuthMiddleware(log *logger.Logger, authService services.AuthService, roleRepo repos.RoleRepo) *AuthMiddleware {
  middlewareLogger := log.With("Middleware", "AuthMiddleware")
  return &AuthMiddleware{log: middlewareLogger, authService: authService, roleRepo: roleRepo}
}

func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
  return func(c *gin.Context) {
    tokenString := extractTokenFromAll(c)
    am.log.Debug("TokenString:", "tokenstring", tokenString)
    if tokenString == "" {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
      return
    }
    ctx, err := am.authService.SetContextFromToken(c.Request.Context(), tokenString)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
      return
    }
    c.Request = c.Request.WithContext(ctx)
    rd := requestdata.GetRequestData(ctx)
    if rd == nil || rd.UserID == uuid.Nil {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden - invalid user id"})
      return
    } 
    c.Next()
  }
}

func (am *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
  return func(c *gin.Context) {
    am.RequireAuth()(c)
    if c.IsAborted() {
      return
    }
    ctx := c.Request.Context()
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "request data missing"})
      return
    }
    if rd.RoleID == uuid.Nil {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no role id in request data"})
      return
    }
    roles, err := am.roleRepo.GetByIDs(ctx, nil, []uuid.UUID{rd.RoleID})
    if err != nil {
      c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load role"})
      return
    }
    if len(roles) == 0 {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found"})
      return
    }
    role := roles[0]
    hasPermission := false
    for _, pm := range role.Permissions {
      if pm.Name == permission || pm.PermissionType == permission {
        hasPermission = true
        break
      }
    }
    if !hasPermission {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
      return
    }
    c.Next()
  }
}

func extractTokenFromAll(c *gin.Context) string {
  if qToken := c.Query("token"); qToken != "" {
    return qToken
  }
  authHeader := c.GetHeader("Authorization")
  if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
    return authHeader[7:]
  }
  var body struct {
    Token       string      `json:"token"`
  }
  if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
    if body.Token != "" {
      return body.Token
    }
  }
  return ""
}


