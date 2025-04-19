package seed

import (
	"os"
	"fmt"
	"gorm.io/gorm"

	"github.com/slotter-org/slotter-backend/internal/repos"
	"github.com/slotter-org/slotter-backend/internal/seed/permission"
)

func SeedAll(
	db									*gorm.DB,
	permissionRepo			repos.PermissionRepo,
	roleRepo						repos.RoleRepo,
) error {
	permissionSeedPathJSON := os.Getenv("SEED_PERMISSION_JSON_PATH")
	fmt.Println("Running SeedAll... seeding permissions")

	if err := permission.SyncPermissions(db, permissionRepo, roleRepo, permissionSeedPathJSON); err != nil {
		return fmt.Errorf("failed to sync permissions: %w", err)
	}

	fmt.Println("SeedAll Complete!")
	return nil
}
