package services

import (
    "context"
    "fmt"

    "gorm.io/gorm"

    "github.com/google/uuid"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/normalization"
    "github.com/slotter-org/slotter-backend/internal/requestdata"
    "github.com/slotter-org/slotter-backend/internal/ssedata"
    "github.com/slotter-org/slotter-backend/internal/sse"
    "github.com/slotter-org/slotter-backend/internal/types"
    "github.com/slotter-org/slotter-backend/internal/repos"
)

type RoleService interface {
    Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error)
    CreateLoggedInWithEntity(ctx context.Context, tx *gorm.DB, name string, description string) (types.Role, error)
    UpdateRole(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, name string, description string, permissions []types.Permission) (types.Role, error)
    UpdatePermissions(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, permissions []types.Permission) (*types.Role, error)
    UpdatePermissionsWithTransaction(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, permissions []types.Permission) (*types.Role, error)
    UpdateFields(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, name string, description string) (types.Role, error)
    UpdateFieldsWithTransaction(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, name string, description string) (*types.Role, error)
    createLoggedIn(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (types.Role, error)
    createLoggedInWithTransaction(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (*types.Role, error)
}

type roleService struct {
    db            *gorm.DB
    log           *logger.Logger
    roleRepo      repos.RoleRepo
    avatarService AvatarService
}

func NewRoleService(db *gorm.DB, baseLog *logger.Logger, roleRepo repos.RoleRepo, avatarService AvatarService) RoleService {
    serviceLog := baseLog.With("service", "RoleService")
    return &roleService{db: db, log: serviceLog, roleRepo: roleRepo, avatarService: avatarService}
}

// -------------------------------------------------------------------
// CREATE
// -------------------------------------------------------------------
func (rs *roleService) Create(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.Role, error) {
    rs.log.Info("Starting Create Roles now...")
    if tx == nil {
        rs.log.Error("Transaction cannot be nil")
        return nil, fmt.Errorf("transaction cannot be nil")
    }

    createdRoles, err := rs.roleRepo.Create(ctx, tx, roles)
    if err != nil {
        rs.log.Error("failed to create roles", "error", err)
        return nil, fmt.Errorf("failed to create roles: %w", err)
    }

    var toUpdateRoles []*types.Role
    for i, role := range createdRoles {
        rs.log.Info(fmt.Sprintf("Uploading avatar for role #%d (ID=%s)", i, role.ID))
        updatedRole, err := rs.avatarService.CreateAndUploadRoleAvatar(ctx, tx, role)
        if err != nil {
            rs.log.Error("avatar upload failed", "roleID", "error", err)
            return nil, fmt.Errorf("failed to create/upload avatar for role %s: %w", role.ID, err)
        }
        toUpdateRoles = append(toUpdateRoles, updatedRole)
    }

    updatedRoles, err := rs.roleRepo.Update(ctx, tx, toUpdateRoles)
    if err != nil {
        rs.log.Error("Failed to update roles with avatar details", "error", err)
        return nil, fmt.Errorf("failed to update roles with avatar details: %w", err)
    }

    rs.log.Info("All role avatars created and upload successfully")
    return updatedRoles, nil
}

// -------------------------------------------------------------------
// CREATE LOGGED-IN WITH ENTITY
// -------------------------------------------------------------------
func (rs *roleService) CreateLoggedInWithEntity(ctx context.Context, tx *gorm.DB, name string, description string) (types.Role, error) {
    rs.log.Info("Starting CreateLoggedInWithEntity now...")
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("Request Data is not set in context")
        return types.Role{}, fmt.Errorf("request data not set in context")
    }

    var entityType string
    if rd.UserType == "company" {
        entityType = "company"
    } else {
        entityType = "wms"
    }

    role, err := rs.createLoggedIn(ctx, tx, entityType, name, description)
    if err != nil {
        return types.Role{}, err
    }

    var channel string
    switch rd.UserType {
    case "company":
        if rd.CompanyID != uuid.Nil {
            channel = "company:" + rd.CompanyID.String()
        }
    case "wms":
        if rd.WmsID != uuid.Nil {
            channel = "wms:" + rd.WmsID.String()
        }
    }

    if channel != "" {
        sseData := ssedata.GetSSEData(ctx)
        if sseData != nil {
            sseData.AppendMessage(sse.SSEMessage{
                Channel: channel,
                Event:   sse.SSEEventRoleCreated,
            })
        }
    }
    return role, nil
}

