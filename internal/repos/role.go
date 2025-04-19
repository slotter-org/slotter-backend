package repos

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type RoleRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) ([]*types.Role, error)

    // UPDATE
    Update(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error)

    // SOFT DELETE
    SoftDeleteByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) error
    SoftDeleteByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) error
    FullDeleteByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) error

    // PERMISSIONS
    AssociatePermissionsByIDs(ctx context.Context, tx *gorm.DB, roleIDs, permissionIDs []uuid.UUID) error
    UnassociatePermissionsByIDs(ctx context.Context, tx *gorm.DB, roleIDs, permissionIDs []uuid.UUID) error
    AssociatePermissions(ctx context.Context, tx *gorm.DB, roles []*types.Role, permissions []*types.Permission) error
    UnassociatePermissions(ctx context.Context, tx *gorm.DB, roles []*types.Role, permissions []*types.Permission) error
}

type roleRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewRoleRepo(db *gorm.DB, baseLog *logger.Logger) RoleRepo {
    repoLog := baseLog.With("repo", "RoleRepo")
    return &roleRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------

func (rr *roleRepo) Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error) {
    rr.log.Info("Starting Create Roles now...")

    // 1) Transaction check
    rr.log.Info("Checking if transaction is nil...")
    transaction := tx
    if transaction != nil {
        rr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if transaction == nil {
        transaction = rr.db
        rr.log.Debug("Transaction is nil, using rr.db instead", "db", transaction)
    }

    // 2) If no roles, skip
    rr.log.Info("Checking length of roles array...")
    if len(roles) == 0 {
        rr.log.Debug("No roles provided, returning empty slice")
        return []*types.Role{}, nil
    }
    rr.log.Debug("Roles provided", "count", len(roles), "roles", roles)

    // 3) Create
    rr.log.Info("Creating roles now...")
    if err := transaction.WithContext(ctx).Create(&roles).Error; err != nil {
        rr.log.Error("Failed to create roles", "error", err)
        return nil, err
    }
    rr.log.Info("Successfully created roles", "count", len(roles))
    rr.log.Debug("Roles created", "roles", roles)
    return roles, nil
}

// ----------------------------------------------------------------
// READ
// ----------------------------------------------------------------

func (rr *roleRepo) GetByIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) ([]*types.Role, error) {
    rr.log.Info("Starting GetByIDs for roles...")

    rr.log.Info("Checking if transaction is nil...")
    transaction := tx
    if transaction != nil {
        rr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if transaction == nil {
        transaction = rr.db
        rr.log.Debug("Transaction is nil, using rr.db", "db", transaction)
    }

    var results []*types.Role
    if len(roleIDs) == 0 {
        rr.log.Debug("No roleIDs provided, returning empty slice")
        return results, nil
    }
    rr.log.Debug("RoleIDs provided", "count", len(roleIDs), "roleIDs", roleIDs)

    rr.log.Info("Fetching roles by IDs now...")
    if err := transaction.WithContext(ctx).
        Preload("Permissions").
        Where("id IN ?", roleIDs).
        Find(&results).Error; err != nil {
        rr.log.Error("Failed to fetch roles by IDs", "error", err)
        return nil, err
    }
    rr.log.Info("Successfully fetched roles by IDs", "count", len(results))
    rr.log.Debug("Roles fetched", "roles", results)
    return results, nil
}

// ----------------------------------------------------------------
// UPDATE
// ----------------------------------------------------------------

func (rr *roleRepo) Update(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error) {
    rr.log.Info("Starting Update Roles now...")

    rr.log.Info("Checking if transaction is nil...")
    transaction := tx
    if transaction != nil {
        rr.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if transaction == nil {
        transaction = rr.db
        rr.log.Debug("Transaction is nil, using rr.db", "db", transaction)
    }

    if len(roles) == 0 {
        rr.log.Debug("No roles provided, returning empty slice")
        return roles, nil
    }
    rr.log.Debug("Updating roles count", "count", len(roles), "roles", roles)

    rr.log.Info("Saving roles now...")
    for i := range roles {
        if err := transaction.WithContext(ctx).Save(&roles[i]).Error; err != nil {
            rr.log.Error("Failed to update a role", "error", err, "role", roles[i])
            return nil, err
        }
    }
    rr.log.Info("Successfully updated roles", "count", len(roles))
    rr.log.Debug("Roles updated", "roles", roles)
    return roles, nil
}

// ----------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------

func (rr *roleRepo) SoftDeleteByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) error {
    rr.log.Info("Starting SoftDeleteByRoles now...")

    // 1) Transaction
    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    // 2) If no roles, skip
    if len(roles) == 0 {
        rr.log.Debug("No roles provided, skipping soft delete")
        return nil
    }
    rr.log.Debug("Soft deleting by role slice", "count", len(roles))

    // 3) Gather IDs
    var roleIDs []uuid.UUID
    for _, r := range roles {
        roleIDs = append(roleIDs, r.ID)
    }
    rr.log.Debug("Collected roleIDs from slice", "roleIDs", roleIDs)

    // 4) Soft delete
    rr.log.Info("Performing soft delete by roleIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", roleIDs).
        Delete(&types.Role{}).Error; err != nil {
        rr.log.Error("Failed to soft delete roles by slice", "error", err)
        return err
    }
    rr.log.Info("Successfully soft deleted roles by slice", "count", len(roleIDs))
    return nil
}

func (rr *roleRepo) SoftDeleteByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) error {
    rr.log.Info("Starting SoftDeleteByRoleIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roleIDs) == 0 {
        rr.log.Debug("No roleIDs provided, skipping soft delete")
        return nil
    }
    rr.log.Debug("Soft deleting by roleIDs", "count", len(roleIDs), "roleIDs", roleIDs)

    rr.log.Info("Performing soft delete by roleIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", roleIDs).
        Delete(&types.Role{}).Error; err != nil {
        rr.log.Error("Failed to soft delete roles by roleIDs", "error", err)
        return err
    }
    rr.log.Info("Successfully soft deleted roles by IDs", "count", len(roleIDs))
    return nil
}

