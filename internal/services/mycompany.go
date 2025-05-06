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

type MyCompanyService interface {
    GetMyWarehouses(ctx context.Context, tx *gorm.DB) ([]types.Warehouse, error)
    GetMyWarehousesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Warehouse, error)
    GetMyUsers(ctx context.Context, tx *gorm.DB) ([]types.User, error)
    GetMyUsersWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.User, error)
    GetMyRoles(ctx context.Context, tx *gorm.DB) ([]types.Role, error)
    GetMyRolesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Role, error)
    GetMyInvitations(ctx context.Context, tx *gorm.DB) ([]types.Invitation, error)
    GetMyInvitationsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Invitation, error)
}

type myCompanyService struct {
    db              *gorm.DB
    log             *logger.Logger
    warehouseRepo   repos.WarehouseRepo
    companyRepo     repos.CompanyRepo
    userRepo        repos.UserRepo
    roleRepo        repos.RoleRepo
    invitationRepo  repos.InvitationRepo
}

func NewMyCompanyService(
    db              *gorm.DB,
    log             *logger.Logger,
    warehouseRepo   repos.WarehouseRepo,
    companyRepo     repos.CompanyRepo,
    userRepo        repos.UserRepo,
    roleRepo        repos.RoleRepo,
    invitationRepo  repos.InvitationRepo,
) MyCompanyService {
    serviceLog := log.With("service", "MyCompanyService")
    return &myCompanyService{
        db:             db,
        log:            serviceLog,
        warehouseRepo:  warehouseRepo,
        companyRepo:    companyRepo,
        userRepo:       userRepo,
        roleRepo:       roleRepo,
        invitationRepo: invitationRepo,
    }
}

// ----------------------------------------------------------------------------
// GetMyWarehouses
// ----------------------------------------------------------------------------

func (cs *myCompanyService) GetMyWarehouses(ctx context.Context, tx *gorm.DB) ([]types.Warehouse, error) {
    if tx != nil {
        return nil, fmt.Errorf("please use GetMyWarehousesWithTransaction if you already have a transaction")
    }

    var results []types.Warehouse
    err := cs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        whs, whErr := cs.GetMyWarehousesWithTransaction(ctx, innerTx)
        if whErr != nil {
            return whErr
        }
        for _, w := range whs {
            results = append(results, *w)
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return results, nil
}

// ----------------------------------------------------------------------------
// GetMyWarehousesWithTransaction
// ----------------------------------------------------------------------------

func (cs *myCompanyService) GetMyWarehousesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Warehouse, error) {
    if tx == nil {
        cs.log.Warn("GetMyWarehousesWithTransaction was called with nil transaction")
        return nil, fmt.Errorf("transaction is required and cannot be nil")
    }

    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        cs.log.Warn("RequestData not set in context.")
        return nil, fmt.Errorf("request data not set in context")
    }
    if rd.CompanyID == uuid.Nil {
        cs.log.Warn("No CompanyID in Request Data. The user might be a Wms user or missing data.")
        return nil, fmt.Errorf("user does not have a valid CompanyID in request data")
    }
    whs, err := cs.warehouseRepo.GetByCompanyID(ctx, tx, rd.CompanyID)
    if err != nil {
        cs.log.Warn("Failed to fetch warehouses by CompanyID", "error", err)
        return nil, err
    }
    if len(whs) == 0 {
        cs.log.Debug("No Warehouses found for the user's company", "companyID", rd.CompanyID)
    }
    cs.log.Info("Fetched warehouses for the user's company", "count", len(whs))
    return whs, nil
}

