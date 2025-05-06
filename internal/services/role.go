package services

import (
  "context"
  "fmt"

  "gorm.io/gorm"
  
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/repos"
)

type RoleService interface {
  Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error)
}

type roleService struct {
  db              *gorm.DB
  log             *logger.Logger
  roleRepo        repos.RoleRepo
  avatarService   AvatarService
}

func NewRoleService(db *gorm.DB, baseLog *logger.Logger, roleRepo repos.RoleRepo, avatarService AvatarService) RoleService {
  serviceLog := baseLog.With("service", "RoleService")
  return &roleService{db: db, log: serviceLog, roleRepo: roleRepo, avatarService: avatarService}
}

func (rs *roleService) Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error) {
  rs.log.Info("Starting Create Roles now...")
  rs.log.Info("Checking if transaction is nil...")
  if tx == nil {
    rs.log.Error("Transaction cannot be nil")
    return nil, fmt.Errorf("transaction cannot be nil")
  }

  createdRoles, err := rs.roleRepo.Create(ctx, tx, roles)
  if err != nil {
    rs.log.Error("failed to create roles", "error", err)
    return nil, fmt.Errorf("failed to create roles: %w", err)
  }

  var toUpdateRoles []*types.Role
  for i, role := range createdRoles {
    rs.log.Info(fmt.Sprintf("Uploading avatar for role #%d (ID=%s)", i, role.ID))
    updatedRole, err := rs.avatarService.CreateAndUploadRoleAvatar(ctx, tx, role)
    if err != nil {
      rs.log.Error("avatar upload failed", "roleID", "error", err)
      return nil, fmt.Errorf("failed to create/upload avatar for role %s: %w", role.ID, err)
    }
    toUpdateRoles = append(toUpdateRoles, updatedRole)
  }
  
  updatedRoles, err := rs.roleRepo.Update(ctx, tx, toUpdateRoles)
  if err != nil {
    rs.log.Error("Failed to update roles with avatar details", "error", err)
    return nil, fmt.Errorf("failed to update roles with avatar details: %w", err)
  }

  rs.log.Info("All role avatars created and upload successfully")
  return updatedRoles, nil
}