// ----------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------

func (rr *roleRepo) FullDeleteByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) error {
    rr.log.Info("Starting FullDeleteByRoles now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roles) == 0 {
        rr.log.Debug("No roles provided, skipping full delete")
        return nil
    }
    rr.log.Debug("Full deleting by role slice", "count", len(roles))

    var roleIDs []uuid.UUID
    for _, r := range roles {
        roleIDs = append(roleIDs, r.ID)
    }
    rr.log.Debug("Collected roleIDs from slice", "roleIDs", roleIDs)

    rr.log.Info("Performing FULL (hard) delete by roleIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", roleIDs).
        Delete(&types.Role{}).Error; err != nil {
        rr.log.Error("Failed to FULL delete roles by slice", "error", err)
        return err
    }
    rr.log.Info("Successfully FULL deleted roles by slice", "count", len(roleIDs))
    return nil
}

func (rr *roleRepo) FullDeleteByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) error {
    rr.log.Info("Starting FullDeleteByRoleIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roleIDs) == 0 {
        rr.log.Debug("No roleIDs provided, skipping full delete")
        return nil
    }
    rr.log.Debug("Full deleting by roleIDs", "count", len(roleIDs), "roleIDs", roleIDs)

    rr.log.Info("Performing FULL (hard) delete by roleIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", roleIDs).
        Delete(&types.Role{}).Error; err != nil {
        rr.log.Error("Failed to FULL delete roles by roleIDs", "error", err)
        return err
    }
    rr.log.Info("Successfully FULL deleted roles by IDs", "count", len(roleIDs))
    return nil
}

// ----------------------------------------------------------------
// PERMISSIONS
// ----------------------------------------------------------------

