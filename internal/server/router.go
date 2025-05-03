package server

import (
  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/cors"

  "github.com/slotter-org/slotter-backend/internal/handlers"
  "github.com/slotter-org/slotter-backend/internal/middleware"
)

type RouterConfig struct {
  AuthHandler           *handlers.AuthHandler
  AuthMiddleware        *middleware.AuthMiddleware
  MeHandler             *handlers.MeHandler
  MyCompanyHandler      *handlers.MyCompanyHandler
  MyWmsHandler          *handlers.MyWmsHandler
  InvitationHandler     *handlers.InvitationHandler
  WsHandler             gin.HandlerFunc
  WarehouseHandler      *handlers.WarehouseHandler
}

func NewRouter(cfg RouterConfig) *gin.Engine {
  router := gin.Default()
  
  //-----------------------------------------
  // Cors Setup
  //-----------------------------------------
  router.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "http://localhost:3000",
        "http://slotter.ai",
        "http://www.slotter.ai",
        "https://www.slotter.ai",
        "https://slotter.ai",      // prod
        "https://www.slotter.ai",  // optional www
    },
    AllowMethods:     []string{"GET","POST","PUT","DELETE","PATCH","OPTIONS"},
    AllowHeaders:     []string{"Authorization","Content-Type","X-Requested-With"},
    AllowCredentials: true,
}))

  //-----------------------------------------
  // Health Routes
  //-----------------------------------------
  router.GET("/healthz", handlers.Healthz)

  //-----------------------------------------
  // Public Routes
  //-----------------------------------------
  api := router.Group("/api")
  {
    api.POST("/register", cfg.AuthHandler.Register)
    api.POST("/login", cfg.AuthHandler.Login)
  }


  //------------------------------------------
  // Protected Routes
  //------------------------------------------
  protected := api.Group("/")
  protected.Use(cfg.AuthMiddleware.RequireAuth())
  protected.POST("/refresh", cfg.AuthHandler.Refresh)
  protected.POST("/logout", cfg.AuthHandler.Logout)
  protected.GET("/ws", cfg.WsHandler)

  //ME
  protected.GET("/me", cfg.MeHandler.GetMe)
  protected.GET("/mywms", cfg.MeHandler.GetMyWms)
  protected.GET("/mycompany", cfg.MeHandler.GetMyCompany)
  protected.GET("/myroles", cfg.MeHandler.GetMyRole)

  //MyCompany/MyWms
  protected.GET("/mycompany/warehouses", cfg.MyCompanyHandler.GetMyWarehouses)
  protected.GET("/mycompany/users", cfg.MyCompanyHandler.GetMyUsers)
  protected.GET("/mycompany/roles", cfg.MyCompanyHandler.GetMyRoles)
  protected.GET("/mywms/companies", cfg.MyWmsHandler.GetMyCompanies)
  protected.GET("/mywms/users", cfg.MyWmsHandler.GetMyUsers)
  protected.GET("/mywms/roles", cfg.MyWmsHandler.GetMyRoles)

  //Warehouse
  protected.POST("/warehouse", cfg.WarehouseHandler.CreateWarehouse)

  //Invitations
  invitations := protected.Group("/invitations")
  invitations.Use(cfg.AuthMiddleware.RequirePermission("manage_invitations"))
  invitations.POST("/", cfg.InvitationHandler.SendInvitation)

  return router
}
