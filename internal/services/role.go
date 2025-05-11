package services

import (
  "context"
  "fmt"

  "gorm.io/gorm"

  "github.com/google/uuid"
  
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/normalization"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/ssedata"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/sse"
)

type RoleService interface {
  Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error)
  CreateLoggedInWithEntity(ctx context.Context, tx *gorm.DB, name string, description string) (types.Role, error)
  createLoggedIn(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (types.Role, error)
  createLoggedInWithTransaction(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (*types.Role, error)
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

func (rs *roleService) CreateLoggedInWithEntity(ctx context.Context, tx *gorm.DB, name string, description string) (types.Role, error) {
  rs.log.Info("Starting CreateLoggedInWithEntity now...")
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    rs.log.Warn("Request Data is not set in context")
    return types.Role{}, fmt.Errorf("request data not set in context")
  }
  var entityType string
  if rd.UserType == "company" {
    entityType = "company"
  } else {
    entityType = "wms"
  }
  role, err := rs.createLoggedIn(ctx, tx, entityType, name, description)
  if err != nil {
    return types.Role{}, err
  }
  var channel string
  switch rd.UserType {
  case "company":
    if rd.CompanyID != uuid.Nil {
      channel = "company:" + rd.CompanyID.String()
    }
  case "wms":
    if rd.WmsID != uuid.Nil {
      channel = "wms:" + rd.WmsID.String()
    }
  }
  if channel != "" {
    sseData := ssedata.GetSSEData(ctx)
    if sseData != nil {
      sseData.AppendMessage(sse.SSEMessage{
        Channel: channel,
        Event: sse.SSEEventRoleCreated,
      })
    }
  }
  return role, nil
}

func (rs *roleService) createLoggedIn(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (types.Role, error) {
  rs.log.Info("Starting createLoggedIn now...")
  if tx != nil {
    rs.log.Warn("Please use createLoggedInWithTransaction if you have a transaction")
    return types.Role{}, fmt.Errorf("transaction passed into non-transaction handling function")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    rs.log.Warn("Request Data is not set in context")
    return types.Role{}, fmt.Errorf("request data is not set in context")
  }
  nName := normalization.ParseInputString(name)
  if nName == "" {
    rs.log.Warn("No name provided to createLoggedIn")
    return types.Role{}, fmt.Errorf("need a non-empty name to create role")
  }
  if entityType == "company" && rd.UserType != "company" {
    rs.log.Warn("UserType mismatch. Expected company but got something else.")
    return types.Role{}, fmt.Errorf("UserType mismatch: can't create company role as a non company user type")
  }
  if entityType == "wms" && rd.UserType != "wms" {
    rs.log.Warn("UserType mismatch. Expected wms but got something else.")
    return types.Role{}, fmt.Errorf("UserType mismatch: can't create wms role as a non wms user type")
  }
  var result types.Role
  err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
    rolePtr, rErr := rs.createLoggedInWithTransaction(ctx, innerTx, entityType, nName, description)
    if rErr != nil {
      return rErr
    }
    if rolePtr == nil {
      rs.log.Warn("createLoggedInWithTransaction returned nil role ptr")
      return fmt.Errorf("nil role ptr returned from createLoggedInWithTransaction")
    }
    result = *rolePtr
    return nil
  })
  if err != nil {
    return types.Role{}, err
  }
  return result, nil
}

func (rs *roleService) createLoggedInWithTransaction(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (*types.Role, error) {
  rs.log.Info("Starting createLoggedInWithTransaction now...")
  if tx == nil {
    rs.log.Warn("createLoggedInWithTransaction called with a nil transaction")
    return nil, fmt.Errorf("transaction is required and cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    rs.log.Warn("RequestData not set in context")
    return nil, fmt.Errorf("request data not set in context")
  }
  var entityID uuid.UUID
  switch entityType {
  case "company":
    if rd.CompanyID == uuid.Nil {
      rs.log.Warn("No CompanyID in RequestData")
      return nil, fmt.Errorf("user does not have a valid company id in request data")
    }
    entityID = rd.CompanyID
  case "wms":
    if rd.WmsID == uuid.Nil {
      rs.log.Warn("No WmsID in RequestData")
      return nil, fmt.Errorf("user does not have a valid wms id in request data")
    }
    entityID = rd.WmsID
  default:
    rs.log.Warn("Unsupported entityType", "entityType", entityType)
    return nil, fmt.Errorf("unsupported entityType: %s", entityType)
  }
  var exists bool
  var err error
  if entityType == "company" {
    exists, err := rs.roleRepo.NameExistsByCompanyID(ctx, tx, entityID, name)
  } else {
    exists, err = rs.roleRepo.NameExistsByWmsID(ctx, tx, entityID, name)
  }
  if err != nil {
    rs.log.Warn("Failure checking new role name existence", "err", err)
    return nil, fmt.Errorf("failed to check role name existence: %w", err)
  }
  if exists {
    rs.log.Warn("Role name already exists for entity", "entityType", entityType, "name", name)
    return nil, fmt.Errorf("role name %q already exists for %s", name, entityType)
  }
  newRole := &types.Role{
    Name: name,
    Description: &description,
  }
  if entityType == "company" {
    newRole.CompanyID = &entityID
  } else {
    newRole.WmsID = &entityID
  }
  newRoles, nrErr := rd.Create(ctx, tx, []*types.Role{newRole})
  if nrErr != nil {
    rs.log.Warn("Failure to create new role", "error", nrErr)
    return nil, fmt.Errorf("failed to create new role: %w", nrErr)
  }
  if len(newRoles) == 0 {
    rs.log.Warn("Creating role returned no items", "count", len(newRoles))
    return nil, fmt.Errorf("creating role returned no roles")
  }
  finalRole := newRoles[0]
  return finalRole, nil
}
