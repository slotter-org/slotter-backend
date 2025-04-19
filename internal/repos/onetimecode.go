package repos

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type OneTimeCodeRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) ([]types.OneTimeCode, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, codeIDs []uuid.UUID) ([]types.OneTimeCode, error)
    GetByCodes(ctx context.Context, tx *gorm.DB, codes []string) ([]types.OneTimeCode, error)

    // PARTIAL UPDATE
    MarkUsed(ctx context.Context, tx *gorm.DB, otCodeID uuid.UUID) error

    // FULL UPDATE
    Update(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) ([]types.OneTimeCode, error)

    // SOFT DELETE
    SoftDeleteByOneTimeCodes(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) error
    SoftDeleteByOneTimeCodeIDs(ctx context.Context, tx *gorm.DB, otCodeIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByOneTimeCodes(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) error
    FullDeleteByOneTimeCodeIDs(ctx context.Context, tx *gorm.DB, otCodeIDs []uuid.UUID) error
}

type oneTimeCodeRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewOneTimeCodeRepo(db *gorm.DB, baseLog *logger.Logger) OneTimeCodeRepo {
    repoLog := baseLog.With("repo", "OneTimeCodeRepo")
    return &oneTimeCodeRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) Create(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) ([]types.OneTimeCode, error) {
    ocr.log.Info("Starting Create OneTimeCodes now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
        ocr.log.Debug("Transaction is nil, using ocr.db", "db", transaction)
    } else {
        ocr.log.Debug("Transaction is not nil", "transaction", transaction)
    }

    if len(otCodes) == 0 {
        ocr.log.Debug("No OneTimeCodes provided, returning empty slice")
        return []types.OneTimeCode{}, nil
    }
    ocr.log.Debug("OneTimeCodes provided", "count", len(otCodes))

    ocr.log.Info("Creating one-time codes now...")
    if err := transaction.WithContext(ctx).Create(&otCodes).Error; err != nil {
        ocr.log.Error("Failed to create one-time codes", "error", err)
        return nil, err
    }
    ocr.log.Info("Successfully created one-time codes", "count", len(otCodes))
    ocr.log.Debug("OneTimeCodes created", "otCodes", otCodes)
    return otCodes, nil
}

// ----------------------------------------------------------------
// READ
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) GetByIDs(ctx context.Context, tx *gorm.DB, codeIDs []uuid.UUID) ([]types.OneTimeCode, error) {
    ocr.log.Info("Starting GetByIDs for OneTimeCodes...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
        ocr.log.Debug("Transaction is nil, using ocr.db", "db", transaction)
    }

    var results []types.OneTimeCode
    if len(codeIDs) == 0 {
        ocr.log.Debug("No OneTimeCodeIDs provided, returning empty slice")
        return results, nil
    }
    ocr.log.Debug("OneTimeCodeIDs provided", "count", len(codeIDs), "codeIDs", codeIDs)

    ocr.log.Info("Fetching one-time codes by IDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN ?", codeIDs).
        Find(&results).Error; err != nil {
        ocr.log.Error("Failed to fetch one-time codes by IDs", "error", err)
        return nil, err
    }
    ocr.log.Info("Successfully fetched one-time codes by IDs", "count", len(results))
    ocr.log.Debug("OneTimeCodes fetched", "otCodes", results)
    return results, nil
}

func (ocr *oneTimeCodeRepo) GetByCodes(ctx context.Context, tx *gorm.DB, codes []string) ([]types.OneTimeCode, error) {
    ocr.log.Info("Starting GetByCodes for OneTimeCodes...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
        ocr.log.Debug("Transaction is nil, using ocr.db", "db", transaction)
    }

    var results []types.OneTimeCode
    if len(codes) == 0 {
        ocr.log.Debug("No codes provided, returning empty slice")
        return results, nil
    }
    ocr.log.Debug("Codes provided", "count", len(codes), "codes", codes)

    ocr.log.Info("Fetching one-time codes by code strings now...")
    if err := transaction.WithContext(ctx).
        Where("code IN ?", codes).
        Find(&results).Error; err != nil {
        ocr.log.Error("Failed to fetch one-time codes by code strings", "error", err)
        return nil, err
    }
    ocr.log.Info("Successfully fetched one-time codes by codes", "count", len(results))
    ocr.log.Debug("OneTimeCodes fetched", "otCodes", results)
    return results, nil
}

// ----------------------------------------------------------------
// PARTIAL UPDATE
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) MarkUsed(ctx context.Context, tx *gorm.DB, otCodeID uuid.UUID) error {
    ocr.log.Info("Starting MarkUsed for OneTimeCode now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
        ocr.log.Debug("Transaction is nil, using ocr.db", "db", transaction)
    }

    if otCodeID == uuid.Nil {
        ocr.log.Debug("otCodeID is nil, skipping MarkUsed")
        return nil
    }

    ocr.log.Info("Locking OneTimeCode row (for update) to mark used...", "otCodeID", otCodeID)
    var otc types.OneTimeCode
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("id = ?", otCodeID).
        First(&otc).Error; err != nil {
        ocr.log.Error("Failed to load one-time code in MarkUsed", "error", err)
        return err
    }

    if otc.Used {
        ocr.log.Debug("OneTimeCode already used, skipping", "otCodeID", otCodeID)
        return nil
    }
    otc.Used = true

    ocr.log.Info("Saving updated one-time code (Used) now...")
    if err := transaction.WithContext(ctx).Save(&otc).Error; err != nil {
        ocr.log.Error("Failed to save one-time code as used", "error", err)
        return err
    }
    ocr.log.Info("Successfully marked one-time code as used", "otCodeID", otCodeID)
    return nil
}

