package repos

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type CompanyRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.Company, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.Company, error)

    GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Company, error)

    // UPDATE
    Update(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.Company, error)

    // SOFT DELETE
    SoftDeleteByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) error
    SoftDeleteByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) error
    FullDeleteByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) error
}

type companyRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewCompanyRepo(db *gorm.DB, baseLog *logger.Logger) CompanyRepo {
    repoLog := baseLog.With("repo", "CompanyRepo")
    return &companyRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------

func (cr *companyRepo) Create(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.Company, error) {
    cr.log.Info("Starting Create Companies now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
        cr.log.Debug("Transaction is nil, using cr.db", "db", transaction)
    } else {
        cr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    if len(companies) == 0 {
        cr.log.Debug("No companies provided, returning empty slice")
        return []*types.Company{}, nil
    }
    cr.log.Debug("Companies provided", "count", len(companies))

    cr.log.Info("Creating companies now...")
    if err := transaction.WithContext(ctx).Create(&companies).Error; err != nil {
        cr.log.Error("Failed to create companies", "error", err)
        return nil, err
    }
    cr.log.Info("Successfully created companies", "count", len(companies))
    cr.log.Debug("Companies created", "companies", companies)
    return companies, nil
}

// ----------------------------------------------------------------
// READ
// ----------------------------------------------------------------

func (cr *companyRepo) GetByIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.Company, error) {
    cr.log.Info("Starting GetByIDs for companies...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
        cr.log.Debug("Transaction is nil, using cr.db", "db", transaction)
    } else {
        cr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    var results []*types.Company
    if len(companyIDs) == 0 {
        cr.log.Debug("No companyIDs provided, returning empty slice")
        return results, nil
    }
    cr.log.Debug("CompanyIDs provided", "count", len(companyIDs), "companyIDs", companyIDs)

    cr.log.Info("Fetching companies by IDs now...")
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Preload("Users").
        Where("id IN ?", companyIDs).
        Find(&results).Error; err != nil {
        cr.log.Error("Failed to fetch companies by IDs", "error", err)
        return nil, err
    }
    cr.log.Info("Successfully fetched companies by IDs", "count", len(results))
    cr.log.Debug("Companies fetched", "companies", results)
    return results, nil
}

func (cr *companyRepo) GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Company, error) {
    cr.log.Info("Starting Get Companies By Wms IDs now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
        cr.log.Debug("Transaction is nil, using cr.db", "db", transaction)
    } else {
        cr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    var results []*types.Company
    if len(wmsIDs) == 0 {
        cr.log.Debug("No wmsIDs provided, returning empty slice")
        return results, nil
    }
    cr.log.Debug("wmsIDs provided", "count", len(wmsIDs), "wmsIDs", wmsIDs)
    cr.log.Info("Fetching companies by Wms IDs now...")
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Preload("Users").
        Where("wms_id IN ?", wmsIDs).
        Find(&results).
        Error; err != nil {
        cr.log.Error("Failed to fetch companies by Wms IDs", "error", err)
        return nil, err
    }
    cr.log.Info("Successfully fetched companies by Wms IDs", "count", len(results))
    cr.log.Debug("Companies fetched", "companies", results)
    return results, nil
}

// ----------------------------------------------------------------
// UPDATE
// ----------------------------------------------------------------

func (cr *companyRepo) Update(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.Company, error) {
    cr.log.Info("Starting Update Companies now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
        cr.log.Debug("Transaction is nil, using cr.db", "db", transaction)
    } else {
        cr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    if len(companies) == 0 {
        cr.log.Debug("No companies provided, returning empty slice")
        return companies, nil
    }
    cr.log.Debug("Updating companies", "count", len(companies))

    cr.log.Info("Saving companies now...")
    for i := range companies {
        if err := transaction.WithContext(ctx).Save(&companies[i]).Error; err != nil {
            cr.log.Error("Failed to update company", "error", err, "company", companies[i])
            return nil, err
        }
    }
    cr.log.Info("Successfully updated companies", "count", len(companies))
    cr.log.Debug("Companies updated", "companies", companies)
    return companies, nil
}

// ----------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------

func (cr *companyRepo) SoftDeleteByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) error {
    cr.log.Info("Starting SoftDeleteByCompanies now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
    }

    if len(companies) == 0 {
        cr.log.Debug("No companies provided, skipping soft delete")
        return nil
    }
    cr.log.Debug("Soft deleting companies by slice", "count", len(companies))

    var companyIDs []uuid.UUID
    for _, c := range companies {
        companyIDs = append(companyIDs, c.ID)
    }
    cr.log.Debug("Collected companyIDs from slice", "companyIDs", companyIDs)

    cr.log.Info("Performing soft delete by companyIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", companyIDs).
        Delete(&types.Company{}).Error; err != nil {
        cr.log.Error("Failed to soft delete companies by slice", "error", err)
        return err
    }
    cr.log.Info("Successfully soft deleted companies by slice", "count", len(companyIDs))
    return nil
}

func (cr *companyRepo) SoftDeleteByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) error {
    cr.log.Info("Starting SoftDeleteByCompanyIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
    }

    if len(companyIDs) == 0 {
        cr.log.Debug("No companyIDs provided, skipping soft delete")
        return nil
    }
    cr.log.Debug("Soft deleting companies by IDs", "count", len(companyIDs), "companyIDs", companyIDs)

    cr.log.Info("Performing soft delete by companyIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", companyIDs).
        Delete(&types.Company{}).Error; err != nil {
        cr.log.Error("Failed to soft delete companies by IDs", "error", err)
        return err
    }
    cr.log.Info("Successfully soft deleted companies by IDs", "count", len(companyIDs))
    return nil
}

// ----------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------

func (cr *companyRepo) FullDeleteByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) error {
    cr.log.Info("Starting FullDeleteByCompanies now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
    }

    if len(companies) == 0 {
        cr.log.Debug("No companies provided, skipping full delete")
        return nil
    }
    cr.log.Debug("Full deleting companies by slice", "count", len(companies))

    var companyIDs []uuid.UUID
    for _, c := range companies {
        companyIDs = append(companyIDs, c.ID)
    }
    cr.log.Debug("Collected companyIDs from slice", "companyIDs", companyIDs)

    cr.log.Info("Performing FULL (hard) delete by companyIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", companyIDs).
        Delete(&types.Company{}).Error; err != nil {
        cr.log.Error("Failed to FULL delete companies by slice", "error", err)
        return err
    }
    cr.log.Info("Successfully FULL deleted companies by slice", "count", len(companyIDs))
    return nil
}

func (cr *companyRepo) FullDeleteByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) error {
    cr.log.Info("Starting FullDeleteByCompanyIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = cr.db
    }

    if len(companyIDs) == 0 {
        cr.log.Debug("No companyIDs provided, skipping full delete")
        return nil
    }
    cr.log.Debug("Full deleting by companyIDs", "count", len(companyIDs), "companyIDs", companyIDs)

    cr.log.Info("Performing FULL (hard) delete by companyIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", companyIDs).
        Delete(&types.Company{}).Error; err != nil {
        cr.log.Error("Failed to FULL delete companies by IDs", "error", err)
        return err
    }
    cr.log.Info("Successfully FULL deleted companies by IDs", "count", len(companyIDs))
    return nil
}

