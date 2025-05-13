package services

import (
    "context"
    "fmt"
    "strings"

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
    CreateLoggedIn(ctx context.Context, tx *gorm.DB, name string, description string) (*types.Role, error)
    UpdatePermissions(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, newPermsSet []types.Permission) (*types.Role, error)
    UpdateRole(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, newName string, newDescription string) (*types.Role, error)
    DeleteRole(ctx context.Context, tx *gorm.DB, roleID uuid.UUID) error
}

type roleService struct {
    db              *gorm.DB
    log             *logger.Logger
    roleRepo        repos.RoleRepo
    permissionRepo  repos.PermissionRepo
    userRepo        repos.UserRepo
    avatarService   AvatarService
}

func NewRoleService(db *gorm.DB, baseLog *logger.Logger, roleRepo repos.RoleRepo, permissionRepo repos.PermissionRepo, userRepo repos.UserRepo, avatarService AvatarService) RoleService {
    serviceLog := baseLog.With("service", "RoleService")
    return &roleService{db: db, log: serviceLog, roleRepo: roleRepo, permissionRepo: permissionRepo, userRepo: userRepo, avatarService: avatarService}
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

func (rs *roleService) CreateLoggedIn(ctx context.Context, tx *gorm.DB, name string, description string) (*types.Role, error) {
    rs.log.Info("Starting CreateLoggedIn now...")
    
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("Request data not set in context")
        return nil, fmt.Errorf("request data not set in context")
    }
    if name == "" {
        rs.log.Warn("Name for new role cannot be empty")
        return nil, fmt.Errorf("name for role cannot be empty")
    }
    entityType := "wms"
    if rd.UserType == "company" {
        entityType = "company"
    }
    createRoleFn := func(innerTx *gorm.DB) (*types.Role, error) {
        normalizedName := normalization.ParseInputString(name)
        normalizedDescription := normalization.ParseInputString(description)
        var exists bool 
        var err error
        if entityType == "company" {
            exists, err = rs.roleRepo.NameExistsByCompanyID(ctx, innerTx, rd.CompanyID, normalizedName)
        } else {
            exists, err = rs.roleRepo.NameExistsByWmsID(ctx, innerTx, rd.WmsID, normalizedName)
        }
        if err != nil {
            rs.log.Error("Failed checking role name existence", "error", err)
            return nil, fmt.Errorf("error checking role name existence: %w", err)
        }
        if exists {
            rs.log.Error("Role name already in use", "name", name, "entityType", entityType)
            return nil, fmt.Errorf("role name is already in use")
        }
        newRole := &types.Role{
            Name: normalizedName,
            Description: &normalizedDescription,
        }
        if entityType == "company" {
            newRole.CompanyID = rd.CompanyID
        } else {
            newRole.WmsID = rd.WmsID
        }
        newRoles, err := rs.Create(ctx, innerTx, []*types.Role{newRole})
        if err != nil {
            rs.log.Error("Failed to create new role", "error", err)
            return nil, fmt.Errorf("failed to create new role: %w", err)
        }
        if len(newRoles) == 0 {
            rs.log.Warn("Creating role returned no roles")
            return nil, fmt.Errorf("creating role returned no roles")
        }
        return newRoles[0], nil
    }
    if tx != nil {
        return createRoleFn(tx)
    }
    var role *types.Role
    err := rs.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
        var err error
        role, err = createRoleFn(innerTx)
        return err
    })
    if err != nil {
        return nil, err
    }
    var channel string
    switch entityType {
    case "company":
        if role.CompanyID != nil && *role.CompanyID != uuid.Nil {
            channel = "company:" + role.CompanyID.String()
        }
    case "wms":
        if role.WmsID != nil && *role.WmsID != uuid.Nil {
            channel = "wms:" + role.WmsID.String()
        }
    }
    if channel != "" {
        sseData := ssedata.GetSSEData(ctx)
        if sseData != nil {
            sseData.AppendMessage(sse.SSEMessage{
                Channel: channel,
                Event: sse.SSEEventRoleCreated,
            })
        }
    }
    return role, nil
}

