package db

import (
  "fmt"
  
  "gorm.io/driver/postgres"
  "gorm.io/gorm"

  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/utils"
  "github.com/slotter-org/slotter-backend/internal/logger"
)

type PostgresService struct {
  db *gorm.DB
  log *logger.Logger
}

func NewPostgresService(log *logger.Logger) (*PostgresService, error) {
  serviceLog := log.With("service", "PostgresService")

  //1) Get and Set Environment Variables
  log.Info("Attempting to load environment variables for Postgres now...")
  postgresHost := utils.GetEnv("POSTGRES_HOST", "localhost", log)
  postgresPort := utils.GetEnv("POSTGRES_PORT", "5432", log)
  postgresUser := utils.GetEnv("POSTGRES_USER", "postgres", log)
  postgresPassword := utils.GetEnv("POSTGRES_PASSWORD", "", log)
  postgresName := utils.GetEnv("POSTGRES_NAME", "slotter", log)
  log.Debug("Environment variables loaded for Postgres",
    "host", postgresHost,
    "port", postgresPort,
    "user", postgresUser,
    "password", postgresPassword,
    "dbname", postgresName,
  )
  log.Info("Environment variables loaded for Postgres :)")

  //2) Construct DSN From Environment Variables
  log.Info("Attempting to construct DSN from environment variables for Postgres now...")
  dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresName)
  log.Debug("Postgres DSN built :)", "dsn", dsn)

  //3) Attempt DB Connection
  log.Info("Attempting to connect to Postgres DB now...")
  db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    DisableForeignKeyConstraintWhenMigrating: true,
  })
  if err != nil {
    log.Error("Failed to connect to Postgres DB", "error", err)
    return nil, fmt.Errorf("Failed to connect to Postgres DB: %w", err)
  }
  log.Debug("Successfully Connected to Postgres DB :)", "db", db)
  log.Info("Successfully Connected to Postgres DB :)")
  
  //4) Enable uuid-ossp Extension
  log.Debug("Attempting to enable uuid-ossp extension now...")
  if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
    log.Error("Failed to enable uuid-ossp extension :(", "error", err)
    return nil, fmt.Errorf("failed to enable uuid-ossp extension: %w", err)
  }
  log.Info("uuid-ossp extension enabled or already exists :)")

  return &PostgresService{db: db, log: serviceLog}, nil
}

