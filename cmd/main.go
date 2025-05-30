package main

import (
  "fmt"
  "os"
  "time"
  
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/utils"
  "github.com/slotter-org/slotter-backend/internal/db"
  "github.com/slotter-org/slotter-backend/internal/seed"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/services"
  "github.com/slotter-org/slotter-backend/internal/socket"
  "github.com/slotter-org/slotter-backend/internal/handlers"
  "github.com/slotter-org/slotter-backend/internal/middleware"
  "github.com/slotter-org/slotter-backend/internal/server"
  "github.com/slotter-org/slotter-backend/internal/sse"
)

func main() {
  // Logger Setup
  logMode := os.Getenv("LOG_MODE")
  if logMode == "" {
    logMode = "development"
  }
  log, err := logger.New(logMode)
  if err != nil {
    fmt.Printf("failed to init logger: %v\n", err)
    os.Exit(1)
  }
  defer log.Sync()


  // Environment Variables
  log.Info("Attempting to load environment variables for Main now...")
  jwtSecretKey := utils.GetEnv("JWT_SECRET_KEY", "defaultsecret", log)
  accessTokenTTL := utils.GetEnvAsInt("ACCESS_TOKEN_TTL", 3600, log)
  refreshTokenTTL := utils.GetEnvAsInt("REFRESH_TOKEN_TTL", 86400, log)
  redisAddress := utils.GetEnv("REDIS_ADDRESS", "localhost:6379", log)
  redisPassword := utils.GetEnv("REDIS_PASSWORD", "", log)
  log.Debug("Environment variables loaded for Main :)",
    "jwtSecretKey", jwtSecretKey,
    "accessTokenTTL", accessTokenTTL,
    "refreshTokenTTL", refreshTokenTTL,
    "redisAddress", redisAddress,
    "redisPassword", redisPassword,
  )

  // Postgres Setup
  log.Info("Setting Up Postgres from Main now...")
  postgresService, err := db.NewPostgresService(log)
  if err != nil {
    log.Warn("DB init failed", "error", err)
  }
  if err = postgresService.AutoMigrateAll(); err != nil {
    log.Warn("Postgres auto migration failed", "error", err)
  }
  thePG := postgresService.DB()
  log.Info("Postgres Setup From Main Successful :)")


  // Repositories Setup
  log.Info("Setting Up Repositories from Main now...")
  wmsRepo := repos.NewWmsRepo(thePG, log)
  companyRepo := repos.NewCompanyRepo(thePG, log)
  warehouseRepo := repos.NewWarehouseRepo(thePG, log)
  userRepo := repos.NewUserRepo(thePG, log)
  permissionRepo := repos.NewPermissionRepo(thePG, log)
  roleRepo := repos.NewRoleRepo(thePG, log)
  userTokenRepo := repos.NewUserTokenRepo(thePG, log)
  invitationRepo := repos.NewInvitationRepo(thePG, log)
  log.Info("Repositories Set Up From Main Successful :)")

  // Seed Setup
  log.Info("Attempting to Seed The Postgres From Main now...")
  if err := seed.SeedAll(thePG, permissionRepo, roleRepo); err != nil {
    log.Warn("Failed to seed data :(", "error", err)
  }
  log.Info("Seeding of Postgres From Main Successful :)")

  // Websocket Setup
  log.Info("Setting Up Websocket Hub From Main Now :)")
  wsHub := socket.NewHub(log)
  log.Info("Websocket Hub Set Up From Main Successful :)")

  // SSE Hub
  log.Info("Setting Up SSE Hub From Main Now :)")
  sseHub := sse.NewSSEHub(log)
  log.Info("SSE Hub Set Up From Main Successful :)")

  // Redis PubSub
  log.Info("Setting Up Redis PubSub From Main Now :)")
  redisChanName := "slotter_hub_broadcast"
  redisPubSub, err := socket.NewRedisPubSub(log, redisAddress, redisPassword, redisChanName)
  if err != nil {
    log.Warn("Failed to init redis pubsub", "error", err)
  } else {
    if err := redisPubSub.StartSubscriber(wsHub); err != nil {
      log.Warn("Failed to subscribe to Redis pub/sub", "error", err)
    } else {
      wsHub.SetRedisPubSub(redisPubSub)
      log.Info("Redis pubsub is active!")
    }
  }
  log.Info("Successfully Set up Redis Pub Sub From Main :)")

  // Services Setup
  log.Info("Setting up Services from Main now...")
  emailService, err := services.NewEmailService(log)
  if err != nil {
    log.Warn("Could not init EmailService", "error", err)
  }
  textService, err := services.NewTextService(log)
  if err != nil {
    log.Warn("Could not init TextService", "error", err)
  }
  bucketService, err := services.NewBucketService(log)
  if err != nil {
    log.Warn("Could not init BucketService", "error", err)
  }
  avatarService, err := services.NewAvatarService(thePG, log, wmsRepo, companyRepo, warehouseRepo, userRepo, roleRepo, permissionRepo, bucketService)
  if err != nil {
    log.Error("Fatal error: Cannot init AvatarService", "error", err)
    os.Exit(1)
  }
  roleService := services.NewRoleService(thePG, log, roleRepo, permissionRepo, userRepo, avatarService)
  authService := services.NewAuthService(thePG, log, userRepo, wmsRepo, companyRepo, roleRepo, roleService, permissionRepo, invitationRepo, avatarService, userTokenRepo, jwtSecretKey, time.Duration(accessTokenTTL)*time.Second, time.Duration(refreshTokenTTL)*time.Second)
  meService := services.NewMeService(thePG, log, userRepo, wmsRepo, companyRepo, roleRepo)
  myCompanyService := services.NewMyCompanyService(thePG, log, warehouseRepo, companyRepo, userRepo, roleRepo, invitationRepo, permissionRepo)
  myWmsService := services.NewMyWmsService(thePG, log, companyRepo, wmsRepo, userRepo, roleRepo, invitationRepo, permissionRepo)
  invitationService := services.NewInvitationService(thePG, log, invitationRepo, userRepo, wmsRepo, companyRepo, roleRepo, permissionRepo, textService, emailService, avatarService)
  warehouseService := services.NewWarehouseService(thePG, log, userRepo, wmsRepo, companyRepo, roleRepo, permissionRepo, warehouseRepo)
  log.Info("Services Set Up From Main Successful :)")


  //  Handler Setup
  log.Info("Setting Up Handlers from Main now...")
  authHandler := handlers.NewAuthHandler(authService, sseHub)
  meHandler := handlers.NewMeHandler(meService)
  myCompanyHandler := handlers.NewMyCompanyHandler(myCompanyService)
  myWmsHandler := handlers.NewMyWmsHandler(myWmsService)
  invitationHandler := handlers.NewInvitationHandler(invitationService, sseHub)
  warehouseHandler := handlers.NewWarehouseHandler(warehouseService, wsHub)
  roleHandler := handlers.NewRoleHandler(roleService, sseHub)
  wsHandler := handlers.WsHandler(wsHub, log)
  sseHandler := handlers.NewSSEHandler(log, sseHub)
  log.Info("Handlers Set Up From Main Successful :)")

  // MiddleWare Setup
  log.Info("Setting Up Middleware from Main now...")
  authMiddleware := middleware.NewAuthMiddleware(log, authService, roleRepo)
  log.Info("Middleware Set Up From Main Successful :)")

  // Router Setup
  log.Info("Setting Up Router from Main now...")
  router := server.NewRouter(server.RouterConfig{
    AuthHandler:            authHandler,
    AuthMiddleware:         authMiddleware,
    MeHandler:              meHandler,
    MyCompanyHandler:       myCompanyHandler,
    MyWmsHandler:           myWmsHandler,
    InvitationHandler:      invitationHandler,
    WarehouseHandler:       warehouseHandler,
    WsHandler:              wsHandler,
    SSEHandler:             sseHandler,
    RoleHandler:            roleHandler,
  })
  log.Info("Router Set Up From Main Successful :)")

  port := utils.GetEnv("PORT", "8080", log)
  fmt.Printf("Server listening on :%s\n", port)
  if err := router.Run(":" + port); err != nil {
    log.Warn("Server failed: %v", err)
  }

  // On Shutdown
  if redisPubSub != nil {
    redisPubSub.Stop()
  }
}
