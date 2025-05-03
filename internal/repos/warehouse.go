package repos

import (
    "context"
    "strings"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
    
    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type WarehouseRepo interface {
    Create(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) ([]*types.Warehouse, error)
    GetByIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) ([]*types.Warehouse, error)
    GetByCompanyID(ctx context.Context, tx *gorm.DB, companyID uuid.UUID) ([]*types.Warehouse, error)
    NameExistsForCompany(ctx context.Context, tx *gorm.DB, companyID uuid.UUID, warehouseName string) (bool, error)
    Update(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) ([]*types.Warehouse, error)
    SoftDeleteByWarehouses(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) error
    SoftDeleteByWarehouseIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) error
    FullDeleteByWarehouses(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) error
    FullDeleteByWarehouseIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) error
}

type warehouseRepo struct {
    db          *gorm.DB
    log         *logger.Logger
}

func NewWarehouseRepo(db *gorm.DB, baseLog *logger.Logger) WarehouseRepo {
    repoLog := baseLog.With("repo", "WarehouseRepo")
    return &warehouseRepo{db: db, log: repoLog}
}

func (wr *warehouseRepo) Create(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) ([]*types.Warehouse, error) {
    wr.log.Info("Starting Create Warehouses now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    } else {
        wr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if len(warehouses) == 0 {
        wr.log.Debug("No warehouses provided, returning empty slice")
        return []*types.Warehouse{}, nil
    }
    wr.log.Debug("Warehouses provided", "count", len(warehouses))

    wr.log.Info("Creating warehouses now...")
    if err := transaction.WithContext(ctx).Create(&warehouses).Error; err != nil {
        wr.log.Error("Failed to create warehouses", "error", err)
        return nil, err
    }
    wr.log.Info("Successfully create warehouses", "count", len(warehouses))
    wr.log.Debug("Warehouses created", "warehouses", warehouses)
    return warehouses, nil
}

func (wr *warehouseRepo) GetByIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) ([]*types.Warehouse, error) {
    wr.log.Info("Starting GetByIDs for warehouses...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    } else {
        wr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    var results []*types.Warehouse
    if len(warehouseIDs) == 0 {
        wr.log.Debug("No warehouseIDs provided, returning empty slice")
        return results, nil
    }
    wr.log.Debug("WarehouseIDs provided", "count", len(warehouseIDs), "warehouseIDs", warehouseIDs)
    wr.log.Info("Fetching warehouses by IDs now...")
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("id IN ?", warehouseIDs).
        Find(&results).Error; err != nil {
        wr.log.Error("Failed to fetch warehouses by IDs", "error", err)
        return nil, err
    }
    wr.log.Info("Successfully fetch warehouses by IDs", "count", len(results))
    wr.log.Debug("Warehouses fetched", "warehouses", results)
    return results, nil
}

func (wr *warehouseRepo) GetByCompanyID(ctx context.Context, tx *gorm.DB, companyID uuid.UUID) ([]*types.Warehouse, error) {
    wr.log.Info("Starting GetByCompanyID for warehouses...")
    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    } else {
        wr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    var results []*types.Warehouse
    if companyID == uuid.Nil {
        wr.log.Debug("companyID is nil, returning empty slice")
        return results, nil
    }
    wr.log.Debug("companyID provided", "companyID", companyID)
    wr.log.Info("Fetching warehouses by companyID with SELECT FOR UPDATE...")
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("company_id = ?", companyID).
        Find(&results).Error; err != nil {
        wr.log.Error("Failed to fetch (and lock) warehouses by companyID", "error", err)
        return nil, err
    }
    wr.log.Info("Successfully fetched and locked warehouses by companyID", "count", len(results))
    wr.log.Debug("Warehouses locked/fetched", "warehouses", results)
    return results, nil
}

func (wr *warehouseRepo) NameExistsForCompany(ctx context.Context, tx *gorm.DB, companyID uuid.UUID, warehouseName string) (bool, error) {
    wr.log.Info("Checking if Warehouse name exists for the given company...")
    if companyID == uuid.Nil {
        wr.log.Warn("CompanyID is nil, returning false early")
        return false, nil
    }
    nameToCheck := strings.TrimSpace(warehouseName)
    if nameToCheck == "" {
        wr.log.Warn("Warehouse name is empty, skipping check")
        return false, nil
    }
    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Using wr.db because transaction is nil")
    }
    var count int64
    if err := transaction.WithContext(ctx).
        Model(&types.Warehouse{}).
        Where("company_id = ? AND name = ?", companyID, nameToCheck).
        Count(&count).Error; err != nil {
        wr.log.Error("Failed to count warehouses for name check", "error", err)
        return false, err
    }
    wr.log.Debug("NameExistsForCompany completed", "companyID", companyID, "warehouseName", warehouseName, "count", count)
    return count > 0, nil
}

