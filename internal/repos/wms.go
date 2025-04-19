package repos

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type WmsRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.Wms, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Wms, error)

    // UPDATE
    Update(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.Wms, error)

    // SOFT DELETE
    SoftDeleteByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) error
    SoftDeleteByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) error
    FullDeleteByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) error
}

type wmsRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewWmsRepo(db *gorm.DB, baseLog *logger.Logger) WmsRepo {
    repoLog := baseLog.With("repo", "WmsRepo")
    return &wmsRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------

func (wr *wmsRepo) Create(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.Wms, error) {
    wr.log.Info("Starting Create Wms now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    } else {
        wr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    if len(wmss) == 0 {
        wr.log.Debug("No Wms records provided, returning empty slice")
        return []*types.Wms{}, nil
    }
    wr.log.Debug("Wms provided", "count", len(wmss))

    wr.log.Info("Creating Wms now...")
    if err := transaction.WithContext(ctx).Create(&wmss).Error; err != nil {
        wr.log.Error("Failed to create Wms records", "error", err)
        return nil, err
    }
    wr.log.Info("Successfully created Wms records", "count", len(wmss))
    wr.log.Debug("Wms created", "wmss", wmss)
    return wmss, nil
}

// ----------------------------------------------------------------
// READ
// ----------------------------------------------------------------

func (wr *wmsRepo) GetByIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Wms, error) {
    wr.log.Info("Starting GetByIDs for Wms...")

    // Use passed-in transaction if non-nil, otherwise default to wr.db
    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    }

    var results []*types.Wms
    if len(wmsIDs) == 0 {
        wr.log.Debug("No wmsIDs provided, returning empty slice")
        return results, nil
    }
    wr.log.Debug("wmsIDs provided", "count", len(wmsIDs), "wmsIDs", wmsIDs)

    wr.log.Info("Fetching Wms by IDs with SELECT FOR UPDATE...")

    // The critical part: add .Clauses(clause.Locking{Strength: "UPDATE"})
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}). // <<-- Lock rows
        Preload("Companies").
        Preload("Users").
        Where("id IN ?", wmsIDs).
        Find(&results).
        Error; err != nil {
        wr.log.Error("Failed to fetch (and lock) Wms by IDs", "error", err)
        return nil, err
    }

    wr.log.Info("Successfully fetched and locked Wms by IDs", "count", len(results))
    wr.log.Debug("Wms locked/fetched", "wmss", results)

    return results, nil
}


// ----------------------------------------------------------------
// UPDATE
// ----------------------------------------------------------------

func (wr *wmsRepo) Update(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.Wms, error) {
    wr.log.Info("Starting Update Wms now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
        wr.log.Debug("Transaction is nil, using wr.db", "db", transaction)
    }

    if len(wmss) == 0 {
        wr.log.Debug("No Wms records provided, returning empty slice")
        return wmss, nil
    }
    wr.log.Debug("Updating Wms", "count", len(wmss))

    wr.log.Info("Saving Wms now...")
    for i := range wmss {
        if err := transaction.WithContext(ctx).Save(&wmss[i]).Error; err != nil {
            wr.log.Error("Failed to update Wms record", "error", err, "wms", wmss[i])
            return nil, err
        }
    }
    wr.log.Info("Successfully updated Wms records", "count", len(wmss))
    wr.log.Debug("Wms updated", "wmss", wmss)
    return wmss, nil
}

// ----------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------

func (wr *wmsRepo) SoftDeleteByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) error {
    wr.log.Info("Starting SoftDeleteByWmss now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }

    if len(wmss) == 0 {
        wr.log.Debug("No Wms records provided, skipping soft delete")
        return nil
    }
    wr.log.Debug("Soft deleting Wms by slice", "count", len(wmss))

    var wmsIDs []uuid.UUID
    for _, w := range wmss {
        wmsIDs = append(wmsIDs, w.ID)
    }
    wr.log.Debug("Collected wmsIDs from slice", "wmsIDs", wmsIDs)

    wr.log.Info("Performing soft delete by wmsIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", wmsIDs).
        Delete(&types.Wms{}).Error; err != nil {
        wr.log.Error("Failed to soft delete Wms by slice", "error", err)
        return err
    }
    wr.log.Info("Successfully soft deleted Wms by slice", "count", len(wmsIDs))
    return nil
}

func (wr *wmsRepo) SoftDeleteByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) error {
    wr.log.Info("Starting SoftDeleteByWmsIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }

    if len(wmsIDs) == 0 {
        wr.log.Debug("No wmsIDs provided, skipping soft delete")
        return nil
    }
    wr.log.Debug("Soft deleting Wms by wmsIDs", "count", len(wmsIDs), "wmsIDs", wmsIDs)

    wr.log.Info("Performing soft delete by wmsIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", wmsIDs).
        Delete(&types.Wms{}).Error; err != nil {
        wr.log.Error("Failed to soft delete Wms by IDs", "error", err)
        return err
    }
    wr.log.Info("Successfully soft deleted Wms by IDs", "count", len(wmsIDs))
    return nil
}

// ----------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------

func (wr *wmsRepo) FullDeleteByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) error {
    wr.log.Info("Starting FullDeleteByWmss now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }

    if len(wmss) == 0 {
        wr.log.Debug("No Wms records provided, skipping full delete")
        return nil
    }
    wr.log.Debug("Full deleting Wms by slice", "count", len(wmss))

    var wmsIDs []uuid.UUID
    for _, w := range wmss {
        wmsIDs = append(wmsIDs, w.ID)
    }
    wr.log.Debug("Collected wmsIDs from slice", "wmsIDs", wmsIDs)

    wr.log.Info("Performing FULL (hard) delete by wmsIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", wmsIDs).
        Delete(&types.Wms{}).Error; err != nil {
        wr.log.Error("Failed to FULL delete Wms by slice", "error", err)
        return err
    }
    wr.log.Info("Successfully FULL deleted Wms by slice", "count", len(wmsIDs))
    return nil
}

func (wr *wmsRepo) FullDeleteByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) error {
    wr.log.Info("Starting FullDeleteByWmsIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = wr.db
    }

    if len(wmsIDs) == 0 {
        wr.log.Debug("No wmsIDs provided, skipping full delete")
        return nil
    }
    wr.log.Debug("Full deleting by wmsIDs", "count", len(wmsIDs), "wmsIDs", wmsIDs)

    wr.log.Info("Performing FULL (hard) delete by wmsIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", wmsIDs).
        Delete(&types.Wms{}).Error; err != nil {
        wr.log.Error("Failed to FULL delete Wms by IDs", "error", err)
        return err
    }
    wr.log.Info("Successfully FULL deleted Wms by IDs", "count", len(wmsIDs))
    return nil
}

