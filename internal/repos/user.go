package repos

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/requestdata"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type UserRepo interface {
    // CREATE
    Create(ctx context.Context, tx *gorm.DB, users []*types.User) ([]*types.User, error)

    // READ
    GetByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) ([]*types.User, error)
    GetByEmails(ctx context.Context, tx *gorm.DB, userEmails []string) ([]*types.User, error)
    EmailExists(ctx context.Context, tx *gorm.DB, userEmail string) (bool, error)
    GetByPhoneNumbers(ctx context.Context, tx *gorm.DB, userPhoneNumbers []string) ([]*types.User, error)
    PhoneNumberExists(ctx context.Context, tx *gorm.DB, userPhoneNumber string) (bool, error)
    GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.User, error)
    GetByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.User, error)
    GetByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.User, error)
    GetByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.User, error)
    GetByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) ([]*types.User, error)
    GetByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.User, error)

    // SOFT DELETE
    SoftDeleteByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) error
    SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error

    // FULL (HARD) DELETE
    FullDeleteByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) error
    FullDeleteByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error

    // MISC
    GetMe(ctx context.Context, tx *gorm.DB) (*types.User, error)
    DeleteMe(ctx context.Context, tx *gorm.DB) error
}

type userRepo struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewUserRepo(db *gorm.DB, baseLog *logger.Logger) UserRepo {
    // Add a repo field for consistent logs
    repoLog := baseLog.With("repo", "UserRepo")
    return &userRepo{db: db, log: repoLog}
}

// ----------------------------------------------------------------
// CREATE
// ----------------------------------------------------------------