func (wr *warehouseRepo) Update(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) ([]*types.Warehouse, error) {
    wr.log.Info("Starting Update Warehouses now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    } else {
        wr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if len(warehouses) == 0 {
        wr.log.Debug("No warehouses provided, returning empty slice")
        return warehouses, nil
    }
    wr.log.Debug("Updating warehouses", "count", len(warehouses))
    wr.log.Info("Saving warehouses now...")
    for i := range warehouses {
        if err := transaction.WithContext(ctx).Save(&warehouses[i]).Error; err != nil {
            wr.log.Error("Failed to update warehouse", "error", err, "warehouse", warehouses[i])
            return nil, err
        }
    }
    wr.log.Info("Successfully updated warehouses", "count", len(warehouses))
    wr.log.Debug("Warehouses updated", "warehouses", warehouses)
    return warehouses, nil
}

func (wr *warehouseRepo) SoftDeleteByWarehouses(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) error {
    wr.log.Info("Starting SoftDeleteByWarehouses now...")
    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }
    if len(warehouses) == 0 {
        wr.log.Debug("No warehouses provided, skipping soft delete")
        return nil
    }
    wr.log.Debug("Soft deleting warehouses by slice", "count", len(warehouses))
    var warehouseIDs []uuid.UUID
    for _, w := range warehouses {
        warehouseIDs = append(warehouseIDs, w.ID)
    }
    wr.log.Debug("Collected warehouseIDs from slice", "warehouseIDs", warehouseIDs)
    wr.log.Info("Performing soft delete by warehouseIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", warehouseIDs).
        Delete(&types.Warehouse{}).Error; err != nil {
        wr.log.Error("Failed to soft delete warehouses by slice", "error", err)
        return err
    }
    wr.log.Info("Successfully soft deleted warehouses by slice", "count", len(warehouseIDs))
    return nil
}

func (wr *warehouseRepo) SoftDeleteByWarehouseIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) error {
    wr.log.Info("Starting SoftDeleteByWarehouseIDs now...")
    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }
    if len(warehouseIDs) == 0 {
        wr.log.Debug("No warehouseIDs provided, skipping soft delete")
        return nil
    }
    wr.log.Debug("Soft deleting warehouses by IDs", "count", len(warehouseIDs), "warehouseIDs", warehouseIDs)
    wr.log.Info("Performing soft delete by warehouseIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", warehouseIDs).
        Delete(&types.Warehouse{}).Error; err != nil {
        wr.log.Error("Failed to soft delete warehouses by IDs", "error", err)
        return err
    }
    wr.log.Info("Successfully soft deleted warehouses by IDs", "count", len(warehouseIDs))
    return nil
}

func (wr *warehouseRepo) FullDeleteByWarehouses(ctx context.Context, tx *gorm.DB, warehouses []*types.Warehouse) error {
    wr.log.Info("Starting FullDeleteByWarehouses now...")
    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }
    if len(warehouses) == 0 {
        wr.log.Debug("No warehouses provided, skipping full delete")
        return nil
    }
    wr.log.Debug("Full deleting warehouses by slice", "count", len(warehouses))
    var warehouseIDs []uuid.UUID
    for _, w := range warehouses {
        warehouseIDs = append(warehouseIDs, w.ID)
    }
    wr.log.Debug("Collected warehouseIDs from slice", "warehouseIDs", warehouseIDs)
    wr.log.Info("Performiing FULL (hard) delete by warehouseIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", warehouseIDs).
        Delete(&types.Warehouse{}).Error; err != nil {
        wr.log.Error("Failed to FULL delete warehouses by slice", "error", err)
        return err
    }
    wr.log.Info("Successfully FULL deleted warehouses by slice", "count", len(warehouseIDs))
    return nil
}

func (wr *warehouseRepo) FullDeleteByWarehouseIDs(ctx context.Context, tx *gorm.DB, warehouseIDs []uuid.UUID) error {
    wr.log.Info("Starting FullDeleteByWarehouseIDs now...")
    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }
    if len(warehouseIDs) == 0 {
        wr.log.Debug("No warehouseIDs provided, skipping full delete")
        return nil
    }
    wr.log.Debug("Full deleting by warehouseIDs", "count", len(warehouseIDs), "warehouseIDs", warehouseIDs)
    wr.log.Info("Performing FULL (hard) delete by warehouseIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", warehouseIDs).
        Delete(&types.Warehouse{}).Error; err != nil {
        wr.log.Error("Failed to FULL delete warehouses by IDs", "error", err)
        return err
    }
    wr.log.Info("Successfully FULL deleted warehouses by IDs", "count", len(warehouseIDs))
    return nil
}