func (cs *myCompanyService) GetMyUsers(ctx context.Context, tx *gorm.DB) ([]types.User, error) {
    if tx != nil {
        return nil, fmt.Errorf("Please use GetMyUsersWithTransaction if you already have a transaction")
    }
    var results []types.User
    err := cs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        users, uErr := cs.GetMyUsersWithTransaction(ctx, innerTx)
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

func (cs *myCompanyService) GetMyUsersWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.User, error) {
    if tx == nil {
        cs.log.Warn("GetMyUsersWithTransaction called with nil transaction")
        return nil, fmt.Errorf("Transaction is required and cannot be nil")
    }
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        cs.log.Warn("Request Data is not set in context.")
        return nil, fmt.Errorf("Request Data not set in context")
    }
    if rd.CompanyID == uuid.Nil {
        cs.log.Warn("No CompanyID in Request Data. The user might be a wms user or missing data.")
        return nil, fmt.Errorf("User does not have a valid CompanyID in Request Data")
    }
    users, err := cs.userRepo.GetByCompanyIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
    if err != nil {
        cs.log.Warn("Failed to fetch users by CompanyID", "error", err)
        return nil, err
    }
    if len(users) == 0 {
        cs.log.Debug("No users found for the users company", "CompanyID", rd.CompanyID)
    }
    cs.log.Info("Fetched users for the user's Company", "count", len(users))
    return users, nil
}

func (cs *myCompanyService) GetMyRoles(ctx context.Context, tx *gorm.DB) ([]types.Role, error) {
    if tx != nil {
        return nil, fmt.Errorf("Please use GetMyRolesWithTransaction if you already have a transaction")
    }
    var results []types.Role
    err := cs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        roles, rErr := cs.GetMyRolesWithTransaction(ctx, innerTx)
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

func (cs *myCompanyService) GetMyRolesWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Role, error) {
    if tx == nil {
        cs.log.Warn("GetMyRolesWithTransaction called with nil transaction")
        return nil, fmt.Errorf("Transaction is required and cannot be nil")
    }
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        cs.log.Warn("Request Data is not set in context.")
        return nil, fmt.Errorf("Request Data not set in context")
    }
    if rd.CompanyID == uuid.Nil {
        cs.log.Warn("No CompanyID in Request Data. The user might be a wms user or missing data.")
        return nil, fmt.Errorf("User does not have a valid CompanyID in Request Data")
    }
    roles, err := cs.roleRepo.GetByCompanyIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
    if err != nil {
        cs.log.Warn("Failed to fetch roles by CompanyID", "error", err)
        return nil, err
    }
    if len(roles) == 0 {
        cs.log.Debug("No roles found for the users company", "CompanyID", rd.CompanyID)
    }
    cs.log.Info("Fetched roles for the users Company", "count", len(roles))
    return roles, nil
}

func (cs *myCompanyService) GetMyInvitations(ctx context.Context, tx *gorm.DB) ([]types.Invitation, error) {
    if tx != nil {
        return nil, fmt.Errorf("please use GetMyInvitationsWithTransaction if you already have a transaction")
    }
    var results []types.Invitation
    err := cs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        invs, iErr := cs.GetMyInvitationsWithTransaction(ctx, innerTx)
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

func (cs *myCompanyService) GetMyInvitationsWithTransaction(ctx context.Context, tx *gorm.DB) ([]*types.Invitation, error) {
    if tx == nil {
        cs.log.Warn("GetMyInvitationsWithTransaction was called with nil transaction")
        return nil, fmt.Errorf("transaction is required and cannot be nil")
    }
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        cs.log.Warn("RequestData not set in context")
        return nil, fmt.Errorf("request data not set in context")
    }
    if rd.CompanyID == uuid.Nil {
        cs.log.Warn("No CompanyID in RequestData. The user might be a Wms user or missing data.")
        return nil, fmt.Errorf("user does not have a valid companyID in request data")
    }
    invsArr, err := cs.invitationRepo.GetByCompanyIDs(ctx, tx,  []uuid.UUID{rd.CompanyID})
    if err != nil {
        cs.log.Warn("Failed to fetch invitations by company IDs", "error", err)
        return nil, err
    }
    if len(invsArr) == 0 {
        cs.log.Debug("No invitations found for the user's company", "companyID", rd.CompanyID)
    }
    cs.log.Info("Fetched invitations for the user's company", "count", len(invsArr))
    return invsArr, nil
}


