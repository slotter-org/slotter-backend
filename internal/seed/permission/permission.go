package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/slotter-org/slotter-backend/internal/repos"
	"github.com/slotter-org/slotter-backend/internal/types"
)

func SyncPermissions(
	db											*gorm.DB,
	permissionRepo					repos.PermissionRepo,
	roleRepo								repos.RoleRepo,
	permissionSeedPathJSON	string,
) error {
	jsonFilePath := permissionSeedPathJSON
	data, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed reading permission seed file: %w", err)
	}
	var filePerms []*types.Permission
	if err := json.Unmarshal(data, &filePerms); err != nil {
		return fmt.Errorf("failed unmarshaling permissions: %w", err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		existing, err := permissionRepo.GetAll(context.Background(), tx)
		if err != nil {
			return fmt.Errorf("failed fetching existing permissions: %w", err)
		}
		fileMap := make(map[string]*types.Permission)
		for _, fp := range filePerms {
			fileMap[fp.Name] = fp
		}
		existingMap := make(map[string]*types.Permission)
		for _, ep := range existing {
			existingMap[ep.Name] = ep
		}
		var toDelete []*types.Permission
		for _, ep := range existing {
			if _, ok := fileMap[ep.Name]; !ok {
				toDelete = append(toDelete, ep)
			}
		}
		var toCreate []*types.Permission
		for _, fp := range filePerms {
			if _, ok := existingMap[fp.Name]; !ok {
				toCreate = append(toCreate, fp)
			}
		}
		var toUpdate []*types.Permission
		for _, fp := range filePerms {
			if ep, ok := existingMap[fp.Name]; ok {
				if ep.PermissionType != fp.PermissionType {
					ep.PermissionType = fp.PermissionType
					toUpdate = append(toUpdate, ep)
				}
			}
		}
		if len(toDelete) > 0 {
			var ids []uuid.UUID
			for _, p := range toDelete {
				ids = append(ids, p.ID)
			}
			if err := tx.Where("id IN ?", ids).Delete(&types.Permission{}).Error; err != nil {
				return fmt.Errorf("failed deleting old permissions: %w", err)
			}
		}
		if len(toUpdate) > 0 {
			if _, err := permissionRepo.Update(context.Background(), tx, toUpdate); err != nil {
				return fmt.Errorf("failed updating changed permissions: %w", err)
			}
		}
		finalExistingPerms, err := permissionRepo.GetAll(context.Background(), tx)
		if err != nil {
			return fmt.Errorf("failed re-fetching updated existing permissions: %w", err)
		}
		existingPermIDs := make(map[uuid.UUID]bool)
		for _, p := range finalExistingPerms {
			var isInToCreate bool
			for _, tc := range toCreate {
				if tc.Name == p.Name {
					isInToCreate = true
					break
				}
			}
			if !isInToCreate {
				existingPermIDs[p.ID] = true
			}
		}
		rolesWithAllPerms, err := findRolesWithAllPerms(tx, existingPermIDs)
		if err != nil {
			return fmt.Errorf("failed finding roles that have all existing permissions: %w", err)
		}
		var createdPerms []*types.Permission
		if len(toCreate) > 0 {
			createdPerms, err = permissionRepo.Create(context.Background(), tx, toCreate)
			if err != nil {
				return fmt.Errorf("failed creating new permissions: %w", err)
			}
		}
		if len(rolesWithAllPerms) > 0 && len(createdPerms) > 0 {
			var newPermIDs []uuid.UUID
			for _, cp := range createdPerms {
				newPermIDs = append(newPermIDs, cp.ID)
			}
			var roleIDs []uuid.UUID
			for _, r := range rolesWithAllPerms {
				roleIDs = append(roleIDs, r.ID)
			}
			if err := roleRepo.AssociatePermissionsByIDs(context.Background(), tx, roleIDs, newPermIDs); err != nil {
				return fmt.Errorf("failed associating new perms with roles: %w", err)
			}
		}
		return nil
	})
}

func findRolesWithAllPerms(tx *gorm.DB, existingPermIDs map[uuid.UUID]bool) ([]*types.Role, error) {
	if len(existingPermIDs) == 0 {
		var roles []*types.Role
		if err := tx.Find(&roles).Error; err != nil {
			return nil, err
		}
		return roles, nil
	}
	var permIDs []uuid.UUID
	for id := range existingPermIDs {
		permIDs = append(permIDs, id)
	}
	type roleCount struct {
		RoleID				uuid.UUID
		Ct						int64
	}
	var results []roleCount
	numPermsNeeded := int64(len(permIDs))
	err := tx.Raw(`
		SELECT role_id, COUNT(DISTINCT permission_id) as ct
			FROM permissions_roles
		WHERE permission_id IN ?
		GROUP BY role_id
		HAVING COUNT(DISTINCT permission_id) = ?
	`, permIDs, numPermsNeeded).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	var roleIDs []uuid.UUID
	for _, rc := range results {
		roleIDs = append(roleIDs, rc.RoleID)
	}
	var roles []*types.Role
	if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