// -------------------------------------------------------------------
// CREATE LOGGED-IN
// -------------------------------------------------------------------
func (rs *roleService) createLoggedIn(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (types.Role, error) {
    rs.log.Info("Starting createLoggedIn now...")
    if tx != nil {
        rs.log.Warn("Please use createLoggedInWithTransaction if you have a transaction")
        return types.Role{}, fmt.Errorf("transaction passed into non-transaction handling function")
    }

    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("Request Data is not set in context")
        return types.Role{}, fmt.Errorf("request data is not set in context")
    }

    nName := normalization.ParseInputString(name)
    if nName == "" {
        rs.log.Warn("No name provided to createLoggedIn")
        return types.Role{}, fmt.Errorf("need a non-empty name to create role")
    }

    if entityType == "company" && rd.UserType != "company" {
        rs.log.Warn("UserType mismatch. Expected company but got something else.")
        return types.Role{}, fmt.Errorf("UserType mismatch: can't create company role as a non company user type")
    }
    if entityType == "wms" && rd.UserType != "wms" {
        rs.log.Warn("UserType mismatch. Expected wms but got something else.")
        return types.Role{}, fmt.Errorf("UserType mismatch: can't create wms role as a non wms user type")
    }

    var result types.Role
    err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        rolePtr, rErr := rs.createLoggedInWithTransaction(ctx, innerTx, entityType, nName, description)
        if rErr != nil {
            return rErr
        }
        if rolePtr == nil {
            rs.log.Warn("createLoggedInWithTransaction returned nil role ptr")
            return fmt.Errorf("nil role ptr returned from createLoggedInWithTransaction")
        }
        result = *rolePtr
        return nil
    })
    if err != nil {
        return types.Role{}, err
    }
    return result, nil
}

// -------------------------------------------------------------------
// CREATE LOGGED-IN WITH TRANSACTION
// -------------------------------------------------------------------
func (rs *roleService) createLoggedInWithTransaction(ctx context.Context, tx *gorm.DB, entityType string, name string, description string) (*types.Role, error) {
    rs.log.Info("Starting createLoggedInWithTransaction now...")
    if tx == nil {
        rs.log.Warn("createLoggedInWithTransaction called with a nil transaction")
        return nil, fmt.Errorf("transaction is required and cannot be nil")
    }

    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("RequestData not set in context")
        return nil, fmt.Errorf("request data not set in context")
    }

    var entityID uuid.UUID
    switch entityType {
    case "company":
        if rd.CompanyID == uuid.Nil {
            rs.log.Warn("No CompanyID in RequestData")
            return nil, fmt.Errorf("user does not have a valid company id in request data")
        }
        entityID = rd.CompanyID
    case "wms":
        if rd.WmsID == uuid.Nil {
            rs.log.Warn("No WmsID in RequestData")
            return nil, fmt.Errorf("user does not have a valid wms id in request data")
        }
        entityID = rd.WmsID
    default:
        rs.log.Warn("Unsupported entityType", "entityType", entityType)
        return nil, fmt.Errorf("unsupported entityType: %s", entityType)
    }

    var exists bool
    var err error
    if entityType == "company" {
        exists, err = rs.roleRepo.NameExistsByCompanyID(ctx, tx, entityID, name)
    } else {
        exists, err = rs.roleRepo.NameExistsByWmsID(ctx, tx, entityID, name)
    }
    if err != nil {
        rs.log.Warn("Failure checking new role name existence", "err", err)
        return nil, fmt.Errorf("failed to check role name existence: %w", err)
    }
    if exists {
        rs.log.Warn("Role name already exists for entity", "entityType", entityType, "name", name)
        return nil, fmt.Errorf("role name %q already exists for %s", name, entityType)
    }

    newRole := &types.Role{
        Name:        name,
        Description: &description,
    }
    if entityType == "company" {
        newRole.CompanyID = &entityID
    } else {
        newRole.WmsID = &entityID
    }

    newRoles, nrErr := rs.Create(ctx, tx, []*types.Role{newRole})
    if nrErr != nil {
        rs.log.Warn("Failure to create new role", "error", nrErr)
        return nil, fmt.Errorf("failed to create new role: %w", nrErr)
    }
    if len(newRoles) == 0 {
        rs.log.Warn("Creating role returned no items", "count", len(newRoles))
        return nil, fmt.Errorf("creating role returned no roles")
    }
    finalRole := newRoles[0]
    return finalRole, nil
}