//--------------------------------------------------------------------------------------------
// UPDATE
//--------------------------------------------------------------------------------------------

func (rs *roleService) UpdatePermissions(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, newPermSet []types.Permission) (*types.Role, error) {

    rs.log.Info("UpdatePermissions starting now...")
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("Request data not set in context")
        return nil, fmt.Errorf("request data not set in context")
    }
    if roleID == uuid.Nil {
        rs.log.Warn("No valid roleID passed")
        return nil, fmt.Errorf("invalid roleID")
    }
    entityType := "wms"
    if rd.UserType == "company" {
        entityType = "company"
    }
    var updatedRole *types.Role
    outerErr := rs.db.Transaction(func(innerTx *gorm.DB) error {
        effectiveTx := innerTx
        if tx != nil {
            effectiveTx = tx
        }
        loadedRoles, err := rs.roleRepo.GetByIDs(ctx, effectiveTx, []uuid.UUID{roleID})
        if err != nil {
            rs.log.Warn("Failed to load role by ID", "roleID", roleID, "error", err)
            return fmt.Errorf("failed to load role: %w", err)
        }
        if len(loadedRoles) == 0 {
            rs.log.Warn("No role found with that ID", "roleID", roleID)
            return fmt.Errorf("no role found with that ID")
        }
        theRole := loadedRoles[0]
        allPerms, allErr := rs.permissionRepo.GetAll(ctx, effectiveTx)
        if allErr != nil {
            rs.log.Warn("Failed to fetch all perms", "error", allErr)
            return fmt.Errorf("cannot fetch all permissions: %w", allErr)
        }
        allPermCount := len(allPerms)
        currentPerms := theRole.Permissions
        currentPermCount := len(currentPerms)
        hasAll := false
        if currentPermCount == allPermCount {
            permSet := make(map[uuid.UUID]bool)
            for _, p := range currentPerms {
                permSet[p.ID] = true
            }
            matchedAll := true
            for _, p := range allPerms {
                if !permSet[p.ID] {
                    matchedAll = false
                    break
                }
            }
            hasAll = matchedAll
        }
        if len(newPermSet) == 0 {
            rs.log.Debug("New permission set is empty; removing all perms from role", "roleID", theRole.ID)
            if hasAll {
                if err := rs.ensureAnotherAllPermsRoleInDomain(ctx, effectiveTx, theRole, allPerms); err != nil {
                    return err
                }
            }
            if unassocErr := rs.roleRepo.UnassociatePermissions(
                ctx, effectiveTx, []*types.Role{theRole}, theRole.Permissions,
            ); unassocErr != nil {
                rs.log.Warn("Failed to remove perms from role", "error", unassocErr)
                return fmt.Errorf("failed to remove all perms: %w", unassocErr)
            }
            rs.log.Info("Successfully removed all perms from role", "roleID", theRole.ID)

        } else {
            existingMap := make(map[uuid.UUID]bool, len(currentPerms))
            for _, cp := range currentPerms {
                existingMap[cp.ID] = true
            }
            incomingMap := make(map[uuid.UUID]bool, len(newPermSet))
            for _, np := range newPermSet {
                incomingMap[np.ID] = true
            }

            var toRemove []*types.Permission
            var toAddIDs []uuid.UUID

            for _, cp := range currentPerms {
                if !incomingMap[cp.ID] {
                    toRemove = append(toRemove, cp)
                }
            }
            for _, np := range newPermSet {
                if !existingMap[np.ID] {
                    toAddIDs = append(toAddIDs, np.ID)
                }
            }
            if hasAll && len(toRemove) > 0 {
                if err := rs.ensureAnotherAllPermsRoleInDomain(ctx, effectiveTx, theRole, allPerms); err != nil {
                    return err
                }
            }
            if len(toRemove) > 0 {
                rs.log.Debug("Removing perms from role", "count", len(toRemove), "roleID", theRole.ID)
                if err := rs.roleRepo.UnassociatePermissions(ctx, effectiveTx, []*types.Role{theRole}, toRemove); err != nil {
                    rs.log.Warn("Failed to unassociate perms", "error", err)
                    return fmt.Errorf("failed to unassociate perms: %w", err)
                }
            }
            if len(toAddIDs) > 0 {
                rs.log.Debug("Adding perms to role", "count", len(toAddIDs), "roleID", theRole.ID)
                realPerms, gErr := rs.permissionRepo.GetByIDs(ctx, effectiveTx, toAddIDs)
                if gErr != nil {
                    rs.log.Warn("Could not re‐fetch perms to add", "error", gErr)
                    return fmt.Errorf("could not re‐fetch perms to add: %w", gErr)
                }
                if err := rs.roleRepo.AssociatePermissions(ctx, effectiveTx, []*types.Role{theRole}, realPerms); err != nil {
                    rs.log.Warn("Failed to associate perms", "error", err)
                    return fmt.Errorf("failed to associate perms: %w", err)
                }
            }
        }
        newList, reloadErr := rs.roleRepo.GetByIDs(ctx, effectiveTx, []uuid.UUID{theRole.ID})
        if reloadErr != nil || len(newList) == 0 {
            rs.log.Warn("Cannot reload role after updates", "error", reloadErr)
            return fmt.Errorf("cannot reload role after updates: %v", reloadErr)
        }
        updatedRole = newList[0]
        return nil
    })

    if outerErr != nil {
        rs.log.Warn("UpdatePermissions transaction failed", "error", outerErr)
        return nil, outerErr
    }
    rs.log.Info("UpdatePermissions completed successfully", "roleID", updatedRole.ID)
    var channel string
    switch entityType {
    case "company":
        if updatedRole.CompanyID != nil && *updatedRole.CompanyID != uuid.Nil {
            channel = "company:" + updatedRole.CompanyID.String()
        }
    case "wms":
        if updatedRole.WmsID != nil && *updatedRole.WmsID != uuid.Nil {
            channel = "wms:" + updatedRole.WmsID.String()
        }
    }
    if channel != "" {
        sseData := ssedata.GetSSEData(ctx)
        if sseData != nil {
            sseData.AppendMessage(sse.SSEMessage{
                Channel: channel,
                Event: sse.SSEEventRoleUpdated,
            })
        }
    }
    return updatedRole, nil
}

