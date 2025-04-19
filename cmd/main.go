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
  avatarService, err := services.NewAvatarService(thePG, log, wmsRepo, companyRepo, userRepo, roleRepo, permissionRepo, bucketService)
  if err != nil {
    log.Error("Fatal error: Cannot init AvatarService", "error", err)
    os.Exit(1)
  }
  authService := services.NewAuthService(thePG, log, userRepo, wmsRepo, companyRepo, roleRepo, permissionRepo, avatarService, userTokenRepo, jwtSecretKey, time.Duration(accessTokenTTL)*time.Second, time.Duration(refreshTokenTTL)*time.Second)
  meService := services.NewMeService(thePG, log, userRepo, wmsRepo, companyRepo, roleRepo)
  invitationService := services.NewInvitationService(thePG, log, invitationRepo, userRepo, wmsRepo, companyRepo, roleRepo, permissionRepo, textService, emailService)
  log.Info("Services Set Up From Main Successful :)")


  //  Handler Setup
  log.Info("Setting Up Handlers from Main now...")
  authHandler := handlers.NewAuthHandler(authService)
  meHandler := handlers.NewMeHandler(meService)
  invitationHandler := handlers.NewInvitationHandler(invitationService)
  wsHandler := handlers.WsHandler(wsHub, log)
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
    InvitationHandler:      invitationHandler,
    WsHandler:              wsHandler,
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
