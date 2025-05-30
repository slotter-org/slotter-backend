package services

import (
  "context"
  "fmt"
  
  "github.com/google/uuid"
  "gorm.io/gorm"

  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/types"
)

type MyWmsService interface {
  GetMyCompanies(ctx context.Context, tx *gorm.DB) ([]types.Company, error)
  GetMyCompaniesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Company, error)
  GetMyUsers(ctx context.Context, tx *gorm.DB) ([]types.User, error)
  GetMyUsersWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.User, error)
  GetMyRoles(ctx context.Context, tx *gorm.DB) ([]types.Role, error)
  GetMyRolesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Role, error)
  GetMyInvitations(ctx context.Context, tx *gorm.DB) ([]types.Invitation, error)
  GetMyInvitationsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Invitation, error)
  GetAllPermissions(ctx context.Context, tx *gorm.DB) ([]types.Permission, error)
  GetAllPermissionsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Permission, error)
}

type myWmsService struct {
  db              *gorm.DB
  log             *logger.Logger
  companyRepo     repos.CompanyRepo
  wmsRepo         repos.WmsRepo
  userRepo        repos.UserRepo
  roleRepo        repos.RoleRepo
  invitationRepo  repos.InvitationRepo
  permissionRepo  repos.PermissionRepo
}

func NewMyWmsService(
  db              *gorm.DB,
  log             *logger.Logger,
  companyRepo     repos.CompanyRepo,
  wmsRepo         repos.WmsRepo,
  userRepo        repos.UserRepo,
  roleRepo        repos.RoleRepo,
  invitationRepo  repos.InvitationRepo,
  permissionRepo  repos.PermissionRepo,
) MyWmsService {
  serviceLog := log.With("service", "MyWmsService")
  return &myWmsService{
    db:             db,
    log:            serviceLog,
    companyRepo:    companyRepo,
    wmsRepo:        wmsRepo,
    userRepo:       userRepo,
    roleRepo:       roleRepo,
    invitationRepo: invitationRepo,
    permissionRepo: permissionRepo, 
  }
}

func (ws *myWmsService) GetMyCompanies(ctx context.Context, tx *gorm.DB) ([]types.Company, error) {
  if tx != nil {
    return nil, fmt.Errorf("Please use GetMyCompaniesWithTransaction if you already have a transaction")
  }
  var results []types.Company
  err := ws.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
    comps, cErr := ws.GetMyCompaniesWithTransaction(ctx, innerTx)
    if cErr != nil {
      return cErr
    }
    for _, c := range comps {
      results = append(results, *c)
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  return results, nil
}

func (ws *myWmsService) GetMyCompaniesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Company, error) {
  if tx == nil {
    ws.log.Warn("GetMyCompaniesWithTransaction called with nil transaction")
    return nil, fmt.Errorf("Transaction is required and cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return nil, fmt.Errorf("Request Data not set in context")
  }
  if rd.WmsID == uuid.Nil {
    ws.log.Warn("No WmsID in Request Data. The user might be a company user or missing data.")
    return nil, fmt.Errorf("User does not have a valid WmsID in Request Data")
  }
  comps, err := ws.companyRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{rd.WmsID})
  if err != nil {
    ws.log.Warn("Failed to fetch companies by WmsID", "error", err)
    return nil, err
  }
  if len(comps) == 0 {
    ws.log.Debug("No companies found for the user's Wms", "WmsID", rd.WmsID)
  }
  ws.log.Info("Fetched companies for the user's Wms", "count", len(comps))
  return comps, nil
}

func (ws *myWmsService) GetMyUsers(ctx context.Context, tx *gorm.DB) ([]types.User, error) {
  if tx != nil {
    return nil, fmt.Errorf("Please use GetMyUsersWithTransaction if you already have a transaction")
  }
  var results []types.User
  err := ws.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
    users, uErr := ws.GetMyUsersWithTransaction(ctx, innerTx)
    if uErr != nil {
      return uErr
    }
    for _, u := range users {
      results = append(results, *u)
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  return results, nil
}

func (ws *myWmsService) GetMyUsersWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.User, error) {
  if tx == nil {
    ws.log.Warn("GetMyUsersWithTransaction called with nil transaction")
    return nil, fmt.Errorf("Transaction is required and cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return nil, fmt.Errorf("Request Data not set in context")
  }
  if rd.WmsID == uuid.Nil {
    ws.log.Warn("No WmsID in Request Data. The user might be a company user or missing data.")
    return nil, fmt.Errorf("User does not have a valid WmsID in Request Data")
  }
  users, err := ws.userRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{rd.WmsID})
  if err != nil {
    ws.log.Warn("Failed to fetch users by WmsID", "error", err)
    return nil, err
  }
  if len(users) == 0 {
    ws.log.Debug("No users found for the user's Wms", "WmsID", rd.WmsID)
  }
  ws.log.Info("Fetched companies for the user's Wms", "count", len(users))
  return users, nil
}