func (rs *roleService) UpdateRole(ctx context.Context, tx *gorm.DB, roleId uuid.UUID, newName string, newDescription string) (*types.Role, error) {
    rs.log.Info("Starting UpdateRole now...")
    
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("No request data in context")
        return nil, fmt.Errorf("no request data in context")
    }
    if roleID == uuid.Nil {
        rs.log.Warn("Invalid roleID passed")
        return nil, fmt.Errorf("invalid roleID")
    }
    entityType := "wms"
    if rd.UserType == "company" {
        entityType = "company"
    }
    var updatedRole *types.Role
    outerErr := rs.db.Transaction(func(innerTx *gorm.DB) error {
        effectiveTx := innerTx
        if tx != nil {
            effectiveTx = tx
        }
        roles, rErr := rs.roleRepo.GetByIDs(ctx, effectiveTx, []uuid.UUID{roleID})
        if rErr != nil {
            rs.log.Warn("Failed to load role by ID", "error", rErr)
            return fmt.Errorf("failed to load role: %w", rErr)
        }
        if len(roles) == 0 {
            rs.log.Warn("No role found with that ID", "roleID", roleID)
            return fmt.Errorf("no role found with that ID")
        }
        theRole := roles[0]
        normalizedName := normalization.ParseInputString(newName)
        if normalizedName != "" && !strings.EqualFold(normalizedName, theRole.Name) {
            if theRole.CompanyID != nil && *theRole.CompanyID != uuid.Nil {
                nameExists, nErr := rs.roleRepo.NameExistsByCompanyID(ctx, effectiveTx, *theRole.CompanyID, normalizedName)
                if nErr != nil {
                    rs.log.Warn("Error checking role name uniqueness by companyID", "error", nErr)
                    return fmt.Errorf("failed checking name uniqueness: %w", nErr)
                }
                if nameExists {
                    rs.log.Warn("Role name already used within company", "companyID", *theRole.CompanyID)
                    return fmt.Errorf("role name '%s' already in use in company", normalizedName)
                }
            } else if theRole.WmsID != nil && *theRole.WmsID != uuid.Nil {
                nameExists, nErr := rs.roleRepo.NameExistsByWmsID(ctx, effectiveTx, *theRole.WmsID)
                if nErr != nil {
                    rs.log.Warn("Error checking role name uniqueness by wmsID", "error", nErr)
                    return fmt.Errorf("failed checking name uniqueness: %w", nErr)
                }
                if nameExists {
                    rs.log.Warn("Role name already used within wms", "wmsID", *theRole.WmsID)
                    return fmt.Errorf("role name '%s' already in use in wms", normalizedName)
                }
            }
            theRole.Name = normalizedName
        }
        normalizedDesc := normalization.ParseInputString(newDescription)
        if !strings.EqualFold(normalizedDesc, theRole.Description) {
            theRole.Description = &normalizedDesc
        }
        updatedSlice, upErr := rs.roleRepo.Update(ctx, effectiveTx, []*types.Role{theRole})
        if upErr != nil {
            rs.log.Warn("Failed to update role in DB", "error", upErr)
            return fmt.Errorf("failed to update role: %w", upErr)
        }
        if len(updatedSlice) == 0 {
            rs.log.Warn("No roles returned from update, unexpected")
            return fmt.Errorf("no updated role returned, unexpected DB behavior")
        }
        updatedRole = updatedSlice[0]
        return nil
    })
    if outerErr != nil {
        return nil, outerErr
    }
    rs.log.Info("Successfully updated Role's name/description", "roleID", roleID)
    var channel string
    switch entityType {
    case "company":
        if updatedRole.CompanyID != nil && *updatedRole.CompanyID != uuid.Nil {
            channel = "company:" + rd.CompanyID.String()
        }
    case "wms":
        if updatedRole.WmsID != nil && *updatedRole.WmsID != uuid.Nil {
            channel = "wms:" + rd.WmsID.String()
        }
    }
    if channel != "" {
        sseData := ssedata.GetSSEData(ctx)
        if sseData != nil {
            sseData.AppendMessage(sse.SSEMessage{
                Channel: channel,
                Event: sse.SSEEventRoleUpdated,
            })
        }
    }
    return updatedRole, nil
}