// ----------------------------------------------------------------
// FULL UPDATE
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) Update(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) ([]types.OneTimeCode, error) {
    ocr.log.Info("Starting Update for OneTimeCodes now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
        ocr.log.Debug("Transaction is nil, using ocr.db", "db", transaction)
    }

    if len(otCodes) == 0 {
        ocr.log.Debug("No one-time codes provided, returning empty slice")
        return otCodes, nil
    }
    ocr.log.Debug("OneTimeCodes to update", "count", len(otCodes), "otCodes", otCodes)

    ocr.log.Info("Saving each one-time code now...")
    for i := range otCodes {
        if err := transaction.WithContext(ctx).Save(&otCodes[i]).Error; err != nil {
            ocr.log.Error("Failed to update one-time code", "error", err, "otCode", otCodes[i])
            return nil, err
        }
    }
    ocr.log.Info("Successfully updated one-time codes", "count", len(otCodes))
    ocr.log.Debug("OneTimeCodes updated", "otCodes", otCodes)
    return otCodes, nil
}

// ----------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) SoftDeleteByOneTimeCodes(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) error {
    ocr.log.Info("Starting SoftDeleteByOneTimeCodes now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
    }

    if len(otCodes) == 0 {
        ocr.log.Debug("No one-time codes provided, skipping soft delete")
        return nil
    }
    ocr.log.Debug("Soft deleting one-time codes by slice", "count", len(otCodes))

    var ids []uuid.UUID
    for _, c := range otCodes {
        ids = append(ids, c.ID)
    }
    ocr.log.Debug("Collected one-time code IDs from slice", "ids", ids)

    ocr.log.Info("Performing soft delete by one-time code IDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", ids).
        Delete(&types.OneTimeCode{}).Error; err != nil {
        ocr.log.Error("Failed to soft delete one-time codes by slice", "error", err)
        return err
    }
    ocr.log.Info("Successfully soft deleted one-time codes by slice", "count", len(ids))
    return nil
}

func (ocr *oneTimeCodeRepo) SoftDeleteByOneTimeCodeIDs(ctx context.Context, tx *gorm.DB, otCodeIDs []uuid.UUID) error {
    ocr.log.Info("Starting SoftDeleteByOneTimeCodeIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
    }

    if len(otCodeIDs) == 0 {
        ocr.log.Debug("No one-time code IDs provided, skipping soft delete")
        return nil
    }
    ocr.log.Debug("Soft deleting by one-time code IDs", "count", len(otCodeIDs), "otCodeIDs", otCodeIDs)

    ocr.log.Info("Performing soft delete by one-time code IDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", otCodeIDs).
        Delete(&types.OneTimeCode{}).Error; err != nil {
        ocr.log.Error("Failed to soft delete one-time codes by IDs", "error", err)
        return err
    }
    ocr.log.Info("Successfully soft deleted one-time codes by IDs", "count", len(otCodeIDs))
    return nil
}

// ----------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------

func (ocr *oneTimeCodeRepo) FullDeleteByOneTimeCodes(ctx context.Context, tx *gorm.DB, otCodes []types.OneTimeCode) error {
    ocr.log.Info("Starting FullDeleteByOneTimeCodes now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
    }

    if len(otCodes) == 0 {
        ocr.log.Debug("No one-time codes provided, skipping full delete")
        return nil
    }
    ocr.log.Debug("Full deleting one-time codes by slice", "count", len(otCodes))

    var ids []uuid.UUID
    for _, c := range otCodes {
        ids = append(ids, c.ID)
    }
    ocr.log.Debug("Collected one-time code IDs from slice", "ids", ids)

    ocr.log.Info("Performing FULL (hard) delete by one-time code IDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", ids).
        Delete(&types.OneTimeCode{}).Error; err != nil {
        ocr.log.Error("Failed to FULL delete one-time codes by slice", "error", err)
        return err
    }
    ocr.log.Info("Successfully FULL deleted one-time codes by slice", "count", len(ids))
    return nil
}

func (ocr *oneTimeCodeRepo) FullDeleteByOneTimeCodeIDs(ctx context.Context, tx *gorm.DB, otCodeIDs []uuid.UUID) error {
    ocr.log.Info("Starting FullDeleteByOneTimeCodeIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ocr.db
    }

    if len(otCodeIDs) == 0 {
        ocr.log.Debug("No one-time code IDs provided, skipping full delete")
        return nil
    }
    ocr.log.Debug("Full deleting by one-time code IDs", "count", len(otCodeIDs), "otCodeIDs", otCodeIDs)

    ocr.log.Info("Performing FULL (hard) delete by one-time code IDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", otCodeIDs).
        Delete(&types.OneTimeCode{}).Error; err != nil {
        ocr.log.Error("Failed to FULL delete one-time codes by IDs", "error", err)
        return err
    }
    ocr.log.Info("Successfully FULL deleted one-time codes by IDs", "count", len(otCodeIDs))
    return nil
}