func (ws *myWmsService) GetMyRoles(ctx context.Context, tx *gorm.DB) ([]types.Role, error) {
  if tx != nil {
    return nil, fmt.Errorf("Please use GetMyRolesWithTransaction if you already have a transaction")
  }
  var results []types.Role
  err := ws.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
    roles, rErr := ws.GetMyRolesWithTransaction(ctx, innerTx)
    if rErr != nil {
      return rErr
    }
    for _, r := range roles {
      results = append(results, *r)
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  return results, nil
}

func (ws *myWmsService) GetMyRolesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Role, error) {
  if tx == nil {
    ws.log.Warn("GetMyRolesWithTransaction called with nil transaction")
    return nil, fmt.Errorf("Transaction is required and cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return nil, fmt.Errorf("Request Data not set in context")
  }
  if rd.WmsID == uuid.Nil {
    ws.log.Warn("No WmsID in Request Data. The user might be a company user or missing data.")
    return nil, fmt.Errorf("User does not have a valid WmsID in Request Data")
  }
  roles, err := ws.roleRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{rd.WmsID})
  if err != nil {
    ws.log.Warn("Failed to fetch roles by WmsID", "error", err)
    return nil, err
  }
  if len(roles) == 0 {
    ws.log.Debug("No roles found for the users Wms", "WmsID", rd.WmsID)
  }
  ws.log.Info("Fetched roles for the user's Wms", "count", len(roles))
  return roles, nil
}

func (ws *myWmsService) GetMyInvitations(ctx context.Context, tx *gorm.DB) ([]types.Invitation, error) {
  if tx != nil {
    return nil, fmt.Errorf("please use GetMyInvitationsWithTransaction if you already have a transaction")
  }
  var results []types.Invitation
  err := ws.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
    invs, iErr := ws.GetMyInvitationsWithTransaction(ctx, innerTx)
    if iErr != nil {
      return iErr
    }
    for _, i := range invs {
      results = append(results, *i)
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  return results, nil
}

func (ws *myWmsService) GetMyInvitationsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Invitation, error) {
  if tx == nil {
    ws.log.Warn("GetMyInvitationsWithTransaction called with nil transaction")
    return nil, fmt.Errorf("transaction is required and cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("RequestData not set in context.")
    return nil, fmt.Errorf("request data not set in context")
  }
  if rd.WmsID == uuid.Nil {
    ws.log.Warn("No WmsID in Request Data. The user might be a Company user or missing data.")
    return nil, fmt.Errorf("user does not have a valid WmsID in request data")
  }
  invsArr, err := ws.invitationRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{rd.WmsID})
  if err != nil {
    ws.log.Warn("Failed to fetch invitations by WmsID", "error", err)
    return nil, err
  }
  if len(invsArr) == 0 {
    ws.log.Debug("No invitations found for the user's wms", "wmsID", rd.WmsID)
  }
  ws.log.Info("Fetched invitations for the user's wms", "count", len(invsArr))
  return invsArr, nil
}

func (ws *myWmsService) GetAllPermissions(ctx context.Context, tx *gorm.DB) ([]types.Permission, error) {
    if tx != nil {
        return nil, fmt.Errorf("please use GetAllPermissionsWithTransaction if you already have a transaction")
    }
    var results []types.Permission
    err := ws.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        ps, psErr := ws.GetAllPermissionsWithTransaction(ctx, innerTx)
        if psErr != nil {
            return psErr
        }
        for _, p := range ps {
            results = append(results, *p)
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return results, nil
}

func (ws *myWmsService) GetAllPermissionsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Permission, error) {
    if tx == nil {
        ws.log.Warn("GetAllPermissionsWithTransaction was called with nil transaction")
        return nil, fmt.Errorf("transaction is required and cannot be nil")
    }
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        ws.log.Warn("RequestData not set in context.")
        return nil, fmt.Errorf("request data not set in context")
    }
    if rd.WmsID == uuid.Nil {
        ws.log.Warn("No WmsID in Request Data. The user might be a Company user or missing data.")
        return nil, fmt.Errorf("user does not have a valid WmsID in request data")
    }
    ps, err := ws.permissionRepo.GetAll(ctx, tx)
    if err != nil {
        ws.log.Debug("Failed to fetch all permissions for wms", "error", err)
        return nil, err
    }
    if len(ps) == 0 {
        ws.log.Debug("No Permission found for get all permissions for wms", "wmsID", rd.WmsID)
    }
    ws.log.Info("Fetched all permissions for the user's wms", "count", len(ps))
    return ps, nil
}