func (rs *roleService) DeleteRole(ctx context.Context, tx *gorm.DB, roleID uuid.UUID) error {
    rs.log.Info("Starting DeleteRole now...", "roleID", roleID)
    
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        rs.log.Warn("No request data in context")
        return fmt.Errorf("no request data in context")
    }
    if roleID == uuid.Nil {
        rs.log.Warn("Invalid roleID passed")
        return fmt.Errorf("invalid roleID")
    }
    entityType := "wms"
    if rd.UserType == "company" {
        entityType = "company"
    }
    return rs.db.Transaction(func(innerTx *gorm.DB) error {
        effectiveTx := innerTx
        if tx != nil {
            effectiveTx = tx
        }
        roles, rErr := rs.roleRepo.GetByIDs(ctx, effectiveTx, []uuid.UUID{roleID})
        if rErr != nil {
            rs.log.Warn("Failed to load role by ID", "error", rErr)
            return fmt.Errorf("failed to load role: %w", rErr)
        }
        if len(roles) == 0 {
            rs.log.Warn("No role found with that ID", "roleID", roleID)
            return fmt.Errorf("no role found with that ID")
        }
        theRole := roles[0]
        users, uErr := rs.userRepo.GetByRoleIDs(ctx, effectiveTx, []uuid.UUID{theRole.ID})
        if uErr != nil {
            rs.log.Warn("Failed to load users by roleID for deletion check", "error", uErr)
            return fmt.Errorf("failed to check user assignment: %w", uErr)
        }
        if len(users) > 0 {
            rs.log.Warn("Cannot delete a Role with assigned users", "roleID", theRole.ID, "count", len(users))
            return fmt.Errorf("cannot delete a Role that still has %d users(s) assigned", len(users))
        }
        allPerms, allErr := rs.permissionRepo.GetAll(ctx, effectiveTx)
        if allErr != nil {
            rs.log.Warn("Failed to load all perms in DeleteRole", "error", allErr)
            return fmt.Errorf("failed to load all perms: %w", allErr)
        }
        allCount := len(allPerms)
        currentPerms := theRole.Permissions
        hasAllPerms := false
        if len(currentPerms) == allCount {
            permSet := make(map[uuid.UUID]bool, allCount)
            for _, p := range currentPerms {
                permSet[p.ID] = true
            }
            matched := true
            for _, p := range allPerms {
                if !permSet[p.ID] {
                    matched = false
                    break
                }
            }
            hasAllPerms = matched
        }
        if hasAllPerms {
            if err := rs.ensureAnotherAllPermsRoleExists(ctx, effectiveTx, theRole, allPerms); err != nil {
                rs.log.Warn("Cannot delete the only all-perms role in its domain", "error", err)
                return err
            }
        }
        if delErr := rs.roleRepo.FullDeleteByRoles(ctx, effectiveTx, []*types.Role{theRole}); delErr != nil {
            rs.log.Warn("Failed to fully delete role from DB", "error", delErr)
            return fmt.Errorf("failed to delete role: %w", delErr)
        }
        rs.log.Info("Role successfully deleted", "roleID", theRole.ID)
        var channel string
        switch entityType {
        case "company":
            if theRole.CompanyID != nil && *theRole.CompanyID != uuid.Nil {
                channel = "company:" + theRole.CompanyID.String()
            }
        case "wms":
            if theRole.WmsID != nil && *theRole.WmsID != uuid.Nil {
                channel = "wms:" + theRole.WmsID.String()
            }
        }
        if channel != "" {
            ssd := ssedata.GetSSEData(ctx)
            if ssd != nil {
                ssd.AppendMessage(sse.SSEMessage{
                    Channel: channel,
                    Event: sse.SSEEventRoleDeleted,
                })
            }
        }
        return nil
    })
}