// -------------------------------------------------------------------
// UPDATE PERMISSIONS
// -------------------------------------------------------------------
func (rs *roleService) UpdatePermissions(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, permissions []types.Permission) (*types.Role, error) {
    rs.log.Info("UpdatePermissions starting now...")
    if tx != nil {
        rs.log.Warn("Use UpdatePermissionsWithTransaction if you are passing in a transaction")
        return nil, fmt.Errorf("use UpdatePermissionsWithTransaction if you are passing in a transaction")
    }

    var updatedRole *types.Role
    err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        r, e := rs.UpdatePermissionsWithTransaction(ctx, innerTx, roleId, permissions)
        if e != nil {
            return e
        }
        updatedRole = r
        return nil
    })
    if err != nil {
        return nil, err
    }
    if updatedRole == nil {
        return nil, fmt.Errorf("unexpected nil role returned from UpdatePermissionsWithTransaction")
    }
    return updatedRole, nil
}

// -------------------------------------------------------------------
// UPDATE PERMISSIONS WITH TRANSACTION
// -------------------------------------------------------------------
func (rs *roleService) UpdatePermissionsWithTransaction(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, permissions []types.Permission) (*types.Role, error) {
    rs.log.Info("UpdatePermissionsWithTransaction starting now...")
    if tx == nil {
        rs.log.Warn("Transaction is nil in UpdatePermissionsWithTransaction")
        return nil, fmt.Errorf("transaction cannot be nil in UpdatePermissionsWithTransaction")
    }

    rs.log.Info("Fetching existing role with permissions...")
    existingRoles, err := rs.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{roleId})
    if err != nil {
        rs.log.Warn("Failed to fetch the role by ID", "roleId", roleId, "error", err)
        return nil, fmt.Errorf("failed to fetch role by ID: %w", err)
    }
    if len(existingRoles) == 0 {
        rs.log.Warn("Role not found for the given ID", "roleId", roleId)
        return nil, fmt.Errorf("role not found for ID: %s", roleId.String())
    }

    role := existingRoles[0]
    if err := tx.Model(&role).Preload("Permissions").Find(&role).Error; err != nil {
        rs.log.Error("Failed to preload role.Permissions", "error", err)
        return nil, fmt.Errorf("failed to preload role permissions: %w", err)
    }

    // For "all-permissions" checks, load all known permissions
    allPerms, err := rs.permissionRepo.GetAll(ctx, tx)
    if err != nil {
        rs.log.Warn("Failed to fetch all permissions", "error", err)
        return nil, fmt.Errorf("failed to fetch all permissions: %w", err)
    }

    // Build current perms map
    currentPermsMap := make(map[uuid.UUID]bool, len(role.Permissions))
    for _, p := range role.Permissions {
        currentPermsMap[p.ID] = true
    }

    // Build new perms map
    newPermsMap := make(map[uuid.UUID]bool, len(permissions))
    for _, p := range permissions {
        newPermsMap[p.ID] = true
    }

    // is "all-permissions"?
    hasAllPermissions := (len(role.Permissions) == len(allPerms))

    // Figure out what to remove vs. add
    var toRemove []uuid.UUID
    var toAdd []uuid.UUID

    for _, oldP := range role.Permissions {
        if !newPermsMap[oldP.ID] {
            toRemove = append(toRemove, oldP.ID)
        }
    }
    for _, newP := range permissions {
        if !currentPermsMap[newP.ID] {
            toAdd = append(toAdd, newP.ID)
        }
    }

    // If removing perms from the only role that has them all, ensure there's another "all-permissions" role
    if hasAllPermissions && len(toRemove) > 0 {
        rs.log.Info("Role currently has all permissions. Checking if removing is allowed...")

        // Fetch sibling roles in the same company or wms
        var siblingRoles []*types.Role
        if role.CompanyID != nil && *role.CompanyID != uuid.Nil {
            siblingRoles, err = rs.roleRepo.GetByCompanyIDs(ctx, tx, []uuid.UUID{*role.CompanyID})
            if err != nil {
                rs.log.Warn("Failed to fetch sibling roles for company", "companyID", *role.CompanyID)
                return nil, fmt.Errorf("failed to fetch sibling roles for company: %w", err)
            }
        } else if role.WmsID != nil && *role.WmsID != uuid.Nil {
            siblingRoles, err = rs.roleRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{*role.WmsID})
            if err != nil {
                rs.log.Warn("Failed to fetch sibling roles for wms", "wmsID", *role.WmsID)
                return nil, fmt.Errorf("failed to fetch sibling roles for wms: %w", err)
            }
        } else {
            rs.log.Error("Role has no valid companyID or wmsID, cannot proceed")
            return nil, fmt.Errorf("role is not associated with a company or wms")
        }

        // Filter out the current role from siblings
        var otherRoles []*types.Role
        for _, sr := range siblingRoles {
            if sr.ID != role.ID {
                otherRoles = append(otherRoles, sr)
            }
        }

        // Preload permissions for each sibling
        for i := range otherRoles {
            if err := tx.Model(otherRoles[i]).Preload("Permissions").Find(&otherRoles[i]).Error; err != nil {
                rs.log.Warn("Failed to preload sibling role permissions", "roleID", otherRoles[i].ID, "error", err)
                return nil, fmt.Errorf("failed to preload sibling role permissions: %w", err)
            }
        }

        // Check if at least one sibling also has all perms
        var siblingWithAll bool
        for _, sr := range otherRoles {
            if len(sr.Permissions) == len(allPerms) {
                siblingWithAll = true
                break
            }
        }
        if !siblingWithAll {
            rs.log.Warn("No other role in the entity has all permissions. Cannot remove from the only 'all-perms' role.")
            return nil, fmt.Errorf("cannot remove permissions from the only all-permissions role")
        }
    }

    // Unassociate removed perms
    if len(toRemove) > 0 {
        rs.log.Info("Unassociating permissions from role...", "count", len(toRemove))
        if err := rs.roleRepo.UnassociatePermissionsByIDs(ctx, tx, []uuid.UUID{role.ID}, toRemove); err != nil {
            rs.log.Warn("Failed to unassociate permissions from role", "roleID", role.ID, "error", err)
            return nil, fmt.Errorf("failed to unassociate permissions: %w", err)
        }
    }

    // Associate new perms
    if len(toAdd) > 0 {
        rs.log.Info("Associating permissions with role...", "count", len(toAdd))
        if err := rs.roleRepo.AssociatePermissionsByIDs(ctx, tx, []uuid.UUID{role.ID}, toAdd); err != nil {
            rs.log.Warn("Failed to associate permissions with role", "roleID", role.ID, "error", err)
            return nil, fmt.Errorf("failed to associate permissions: %w", err)
        }
    }

    // Re-fetch updated role
    var updatedRole types.Role
    if err := tx.Model(&role).Preload("Permissions").First(&updatedRole).Error; err != nil {
        rs.log.Warn("Failed to fetch updated role after permission changes", "error", err)
        return nil, fmt.Errorf("failed to fetch updated role: %w", err)
    }

    rs.log.Info("Successfully updated permissions for role", "roleID", role.ID)
    return &updatedRole, nil
}