func (rr *roleRepo) AssociatePermissionsByIDs(ctx context.Context, tx *gorm.DB, roleIDs, permissionIDs []uuid.UUID) error {
    rr.log.Info("Starting AssociatePermissionsByIDs now...")

    // 1) Transaction
    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roleIDs) == 0 || len(permissionIDs) == 0 {
        rr.log.Debug("No roleIDs or permissionIDs provided, skipping association", "roleIDs_count", len(roleIDs), "permIDs_count", len(permissionIDs))
        return nil
    }
    rr.log.Debug("RoleIDs and PermissionIDs provided", "roleIDs", roleIDs, "permissionIDs", permissionIDs)

    // 2) Fetch roles
    roles, err := rr.GetByIDs(ctx, transaction, roleIDs)
    if err != nil {
        rr.log.Error("Failed to fetch roles for association", "error", err)
        return err
    }
    rr.log.Debug("Fetched roles for association", "count", len(roles))

    // 3) Fetch permissions
    // Because there's no direct "permissionRepo" reference here, you might do it externally or inject a permissionRepo.
    // For demonstration, let's assume you have to do it here:
    var perms []types.Permission
    if err := transaction.WithContext(ctx).
        Where("id IN ?", permissionIDs).
        Find(&perms).Error; err != nil {
        rr.log.Error("Failed to fetch permissions for association", "error", err)
        return err
    }
    rr.log.Debug("Fetched permissions for association", "count", len(perms))

    // 4) Associate
    rr.log.Info("Associating permissions with roles now...")
    for _, role := range roles {
        if e := transaction.Model(&role).Association("Permissions").Append(perms); e != nil {
            rr.log.Error("Failed to associate permissions with role", "roleID", role.ID, "error", e)
            return e
        }
    }
    rr.log.Info("Successfully associated permissions with roles")
    return nil
}

func (rr *roleRepo) UnassociatePermissionsByIDs(ctx context.Context, tx *gorm.DB, roleIDs, permissionIDs []uuid.UUID) error {
    rr.log.Info("Starting UnassociatePermissionsByIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roleIDs) == 0 || len(permissionIDs) == 0 {
        rr.log.Debug("No roleIDs or permissionIDs provided, skipping unassociation")
        return nil
    }
    rr.log.Debug("RoleIDs and PermissionIDs provided", "roleIDs", roleIDs, "permissionIDs", permissionIDs)

    // 1) Fetch roles
    roles, err := rr.GetByIDs(ctx, transaction, roleIDs)
    if err != nil {
        rr.log.Error("Failed to fetch roles for unassociation", "error", err)
        return err
    }
    rr.log.Debug("Fetched roles", "count", len(roles))

    // 2) Fetch permissions
    var perms []types.Permission
    if err := transaction.WithContext(ctx).
        Where("id IN ?", permissionIDs).
        Find(&perms).Error; err != nil {
        rr.log.Error("Failed to fetch permissions for unassociation", "error", err)
        return err
    }
    rr.log.Debug("Fetched permissions for unassociation", "count", len(perms))

    // 3) Unassociate
    rr.log.Info("Unassociating permissions from roles now...")
    for _, role := range roles {
        if e := transaction.Model(&role).Association("Permissions").Delete(perms); e != nil {
            rr.log.Error("Failed to unassociate permissions from role", "roleID", role.ID, "error", e)
            return e
        }
    }
    rr.log.Info("Successfully unassociated permissions from roles")
    return nil
}

func (rr *roleRepo) AssociatePermissions(ctx context.Context, tx *gorm.DB, roles []*types.Role, permissions []*types.Permission) error {
    rr.log.Info("Starting AssociatePermissions now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roles) == 0 || len(permissions) == 0 {
        rr.log.Debug("No roles or permissions provided, skipping association")
        return nil
    }
    rr.log.Debug("Roles and Permissions provided", "rolesCount", len(roles), "permsCount", len(permissions))

    rr.log.Info("Associating permissions with roles now...")
    for _, role := range roles {
        if e := transaction.Model(&role).Association("Permissions").Append(permissions); e != nil {
            rr.log.Error("Failed to associate permissions with role", "roleID", role.ID, "error", e)
            return e
        }
    }
    rr.log.Info("Successfully associated permissions with roles")
    return nil
}

func (rr *roleRepo) UnassociatePermissions(ctx context.Context, tx *gorm.DB, roles []*types.Role, permissions []*types.Permission) error {
    rr.log.Info("Starting UnassociatePermissions now...")

    transaction := tx
    if transaction == nil {
        transaction = rr.db
    }

    if len(roles) == 0 || len(permissions) == 0 {
        rr.log.Debug("No roles or permissions provided, skipping unassociation")
        return nil
    }
    rr.log.Debug("Roles and Permissions provided", "rolesCount", len(roles), "permsCount", len(permissions))

    rr.log.Info("Unassociating permissions from roles now...")
    for _, role := range roles {
        if e := transaction.Model(&role).Association("Permissions").Delete(permissions); e != nil {
            rr.log.Error("Failed to unassociate permissions from role", "roleID", role.ID, "error", e)
            return e
        }
    }
    rr.log.Info("Successfully unassociated permissions from roles")
    return nil
}