func (ur *userRepo) Create(ctx context.Context, tx *gorm.DB, users []*types.User) ([]*types.User, error) {
    ur.log.Info("Starting Create Users now...")

    // 1) Check transaction
    ur.log.Info("Checking if transaction is nil...")
    transaction := tx
    if transaction != nil {
        ur.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db instead", "db", transaction)
    }

    // 2) Check if empty
    ur.log.Info("Checking length of given users array...")
    if len(users) == 0 {
        ur.log.Debug("Users array is empty, returning empty slice", "count", 0)
        return []*types.User{}, nil
    }
    ur.log.Debug("Users array has items", "count", len(users))

    // 3) Create
    ur.log.Info("Creating users now in DB...")
    if err := transaction.WithContext(ctx).Create(&users).Error; err != nil {
        ur.log.Error("Failed to create users", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully created users", "count", len(users))
    ur.log.Debug("Users created details", "users", users)
    return users, nil
}

// ----------------------------------------------------------------
// READ
// ----------------------------------------------------------------

func (ur *userRepo) GetByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) ([]*types.User, error) {
    ur.log.Info("Starting GetByIDs for Users now...")

    // 1) Transaction
    ur.log.Info("Checking if transaction is nil...")
    transaction := tx
    if transaction != nil {
        ur.log.Debug("Transaction is not nil", "transaction", transaction)
    }
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User

    // 2) If no userIDs
    ur.log.Info("Checking length of userIDs array...")
    if len(userIDs) == 0 {
        ur.log.Debug("No userIDs provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("UserIDs provided", "count", len(userIDs), "userIDs", userIDs)

    // 3) Query
    ur.log.Info("Fetching users by userIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN ?", userIDs).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by IDs", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by IDs", "count", len(results))
    ur.log.Debug("Users fetched", "users", results)
    return results, nil
}

func (ur *userRepo) GetByEmails(ctx context.Context, tx *gorm.DB, userEmails []string) ([]*types.User, error) {
    ur.log.Info("Starting GetByEmails for Users now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User
    if len(userEmails) == 0 {
        ur.log.Debug("No userEmails provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("UserEmails provided", "count", len(userEmails), "emails", userEmails)

    ur.log.Info("Fetching users by Emails now...")
    if err := transaction.WithContext(ctx).
        Where("email IN ?", userEmails).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by emails", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by emails", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) EmailExists(ctx context.Context, tx *gorm.DB, userEmail string) (bool, error) {
    ur.log.Info("Starting EmailExists now...", "email", userEmail)

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var count int64
    ur.log.Info("Counting users with the provided email...")
    if err := transaction.WithContext(ctx).
        Model(&types.User{}).
        Where("email = ?", userEmail).
        Count(&count).Error; err != nil {
        ur.log.Error("Failed to count users by email", "error", err)
        return false, err
    }
    exists := count > 0
    ur.log.Info("EmailExists check complete", "email", userEmail, "exists", exists)
    return exists, nil
}

func (ur *userRepo) GetByPhoneNumbers(ctx context.Context, tx *gorm.DB, userPhoneNumbers []string) ([]*types.User, error) {
    ur.log.Info("Starting GetByPhoneNumbers now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User
    if len(userPhoneNumbers) == 0 {
        ur.log.Debug("No phoneNumbers provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("PhoneNumbers provided", "count", len(userPhoneNumbers), "phones", userPhoneNumbers)

    ur.log.Info("Fetching users by phoneNumbers now...")
    if err := transaction.WithContext(ctx).
        Where("phone_number IN ?", userPhoneNumbers).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by phoneNumbers", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by phoneNumbers", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) PhoneNumberExists(ctx context.Context, tx *gorm.DB, userPhoneNumber string) (bool, error) {
    ur.log.Info("Starting PhoneNumberExists now...", "phoneNumber", userPhoneNumber)

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var count int64
    ur.log.Info("Counting users with the provided phoneNumber...")
    if err := transaction.WithContext(ctx).
        Model(&types.User{}).
        Where("phone_number = ?", userPhoneNumber).
        Count(&count).Error; err != nil {
        ur.log.Error("Failed to count users by phoneNumber", "error", err)
        return false, err
    }
    exists := count > 0
    ur.log.Info("PhoneNumberExists check complete", "phoneNumber", userPhoneNumber, "exists", exists)
    return exists, nil
}

func (ur *userRepo) GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.User, error) {
    ur.log.Info("Starting GetByWmsIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User
    if len(wmsIDs) == 0 {
        ur.log.Debug("No wmsIDs provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("WmsIDs provided", "count", len(wmsIDs), "wmsIDs", wmsIDs)

    ur.log.Info("Fetching users by WmsIDs now...")
    if err := transaction.WithContext(ctx).
        Where("wms_id IN ?", wmsIDs).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by WmsIDs", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by WmsIDs", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) GetByWmss(ctx context.Context, tx *gorm.DB, wmss []*types.Wms) ([]*types.User, error) {
    ur.log.Info("Starting GetByWmss now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(wmss) == 0 {
        ur.log.Debug("No Wms records provided, returning empty slice")
        return nil, nil
    }
    ur.log.Debug("Wms records provided", "count", len(wmss), "wmss", wmss)

    ur.log.Info("Collecting Wms IDs from slice now...")
    var wmsIDs []uuid.UUID
    for _, w := range wmss {
        wmsIDs = append(wmsIDs, w.ID)
    }
    ur.log.Debug("WmsIDs collected", "wmsIDs", wmsIDs)

    ur.log.Info("Fetching users by WmsIDs now (calling GetByWmsIDs internally)...")
    results, err := ur.GetByWmsIDs(ctx, transaction, wmsIDs)
    if err != nil {
        ur.log.Error("Failed to get users by Wmss", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by Wmss", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) GetByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.User, error) {
    ur.log.Info("Starting GetByCompanyIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User
    if len(companyIDs) == 0 {
        ur.log.Debug("No companyIDs provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("CompanyIDs provided", "count", len(companyIDs), "companyIDs", companyIDs)

    ur.log.Info("Fetching users by companyIDs now...")
    if err := transaction.WithContext(ctx).
        Where("company_id IN ?", companyIDs).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by companyIDs", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by companyIDs", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) GetByCompanies(ctx context.Context, tx *gorm.DB, companies []*types.Company) ([]*types.User, error) {
    ur.log.Info("Starting GetByCompanies now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(companies) == 0 {
        ur.log.Debug("No companies provided, returning empty slice")
        return nil, nil
    }
    ur.log.Debug("Companies provided", "count", len(companies), "companies", companies)

    ur.log.Info("Collecting companyIDs now...")
    var companyIDs []uuid.UUID
    for _, c := range companies {
        companyIDs = append(companyIDs, c.ID)
    }
    ur.log.Debug("CompanyIDs collected", "companyIDs", companyIDs)

    ur.log.Info("Fetching users by companyIDs (calling GetByCompanyIDs internally)...")
    results, err := ur.GetByCompanyIDs(ctx, transaction, companyIDs)
    if err != nil {
        ur.log.Error("Failed to get users by companies", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by companies", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) GetByRoleIDs(ctx context.Context, tx *gorm.DB, roleIDs []uuid.UUID) ([]*types.User, error) {
    ur.log.Info("Starting GetByRoleIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    var results []*types.User
    if len(roleIDs) == 0 {
        ur.log.Debug("No roleIDs provided, returning empty slice")
        return results, nil
    }
    ur.log.Debug("RoleIDs provided", "count", len(roleIDs), "roleIDs", roleIDs)

    ur.log.Info("Fetching users by roleIDs now...")
    if err := transaction.WithContext(ctx).
        Distinct("users.*").
        Where("role_id IN (?)", roleIDs).
        Find(&results).Error; err != nil {
        ur.log.Error("Failed to fetch users by roleIDs", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by roleIDs", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

func (ur *userRepo) GetByRoles(ctx context.Context, tx *gorm.DB, roles []*types.Role) ([]*types.User, error) {
    ur.log.Info("Starting GetByRoles now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(roles) == 0 {
        ur.log.Debug("No roles provided, returning empty slice")
        return nil, nil
    }
    ur.log.Debug("Roles provided", "count", len(roles), "roles", roles)

    ur.log.Info("Collecting roleIDs now...")
    var roleIDs []uuid.UUID
    for _, r := range roles {
        roleIDs = append(roleIDs, r.ID)
    }
    ur.log.Debug("RoleIDs collected", "roleIDs", roleIDs)

    ur.log.Info("Fetching users by roleIDs (calling GetByRoleIDs internally)...")
    results, err := ur.GetByRoleIDs(ctx, transaction, roleIDs)
    if err != nil {
        ur.log.Error("Failed to get users by roles", "error", err)
        return nil, err
    }
    ur.log.Info("Successfully fetched users by roles", "count", len(results))
    ur.log.Debug("Fetched users", "users", results)
    return results, nil
}

// ----------------------------------------------------------------
// SOFT DELETE
// ----------------------------------------------------------------

func (ur *userRepo) SoftDeleteByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) error {
    ur.log.Info("Starting SoftDeleteByUsers now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(users) == 0 {
        ur.log.Debug("No users provided, skipping soft delete")
        return nil
    }
    ur.log.Debug("Soft deleting by user slice", "count", len(users))

    var userIDs []uuid.UUID
    for _, u := range users {
        userIDs = append(userIDs, u.ID)
    }
    ur.log.Debug("Collected userIDs", "userIDs", userIDs)

    ur.log.Info("Performing soft delete by userIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", userIDs).
        Delete(&types.User{}).Error; err != nil {
        ur.log.Error("Failed to soft delete users by slice", "error", err)
        return err
    }
    ur.log.Info("Successfully soft deleted users by slice", "count", len(userIDs))
    return nil
}

func (ur *userRepo) SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error {
    ur.log.Info("Starting SoftDeleteByIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(userIDs) == 0 {
        ur.log.Debug("No userIDs provided, skipping soft delete")
        return nil
    }
    ur.log.Debug("Soft deleting by userIDs", "count", len(userIDs), "userIDs", userIDs)

    ur.log.Info("Performing soft delete by userIDs now...")
    if err := transaction.WithContext(ctx).
        Where("id IN (?)", userIDs).
        Delete(&types.User{}).Error; err != nil {
        ur.log.Error("Failed to soft delete users by IDs", "error", err)
        return err
    }
    ur.log.Info("Successfully soft deleted users by IDs", "count", len(userIDs))
    return nil
}

// ----------------------------------------------------------------
// FULL (HARD) DELETE
// ----------------------------------------------------------------

func (ur *userRepo) FullDeleteByUsers(ctx context.Context, tx *gorm.DB, users []*types.User) error {
    ur.log.Info("Starting FullDeleteByUsers now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(users) == 0 {
        ur.log.Debug("No users provided, skipping full delete")
        return nil
    }
    ur.log.Debug("Full deleting by user slice", "count", len(users))

    var userIDs []uuid.UUID
    for _, u := range users {
        userIDs = append(userIDs, u.ID)
    }
    ur.log.Debug("Collected userIDs", "userIDs", userIDs)

    ur.log.Info("Performing FULL (hard) delete by userIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", userIDs).
        Delete(&types.User{}).Error; err != nil {
        ur.log.Error("Failed to FULL delete users by slice", "error", err)
        return err
    }
    ur.log.Info("Successfully FULL deleted users by slice", "count", len(userIDs))
    return nil
}

func (ur *userRepo) FullDeleteByIDs(ctx context.Context, tx *gorm.DB, userIDs []uuid.UUID) error {
    ur.log.Info("Starting FullDeleteByIDs now...")

    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    if len(userIDs) == 0 {
        ur.log.Debug("No userIDs provided, skipping full delete")
        return nil
    }
    ur.log.Debug("Full deleting by userIDs", "count", len(userIDs), "userIDs", userIDs)

    ur.log.Info("Performing FULL (hard) delete by userIDs now...")
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN (?)", userIDs).
        Delete(&types.User{}).Error; err != nil {
        ur.log.Error("Failed to FULL delete users by IDs", "error", err)
        return err
    }
    ur.log.Info("Successfully FULL deleted users by IDs", "count", len(userIDs))
    return nil
}

// ----------------------------------------------------------------
// MISC - GET ME / DELETE ME
// ----------------------------------------------------------------

func (ur *userRepo) GetMe(ctx context.Context, tx *gorm.DB) (*types.User, error) {
    ur.log.Info("Starting GetMe now...")

    // 1) Transaction
    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    // 2) Grab request data
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        ur.log.Error("No request data in context, cannot get me!")
        return &types.User{}, fmt.Errorf("no request data found in context")
    }
    ur.log.Debug("Request data found for GetMe", "userID", rd.UserID)

    // 3) Query
    var user *types.User
    ur.log.Info("Fetching current user by rd.UserID now...")
    if err := transaction.WithContext(ctx).
        Where("id = ?", rd.UserID).
        First(&user).Error; err != nil {
        ur.log.Error("Failed to fetch current user (GetMe)", "error", err, "userID", rd.UserID)
        return user, err
    }
    ur.log.Info("Successfully fetched current user (GetMe)", "userID", rd.UserID)
    ur.log.Debug("User fetched details", "user", user)
    return user, nil
}

func (ur *userRepo) DeleteMe(ctx context.Context, tx *gorm.DB) error {
    ur.log.Info("Starting DeleteMe now...")

    // 1) Transaction
    transaction := tx
    if transaction == nil {
        transaction = ur.db
        ur.log.Debug("Transaction is nil, using ur.db", "db", transaction)
    }

    // 2) Grab request data
    rd := requestdata.GetRequestData(ctx)
    if rd == nil {
        ur.log.Error("No request data in context, cannot delete me!")
        return fmt.Errorf("no request data found in context")
    }
    ur.log.Debug("Request data found for DeleteMe", "userID", rd.UserID)

    // 3) Soft delete (or full, depending on your preference)
    ur.log.Info("Performing soft delete on current user now...", "userID", rd.UserID)
    if err := transaction.WithContext(ctx).
        Where("id = ?", rd.UserID).
        Delete(&types.User{}).Error; err != nil {
        ur.log.Error("Failed to delete current user (DeleteMe)", "error", err, "userID", rd.UserID)
        return err
    }
    ur.log.Info("Successfully soft deleted current user (DeleteMe)", "userID", rd.UserID)
    return nil
}

