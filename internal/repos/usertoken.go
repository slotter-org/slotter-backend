package repos

import (
    "context"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
    
    "github.com/yungbote/slotter/backend/internal/logger"
    "github.com/yungbote/slotter/backend/internal/types"
)

type UserTokenRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) ([]*types.UserToken, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) ([]*types.UserToken, error)
    GetByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) ([]*types.UserToken, error)
    GetByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) ([]*types.UserToken, error)
    GetByAccessTokens(ctx context.Context, tx *gorm.DB, accessTokens []string) ([]*types.UserToken, error)
    GetByRefreshTokens(ctx context.Context, tx *gorm.DB, refreshTokens []string) ([]*types.UserToken, error)

    // PARTIAL UPDATE / LOCKING EXAMPLE (If needed)
    // e.g., LockAndGetByRefreshToken(ctx, tx, refreshToken string) (*types.UserToken, error)
    // (Optional convenience methods for locking or updating fields on a single token.)

    // SOFT DELETE
    SoftDeleteByTokens(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) error
    SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) error
    SoftDeleteByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByTokens(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) error
    FullDeleteByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) error
    FullDeleteByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error
}

type userTokenRepo struct {
    db      *gorm.DB
    log     *logger.Logger
}

func NewUserTokenRepo(db *gorm.DB, baseLog *logger.Logger) UserTokenRepo {
    repoLog := baseLog.With("repo", "UserTokenRepo")
    return &userTokenRepo{db: db, log: repoLog}
}

//------------------------------------------------------------------------------
// CREATE
//------------------------------------------------------------------------------

func (utr *userTokenRepo) Create(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) ([]*types.UserToken, error) {
    utr.log.Info("Starting Create UserTokens now...")

    // 1) Transaction check
    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    // 2) If no userTokens, skip
    if len(userTokens) == 0 {
        utr.log.Debug("No userTokens provided, returning empty slice")
        return []*types.UserToken{}, nil
    }
    utr.log.Debug("Creating userTokens in DB", "count", len(userTokens))

    // 3) Create
    if err := transaction.WithContext(ctx).Create(&userTokens).Error; err != nil {
        utr.log.Error("Failed to create userTokens", "error", err)
        return nil, err
    }
    utr.log.Info("Successfully created userTokens", "count", len(userTokens))
    utr.log.Debug("UserTokens created", "userTokens", userTokens)

    return userTokens, nil
}

//------------------------------------------------------------------------------
// READ
//------------------------------------------------------------------------------

func (utr *userTokenRepo) GetByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) ([]*types.UserToken, error) {
    utr.log.Info("Starting GetByIDs for UserTokens...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    var results []*types.UserToken
    if len(tokenIDs) == 0 {
        utr.log.Debug("No tokenIDs provided, returning empty slice")
        return results, nil
    }
    utr.log.Debug("Fetching userTokens by tokenIDs", "count", len(tokenIDs), "tokenIDs", tokenIDs)

    if err := transaction.WithContext(ctx).
        Where("id IN ?", tokenIDs).
        Find(&results).Error; err != nil {
        utr.log.Error("Failed to fetch userTokens by IDs", "error", err)
        return nil, err
    }
    utr.log.Info("Successfully fetched userTokens by IDs", "count", len(results))
    utr.log.Debug("UserTokens fetched by IDs", "userTokens", results)
    return results, nil
}

func (utr *userTokenRepo) GetByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) ([]*types.UserToken, error) {
    utr.log.Info("Starting GetByUsers for UserTokens...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(users) == 0 {
        utr.log.Debug("No user slice provided, returning empty slice")
        return []*types.UserToken{}, nil
    }
    utr.log.Debug("Extracting userIDs from the user slice")
    var userIDs []uuid.UUID
    for _, u := range users {
        userIDs = append(userIDs, u.ID)
    }
    return utr.GetByUserIDs(ctx, transaction, userIDs)
}