func (s *PostgresService) AutoMigrateAll() error {
  s.log.Info("Starting AutoMigrateAll for all GORM models now...")
  
  err := s.db.AutoMigrate(
    &types.User{},
    &types.Role{},
    &types.Company{},
    &types.Wms{},
    &types.Permission{},
    &types.OneTimeCode{},
    &types.UserToken{},
    &types.Invitation{},
    &types.ChatSession{},
    &types.ChatMessage{},
  )
  if err != nil {
    s.log.Error("AutoMigrateAll failed for Base Tables :(", "error", err)
    return err
  }
  s.log.Info("AutoMigrateAll completed successfully for Base Tables :)")


  s.log.Info("Configuring Foreign Key Relationships for Base Tables now...")
  // -- Wms.default_role_id => role.id (ON DELETE SET NULL)
  if err := s.db.Exec(`
    ALTER TABLE "wms"
    ADD CONSTRAINT "fk_wms_default_role"
    FOREIGN KEY ("default_role_id")
    REFERENCES "role"("id")
    ON DELETE SET NULL
  `).Error; err != nil {
    return fmt.Errorf("failed to add fk_wms_default_role: %w", err)
  }
  // -- Company.wms_id => wms.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "company"
      ADD CONSTRAINT "fk_company_wms_id"
      FOREIGN KEY ("wms_id")
      REFERENCES "wms"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_company_wms_id: %w", err)
  }
  // -- Company.default_role_id => role.id (ON DELETE SET NULL)
  if err := s.db.Exec(`
      ALTER TABLE "company"
      ADD CONSTRAINT "fk_company_default_role"
      FOREIGN KEY ("default_role_id")
      REFERENCES "role"("id")
      ON DELETE SET NULL
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_company_default_role: %w", err)
  }
  // -- OneTimeCode.user_id => user.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "one_time_code"
      ADD CONSTRAINT "fk_one_time_code_user_id"
      FOREIGN KEY ("user_id")
      REFERENCES "user"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_one_time_code_user_id: %w", err)
  }
  // -- Role.wms_id => wms.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "role"
      ADD CONSTRAINT "fk_role_wms_id"
      FOREIGN KEY ("wms_id")
      REFERENCES "wms"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_role_wms_id: %w", err)
  }
  // -- Role.company_id => company.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "role"
      ADD CONSTRAINT "fk_role_company_id"
      FOREIGN KEY ("company_id")
      REFERENCES "company"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_role_company_id: %w", err)
  }
  // -- User.wms_id => wms.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "user"
      ADD CONSTRAINT "fk_user_wms_id"
      FOREIGN KEY ("wms_id")
      REFERENCES "wms"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_user_wms_id: %w", err)
  }
  // -- User.company_id => company.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "user"
      ADD CONSTRAINT "fk_user_company_id"
      FOREIGN KEY ("company_id")
      REFERENCES "company"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_user_company_id: %w", err)
  }
  // -- User.role_id => role.id (ON DELETE SET NULL)
  if err := s.db.Exec(`
      ALTER TABLE "user"
      ADD CONSTRAINT "fk_user_role_id"
      FOREIGN KEY ("role_id")
      REFERENCES "role"("id")
      ON DELETE SET NULL
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_user_role_id: %w", err)
  }
  // -- UserToken.user_id => user.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "user_token"
      ADD CONSTRAINT "fk_user_token_user_id"
      FOREIGN KEY ("user_id")
      REFERENCES "user"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_user_token_user_id: %w", err)
  }
  // 3) Add constraints to the many-to-many pivot table that GORM auto-creates for `permissions_roles`.
  //    That pivot table is named "permissions_roles" by default (alphabetical).
  //    If you want FKs on it, do:
  if err := s.db.Exec(`
      ALTER TABLE "permissions_roles"
      ADD CONSTRAINT "fk_permissions_roles_role_id"
      FOREIGN KEY ("role_id")
      REFERENCES "role"("id")
      ON DELETE CASCADE,
      ADD CONSTRAINT "fk_permissions_roles_permission_id"
      FOREIGN KEY ("permission_id")
      REFERENCES "permission"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      // If you never inserted any data, GORM might not even have created the join table yet
      // but typically it does. If you get an error, ensure the table name matches your actual pivot table.
      return fmt.Errorf("failed to add FK constraints to permissions_roles pivot: %w", err)
  }

  // -- Invitation.invite_user_id => user.id (ON DELETE SET NULL)
  if err := s.db.Exec(`
      ALTER TABLE "invitation"
      ADD CONSTRAINT "fk_invitation_invite_user_id"
      FOREIGN KEY ("invite_user_id")
      REFERENCES "user"("id")
      ON DELETE SET NULL
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_invitation_invite_user_id: %w", err)
  }
  // -- Invitation.wms_id => wms.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "invitation"
      ADD CONSTRAINT "fk_invitation_wms_id"
      FOREIGN KEY ("wms_id")
      REFERENCES "wms"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_invitation_wms_id: %w", err)
  }
  // -- Invitation.company_id => company.id (ON DELETE CASCADE)
  if err := s.db.Exec(`
      ALTER TABLE "invitation"
      ADD CONSTRAINT "fk_invitation_company_id"
      FOREIGN KEY ("company_id")
      REFERENCES "company"("id")
      ON DELETE CASCADE
  `).Error; err != nil {
      return fmt.Errorf("failed to add fk_invitation_company_id: %w", err)
  }
  // -- ChatSession
  if err := s.db.Exec(`
      ALTER TABLE "chat_session"
      ADD CONSTRAINT "fk_chat_session_user_id"
      FOREIGN KEY ("user_id")
      REFERENCES "user" ("id")
      ON DELETE CASCADE
  `).Error; err != nil {
    return fmt.Errorf("failed to add fk_chat_session_user_id: %w", err)
  }
  // -- ChatMessage
  if err := s.db.Exec(`
      ALTER TABLE "chat_message"
      ADD CONSTRAINT "fk_chat_message_session_id"
      FOREIGN KEY ("session_id")
      REFERENCES "chat_session" ("id")
      ON DELETE CASCADE
  `).Error; err != nil {
    return fmt.Errorf("failed to add fk_chat_message_session_id: %w", err)
  }
  s.log.Info("Successfully Added Foreign Key Relationships to Base Tables :)")

  return nil
}

func (s *PostgresService) DB() *gorm.DB {
  return s.db
}