// -------------------------------------------------------------------
// UPDATE FIELDS
// -------------------------------------------------------------------
func (rs *roleService) UpdateFields(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, name string, description string) (types.Role, error) {
    rs.log.Info("Starting UpdateFields for role now...")
    if tx != nil {
        rs.log.Warn("UpdateFields called with a transaction. Use UpdateFieldsWithTransaction.")
        return types.Role{}, fmt.Errorf("use UpdateFieldsWithTransaction if you are passing in a transaction")
    }

    var updatedRole types.Role
    err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        r, e := rs.UpdateFieldsWithTransaction(ctx, innerTx, roleId, name, description)
        if e != nil {
            return e
        }
        updatedRole = *r
        return nil
    })
    if err != nil {
        return types.Role{}, err
    }
    return updatedRole, nil
}

// -------------------------------------------------------------------
// UPDATE FIELDS WITH TRANSACTION
// -------------------------------------------------------------------
func (rs *roleService) UpdateFieldsWithTransaction(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, name string, description string) (*types.Role, error) {
    rs.log.Info("UpdateFieldsWithTransaction starting now...")
    if tx == nil {
        rs.log.Warn("Transaction is nil in UpdateFieldsWithTransaction")
        return nil, fmt.Errorf("transaction cannot be nil in UpdateFieldsWithTransaction")
    }

    rs.log.Info("Fetching existing role from DB...")
    existingRoles, err := rs.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{roleId})
    if err != nil {
        rs.log.Warn("Failed to fetch role by ID", "roleId", roleId, "error", err)
        return nil, fmt.Errorf("failed to fetch role by ID: %w", err)
    }
    if len(existingRoles) == 0 {
        rs.log.Warn("No role found for the given ID", "roleId", roleId)
        return nil, fmt.Errorf("role not found for ID: %s", roleId.String())
    }

    role := existingRoles[0]

    // Update name if not empty
    if name != "" {
        var exists bool
        nName := normalization.ParseInputString(name)

        // Check uniqueness within the same entity (company or wms)
        if role.CompanyID != nil && *role.CompanyID != uuid.Nil {
            exists, err = rs.roleRepo.NameExistsByCompanyID(ctx, tx, *role.CompanyID, nName)
        } else if role.WmsID != nil && *role.WmsID != uuid.Nil {
            exists, err = rs.roleRepo.NameExistsByWmsID(ctx, tx, *role.WmsID, nName)
        } else {
            rs.log.Warn("Role doesn't have either a wms or company associated")
            return nil, fmt.Errorf("role has no wms or company association")
        }
        if err != nil {
            rs.log.Warn("Failed to check name existence for new name for role", "err", err)
            return nil, fmt.Errorf("failed to check name existence for new name for role: %w", err)
        }
        if exists {
            rs.log.Warn("Name already in use for another role")
            return nil, fmt.Errorf("name is already in use by another role")
        }
        role.Name = nName
    }

    // Update description if not "nochange"
    if description != "nochange" {
        role.Description = &description
    }

    rs.log.Info("Updating the role in DB with new fields...")
    updated, err := rs.roleRepo.Update(ctx, tx, []*types.Role{role})
    if err != nil {
        rs.log.Warn("Failed to update role fields", "error", err)
        return nil, fmt.Errorf("failed to update role fields: %w", err)
    }
    if len(updated) == 0 {
        rs.log.Warn("No updated results returned from roleRepo.Update")
        return nil, fmt.Errorf("no updated results returned from roleRepo.Update")
    }

    updatedRole := updated[0]
    rs.log.Info("Successfully updated the role fields", "roleId", updatedRole.ID)
    return updatedRole, nil
}

