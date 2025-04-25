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
}

type myCompanyService struct {
    db            *gorm.DB
    log           *logger.Logger
    warehouseRepo repos.WarehouseRepo
    companyRepo   repos.CompanyRepo
}

func NewMyCompanyService(
    db            *gorm.DB,
    log           *logger.Logger,
    warehouseRepo repos.WarehouseRepo,
    companyRepo   repos.CompanyRepo,
) MyCompanyService {
    serviceLog := log.With("service", "MyCompanyService")
    return &myCompanyService{
        db:             db,
        log:            serviceLog,
        warehouseRepo:  warehouseRepo,
        companyRepo:    companyRepo,
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