func (rs *roleService) ensureAnotherAllPermsRoleExists(ctx context.Context, tx *gorm.DB, theRole *types.Role, allPermCount int) error {
    rs.log.Info("Checking if another role has all perms now...")
    var rolesInDomain []*types.Role
    var err error

    if theRole.CompanyID != nil && *theRole.CompanyID != uuid.Nil {
        rolesInDomain, err = rs.roleRepo.GetByCompanyIDs(ctx, tx, []uuid.UUID{*theRole.CompanyID})
        if err != nil {
            rs.log.Warn("Failed to load roles by companyID", "error", err)
            return fmt.Errorf("failed loading roles by company: %w", err)
        }
    } else if theRole.WmsID != nil && *theRole.WmsID != uuid.Nil {
        rolesInDomain, err = rs.roleRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{*theRole.WmsID})
        if err != nil {
            rs.log.Warn("Failed to load roles by wmsID", "error", err)
            return fmt.Errorf("failed loading roles by wms: %w", err)
        }
    } else {
        rs.log.Warn("Role has neither companyID nor wmsID")
        return fmt.Errorf("role missing domain")
    }
    if len(rolesInDomain) == 0 {
        return fmt.Errorf("no other roles in domain; cannot remove perms from the only role")
    }
    var others []*types.Role
    for _, r := range rolesInDomain {
        if r.ID != theRole.ID {
            others = append(others, r)
        }
    }
    if len(others) == 0 {
        return fmt.Errorf("this is the only role in the domain. Cannot remove perms")
    }
    allCount := len(allPerms)
    for _, r := range others {
        permsOfR := r.Permissions
        if len(permsOfR) != allCount {
            continue
        }
        permMap := make(map[uuid.UUID]bool, len(permsOfR))
        for _, p := range permsOfR {
            permMap[p.ID] = true
        }
        matched := true
        for _, p := range allPerms {
            if !permMap[p.ID] {
                matched = false
                break
            }
        }
        if matched {
            rs.log.Debug("Another role in domain already has all perms", "otherRoleID", r.ID)
            return nil
        }
    }
    return fmt.Errorf("cannot remove all perms from the only role with all perms")
}