// -------------------------------------------------------------------
// UPDATE ROLE (DYNAMIC)
// -------------------------------------------------------------------
//
// name == ""         => skip name update
// description=="..." => if "nochange", skip description; otherwise update
// permissions==nil   => skip perms update; (non-nil => set exactly to the provided slice)
func (rs *roleService) UpdateRole(
    ctx context.Context,
    tx *gorm.DB,
    roleId uuid.UUID,
    name string,
    description string, // pass "nochange" if you want to skip description
    permissions []types.Permission, // nil => skip perms; non-nil => set perms exactly
) (types.Role, error) {

    rs.log.Info("UpdateRole starting now...")
    var finalRole types.Role

    updateFn := func(innerTx *gorm.DB) error {
        // 1) Update fields if name != "" or description != "nochange"
        if name != "" || description != "nochange" {
            rs.log.Info("Updating fields for role name/description...")
            updatedFieldsRole, err := rs.UpdateFieldsWithTransaction(ctx, innerTx, roleId, name, description)
            if err != nil {
                return err
            }
            finalRole = *updatedFieldsRole
        }

        // 2) Update perms if permissions != nil (including empty slice)
        if permissions != nil {
            rs.log.Info("Updating permissions for role...")
            updatedPermsRole, err := rs.UpdatePermissionsWithTransaction(ctx, innerTx, roleId, permissions)
            if err != nil {
                return err
            }
            finalRole = *updatedPermsRole
        }

        // 3) If NOTHING changed, just fetch the existing role
        if name == "" && description == "nochange" && permissions == nil {
            rs.log.Info("No fields or permissions to update. Just fetching the role.")
            existing, err := rs.roleRepo.GetByIDs(ctx, innerTx, []uuid.UUID{roleId})
            if err != nil {
                return fmt.Errorf("failed to fetch role: %w", err)
            }
            if len(existing) == 0 {
                return fmt.Errorf("role not found for ID: %s", roleId)
            }
            finalRole = *existing[0]
        }

        return nil
    }

    // Use existing transaction if provided, otherwise start a new one
    if tx != nil {
        rs.log.Info("Using the provided transaction for UpdateRole...")
        if err := updateFn(tx); err != nil {
            return types.Role{}, err
        }
    } else {
        rs.log.Info("No transaction provided; creating one for UpdateRole...")
        err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
            return updateFn(innerTx)
        })
        if err != nil {
            return types.Role{}, err
        }
    }

    rs.log.Info("UpdateRole completed successfully.", "roleId", finalRole.ID)
    return finalRole, nil
}

