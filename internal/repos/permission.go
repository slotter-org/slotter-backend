package repos

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/yungbote/slotter/backend/internal/logger"
    "github.com/yungbote/slotter/backend/internal/types"
)

// PermissionRepo defines the interface for interacting with the Permission model.
// Patterned after the other repos (e.g., user.go, role.go, etc.).
type PermissionRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) ([]*types.Permission, error)

    // READ
    GetAll(ctx context.Context, tx *gorm.DB) ([]*types.Permission, error)
    GetByIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) ([]*types.Permission, error)

    // UPDATE
    Update(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) ([]*types.Permission, error)

    // SOFT DELETE
    SoftDeleteByPermissions(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) error
    SoftDeleteByPermissionIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByPermissions(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) error
    FullDeleteByPermissionIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) error
}

type permissionRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

// NewPermissionRepo creates a new instance of a PermissionRepo
func NewPermissionRepo(db *gorm.DB, baseLog *logger.Logger) PermissionRepo {
    repoLog := baseLog.With("repo", "PermissionRepo")
    return &permissionRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------------------
func (pr *permissionRepo) Create(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) ([]*types.Permission, error) {
    pr.log.Info("Starting Create Permissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permissions) == 0 {
        pr.log.Debug("No permissions provided, returning empty slice")
        return []*types.Permission{}, nil
    }
    if err := transaction.WithContext(ctx).Create(&permissions).Error; err != nil {
        pr.log.Error("Failed to create permissions", "error", err)
        return nil, err
    }
    pr.log.Info("Successfully created permissions", "count", len(permissions))
    return permissions, nil
}

// ----------------------------------------------------------------------------
// READ
// ----------------------------------------------------------------------------
func (pr *permissionRepo) GetAll(ctx context.Context, tx *gorm.DB) ([]*types.Permission, error) {
    pr.log.Info("Starting GetAll for Permissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    var results []*types.Permission
    if err := transaction.WithContext(ctx).Find(&results).Error; err != nil {
        pr.log.Error("Failed to fetch all permissions", "error", err)
        return nil, err
    }
    pr.log.Info("Successfully fetched all permissions", "count", len(results))
    return results, nil
}

func (pr *permissionRepo) GetByIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) ([]*types.Permission, error) {
    pr.log.Info("Starting GetByIDs for Permissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    var results []*types.Permission
    if len(permIDs) == 0 {
        pr.log.Debug("No permission IDs provided, returning empty slice")
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Where("id IN ?", permIDs).
        Find(&results).Error; err != nil {
        pr.log.Error("Failed to fetch permissions by IDs", "error", err)
        return nil, err
    }
    pr.log.Info("Successfully fetched permissions by IDs", "count", len(results))
    return results, nil
}

// ----------------------------------------------------------------------------
// UPDATE
// ----------------------------------------------------------------------------
func (pr *permissionRepo) Update(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) ([]*types.Permission, error) {
    pr.log.Info("Starting Update Permissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permissions) == 0 {
        pr.log.Debug("No permissions provided, returning empty slice")
        return permissions, nil
    }
    for i := range permissions {
        if err := transaction.WithContext(ctx).Save(&permissions[i]).Error; err != nil {
            pr.log.Error("Failed to update permission", "error", err, "permission", permissions[i])
            return nil, err
        }
    }
    pr.log.Info("Successfully updated permissions", "count", len(permissions))
    return permissions, nil
}

// ----------------------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------------------
func (pr *permissionRepo) SoftDeleteByPermissions(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) error {
    pr.log.Info("Starting SoftDeleteByPermissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permissions) == 0 {
        pr.log.Debug("No permissions provided, skipping soft delete")
        return nil
    }
    var ids []uuid.UUID
    for _, p := range permissions {
        ids = append(ids, p.ID)
    }
    pr.log.Debug("Collected permission IDs for soft delete", "ids", ids)
    if err := transaction.WithContext(ctx).
        Where("id IN ?", ids).
        Delete(&types.Permission{}).Error; err != nil {
        pr.log.Error("Failed to soft delete permissions by slice", "error", err)
        return err
    }
    pr.log.Info("Successfully soft deleted permissions by slice", "count", len(ids))
    return nil
}

func (pr *permissionRepo) SoftDeleteByPermissionIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) error {
    pr.log.Info("Starting SoftDeleteByPermissionIDs now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permIDs) == 0 {
        pr.log.Debug("No permission IDs provided, skipping soft delete")
        return nil
    }
    pr.log.Debug("Soft deleting by permission IDs", "count", len(permIDs), "permIDs", permIDs)
    if err := transaction.WithContext(ctx).
        Where("id IN ?", permIDs).
        Delete(&types.Permission{}).Error; err != nil {
        pr.log.Error("Failed to soft delete permissions by IDs", "error", err)
        return err
    }
    pr.log.Info("Successfully soft deleted permissions by IDs", "count", len(permIDs))
    return nil
}

// ----------------------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------------------
func (pr *permissionRepo) FullDeleteByPermissions(ctx context.Context, tx *gorm.DB, permissions []*types.Permission) error {
    pr.log.Info("Starting FullDeleteByPermissions now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permissions) == 0 {
        pr.log.Debug("No permissions provided, skipping full delete")
        return nil
    }
    var ids []uuid.UUID
    for _, p := range permissions {
        ids = append(ids, p.ID)
    }
    pr.log.Debug("Collected permission IDs for FULL (hard) delete", "ids", ids)
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN ?", ids).
        Delete(&types.Permission{}).Error; err != nil {
        pr.log.Error("Failed to FULL delete permissions by slice", "error", err)
        return err
    }
    pr.log.Info("Successfully FULL deleted permissions by slice", "count", len(ids))
    return nil
}

func (pr *permissionRepo) FullDeleteByPermissionIDs(ctx context.Context, tx *gorm.DB, permIDs []uuid.UUID) error {
    pr.log.Info("Starting FullDeleteByPermissionIDs now...")
    transaction := tx
    if transaction == nil {
        transaction = pr.db
    }
    if len(permIDs) == 0 {
        pr.log.Debug("No permission IDs provided, skipping full delete")
        return nil
    }
    pr.log.Debug("Full deleting by permission IDs", "count", len(permIDs), "permIDs", permIDs)
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN ?", permIDs).
        Delete(&types.Permission{}).Error; err != nil {
        pr.log.Error("Failed to FULL delete permissions by IDs", "error", err)
        return err
    }
    pr.log.Info("Successfully FULL deleted permissions by IDs", "count", len(permIDs))
    return nil
}