func (utr *userTokenRepo) GetByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) ([]*types.UserToken, error) {
    utr.log.Info("Starting GetByUserIDs for UserTokens...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    var results []*types.UserToken
    if len(userIDs) == 0 {
        utr.log.Debug("No userIDs provided, returning empty slice")
        return results, nil
    }
    utr.log.Debug("Fetching userTokens by userIDs", "count", len(userIDs), "userIDs", userIDs)

    if err := transaction.WithContext(ctx).
        Where("user_id IN ?", userIDs).
        Find(&results).Error; err != nil {
        utr.log.Error("Failed to fetch userTokens by userIDs", "error", err)
        return nil, err
    }
    utr.log.Info("Successfully fetched userTokens by userIDs", "count", len(results))
    utr.log.Debug("UserTokens fetched by userIDs", "userTokens", results)
    return results, nil
}

func (utr *userTokenRepo) GetByAccessTokens(ctx context.Context, tx *gorm.DB, accessTokens []string) ([]*types.UserToken, error) {
    utr.log.Info("Starting GetByAccessTokens for UserTokens...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    var results []*types.UserToken
    if len(accessTokens) == 0 {
        utr.log.Debug("No accessTokens provided, returning empty slice")
        return results, nil
    }
    utr.log.Debug("Fetching userTokens by accessTokens", "count", len(accessTokens))

    if err := transaction.WithContext(ctx).
        Where("access_token IN ?", accessTokens).
        Find(&results).Error; err != nil {
        utr.log.Error("Failed to fetch userTokens by accessTokens", "error", err)
        return nil, err
    }
    utr.log.Info("Successfully fetched userTokens by accessTokens", "count", len(results))
    utr.log.Debug("UserTokens fetched by accessTokens", "userTokens", results)
    return results, nil
}

func (utr *userTokenRepo) GetByRefreshTokens(ctx context.Context, tx *gorm.DB, refreshTokens []string) ([]*types.UserToken, error) {
    utr.log.Info("Starting GetByRefreshTokens for UserTokens...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    var results []*types.UserToken
    if len(refreshTokens) == 0 {
        utr.log.Debug("No refreshTokens provided, returning empty slice")
        return results, nil
    }
    utr.log.Debug("Fetching userTokens by refreshTokens", "count", len(refreshTokens))

    if err := transaction.WithContext(ctx).
        Where("refresh_token IN ?", refreshTokens).
        Find(&results).Error; err != nil {
        utr.log.Error("Failed to fetch userTokens by refreshTokens", "error", err)
        return nil, err
    }
    utr.log.Info("Successfully fetched userTokens by refreshTokens", "count", len(results))
    utr.log.Debug("UserTokens fetched by refreshTokens", "userTokens", results)
    return results, nil
}

//------------------------------------------------------------------------------
// SOFT DELETE
//------------------------------------------------------------------------------

func (utr *userTokenRepo) SoftDeleteByTokens(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) error {
    utr.log.Info("Starting SoftDeleteByTokens now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(userTokens) == 0 {
        utr.log.Debug("No userTokens provided, skipping soft delete")
        return nil
    }
    utr.log.Debug("Soft deleting userTokens by slice", "count", len(userTokens))

    var tokenIDs []uuid.UUID
    for _, t := range userTokens {
        tokenIDs = append(tokenIDs, t.ID)
    }
    return utr.SoftDeleteByIDs(ctx, transaction, tokenIDs)
}

func (utr *userTokenRepo) SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) error {
    utr.log.Info("Starting SoftDeleteByIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(tokenIDs) == 0 {
        utr.log.Debug("No tokenIDs provided, skipping soft delete")
        return nil
    }
    utr.log.Debug("Soft deleting userTokens by tokenIDs", "count", len(tokenIDs), "tokenIDs", tokenIDs)

    if err := transaction.WithContext(ctx).
        Where("id IN (?)", tokenIDs).
        Delete(&types.UserToken{}).Error; err != nil {
        utr.log.Error("Failed to soft delete userTokens by IDs", "error", err)
        return err
    }
    utr.log.Info("Successfully soft deleted userTokens by IDs", "count", len(tokenIDs))
    return nil
}

func (utr *userTokenRepo) SoftDeleteByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error {
    utr.log.Info("Starting SoftDeleteByUserIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(userIDs) == 0 {
        utr.log.Debug("No userIDs provided, skipping soft delete")
        return nil
    }
    utr.log.Debug("Soft deleting userTokens by userIDs", "count", len(userIDs), "userIDs", userIDs)

    if err := transaction.WithContext(ctx).
        Where("user_id IN (?)", userIDs).
        Delete(&types.UserToken{}).Error; err != nil {
        utr.log.Error("Failed to soft delete userTokens by userIDs", "error", err)
        return err
    }
    utr.log.Info("Successfully soft deleted userTokens by userIDs", "count", len(userIDs))
    return nil
}

//------------------------------------------------------------------------------
// FULL (HARD) DELETE
//------------------------------------------------------------------------------

func (utr *userTokenRepo) FullDeleteByTokens(ctx context.Context, tx *gorm.DB, userTokens []*types.UserToken) error {
    utr.log.Info("Starting FullDeleteByTokens now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(userTokens) == 0 {
        utr.log.Debug("No userTokens provided, skipping full delete")
        return nil
    }
    utr.log.Debug("Full deleting userTokens by slice", "count", len(userTokens))

    var tokenIDs []uuid.UUID
    for _, t := range userTokens {
        tokenIDs = append(tokenIDs, t.ID)
    }
    return utr.FullDeleteByIDs(ctx, transaction, tokenIDs)
}

func (utr *userTokenRepo) FullDeleteByIDs(ctx context.Context, tx *gorm.DB, tokenIDs []uuid.UUID) error {
    utr.log.Info("Starting FullDeleteByIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(tokenIDs) == 0 {
        utr.log.Debug("No tokenIDs provided, skipping full delete")
        return nil
    }
    utr.log.Debug("Full deleting userTokens by tokenIDs", "count", len(tokenIDs), "tokenIDs", tokenIDs)

    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", tokenIDs).
        Delete(&types.UserToken{}).Error; err != nil {
        utr.log.Error("Failed to FULL delete userTokens by IDs", "error", err)
        return err
    }
    utr.log.Info("Successfully FULL deleted userTokens by IDs", "count", len(tokenIDs))
    return nil
}

func (utr *userTokenRepo) FullDeleteByUserIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error {
    utr.log.Info("Starting FullDeleteByUserIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = utr.db
        utr.log.Debug("Transaction is nil, using utr.db")
    }

    if len(userIDs) == 0 {
        utr.log.Debug("No userIDs provided, skipping full delete")
        return nil
    }
    utr.log.Debug("Full deleting userTokens by userIDs", "count", len(userIDs), "userIDs", userIDs)

    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("user_id IN (?)", userIDs).
        Delete(&types.UserToken{}).Error; err != nil {
        utr.log.Error("Failed to FULL delete userTokens by userIDs", "error", err)
        return err
    }
    utr.log.Info("Successfully FULL deleted userTokens by userIDs", "count", len(userIDs))
    return nil
}
