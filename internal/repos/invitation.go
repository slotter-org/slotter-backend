package repos

import (
    "context"
    "time"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    
    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type InvitationRepo interface {
    Create(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) ([]*types.Invitation, error)

    GetByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) ([]*types.Invitation, error)
    GetByTokens(ctx context.Context, tx *gorm.DB, tokens []string) ([]*types.Invitation, error)
    GetByEmails(ctx context.Context, tx *gorm.DB, emails []string) ([]*types.Invitation, error)
    GetByPhoneNumbers(ctx context.Context, tx *gorm.DB, phones []string) ([]*types.Invitation, error)

    GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Invitation, error)
    GetByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.Invitation, error)

    Update(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) ([]*types.Invitation, error)
    //MarkStatus(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, status types.InvitationStatus) error
    //MarkAccepted(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, acceptedAt time.Time) error
    //MarkCanceled(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, canceledAt time.Time) error

    SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) error
    SoftDeleteByInvitations(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) error

    FullDeleteByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) error
    FullDeleteByInvitations(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) error

    BulkExpireInvitations(ctx context.Context, tx *gorm.DB) (int64, error)
}

type invitationRepo struct {
    db      *gorm.DB
    log     *logger.Logger
}

func NewInvitationRepo(db *gorm.DB, baseLog *logger.Logger) InvitationRepo {
    repoLog := baseLog.With("repo", "InvitationRepo")
    return &invitationRepo{db: db, log: repoLog}
}

func (ir *invitationRepo) Create(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.Create started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    if len(invites) == 0 {
        ir.log.Debug("No invitations provided, returning empty slice")
        return []*types.Invitation{}, nil
    }
    if err := transaction.WithContext(ctx).Create(&invites).Error; err != nil {
        ir.log.Error("Failed to create invitations", "error", err)
        return nil, err
    }
    ir.log.Info("Successfully created invitations", "count", len(invites))
    return invites, nil
} 

func (ir *invitationRepo) GetByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.GetByIDs started")
    
    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(inviteIDs) == 0 {
        ir.log.Debug("No inviteIDs provided, returning empty slice")
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Preload("Wms").
        Preload("Company").
        Where("id IN ?", inviteIDs).
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by IDs", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by IDs", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) GetByTokens(ctx context.Context, tx *gorm.DB, tokens []string) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.GetByTokens started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(tokens) == 0 {
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("token IN ?", tokens).
        Preload("Wms").
        Preload("Company").
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by tokens", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by tokens", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) GetByEmails(ctx context.Context, tx *gorm.DB, emails []string) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.GetByEmails started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(emails) == 0 {
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("email IN ?", emails).
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by emails", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by emails", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) GetByPhoneNumbers(ctx context.Context, tx *gorm.DB, phones []string) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.GetByPhoneNumbers started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(phones) == 0 {
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("phone_number IN ?", phones).
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by phone numbers", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by phone numbers", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) GetByWmsIDs(ctx context.Context, tx *gorm.DB, wmsIDs []uuid.UUID) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.GetByWmsIDs started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(wmsIDs) == 0 {
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("wms_id IN ?", wmsIDs).
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by wmsIDs", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by wmsIDs", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) GetByCompanyIDs(ctx context.Context, tx *gorm.DB, companyIDs []uuid.UUID) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo...GetByCompanyIDs started")
    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var results []*types.Invitation
    if len(companyIDs) == 0 {
        return results, nil
    }
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("company_id IN ?", companyIDs).
        Find(&results).Error; err != nil {
        ir.log.Error("Failed to fetch invitations by companyIDs", "error", err)
        return nil, err
    }
    ir.log.Info("Fetched invitations by companyIDs", "count", len(results))
    return results, nil
}

func (ir *invitationRepo) Update(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) ([]*types.Invitation, error) {
    ir.log.Info("InvitationRepo.Update started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    if len(invites) == 0 {
        ir.log.Debug("No invites provided; returning empty")
        return invites, nil
    }
    for i := range invites {
        if err := transaction.WithContext(ctx).Save(&invites[i]).Error; err != nil {
            ir.log.Error("Failed to update invites", "error", err, "invite", invites[i])
            return nil, err
        }
    }
    ir.log.Info("Updated invitations", "count", len(invites))
    return invites, nil
}

/*
func (ir *invitationRepo) MarkStatus(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, status types.InvitationStatus) error {
    ir.log.Info("InvitationRepo.MarkStatus started", "inviteID", inviteID, "status", status)

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }

    var inv types.Invitation
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("id = ?", inviteID).
        First(&inv).Error; err != nil {
        return err
    }
    inv.Status = status
    switch status {
    case types.InvitationStatusAccepted:
        inv.AcceptedAt = &time.Now()
    case types.InvitationStatusCanceled:
        inv.CanceledAt = &time.Now()
    case types.InvitationStatusExpired:
        inv.CanceledAt = &time.Now()
    default:

    }
    if err := transaction.WithContext(ctx).Save(&inv).Error; err != nil {
        return err
    }
    return nil
}

func (ir *invitationRepo) MarkAccepted(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, acceptedAt time.Time) error {
    ir.log.Info("InvitationRepo.MarkAccepted started", "inviteID", inviteID)

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var inv types.Invitation
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("id = ?", inviteID).
        First(&inv).Error; err != nil {
        return err
    }
    inv.Status = types.InvitationStatusAccepted
    if acceptedAt.IsZero() {
        inv.AcceptedAt = &time.Now()
    } else {
        inv.AcceptedAt = &acceptedAt
    }
    if err := transaction.WithContext(ctx).Save(&inv).Error; err != nil {
        return err
    }
    return nil
}

func (ir *invitationRepo) MarkCanceled(ctx context.Context, tx *gorm.DB, inviteID uuid.UUID, canceledAt time.Time) error {
    ir.log.Info("InvitationRepo.MarkCanceled started", "inviteID", inviteID)

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }
    var inv types.Invitation
    if err := transaction.WithContext(ctx).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("id = ?", inviteID).
        First(&inv).Error; err != nil {
        return err
    }
    inv.Status = types.InvitationStatusCanceled
    if canceledAt.IsZero() {
        inv.CanceledAt = &time.Now()
    } else {
        inv.CanceledAt = canceledAt
    }
    if err := transaction.WithContext(ctx).Save(&inv).Error; err != nil {
        return err
    }
    return nil
}
*/
func (ir *invitationRepo) SoftDeleteByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) error {
    ir.log.Info("InvitationRepo.SoftDeleteByIDs started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }

    if len(inviteIDs) == 0 {
        ir.log.Debug("No inviteIDs provided; skipping soft delete")
        return nil
    }
    if err := transaction.WithContext(ctx).
        Where("id IN ?", inviteIDs).
        Delete(&types.Invitation{}).Error; err != nil {
        ir.log.Error("Failed to soft delete invites by IDs", "error", err)
        return err
    }
    ir.log.Info("Soft deleted invitations by IDs", "count", len(inviteIDs))
    return nil
}

func (ir *invitationRepo) SoftDeleteByInvitations(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) error {
    ir.log.Info("InvitationRepo.SoftDeleteByInvitations started")

    if len(invites) == 0 {
        return nil
    }
    var ids []uuid.UUID
    for _, iv := range invites {
        ids = append(ids, iv.ID)
    }
    return ir.SoftDeleteByIDs(ctx, tx, ids)
}

func (ir *invitationRepo) FullDeleteByIDs(ctx context.Context, tx *gorm.DB, inviteIDs []uuid.UUID) error {
    ir.log.Info("InvitationRepo.FullDeleteByIDs started")

    transaction := tx
    if transaction == nil {
        transaction = ir.db
    }

    if len(inviteIDs) == 0 {
        ir.log.Debug("No inviteIDs provided; skipping full delete")
        return nil
    }
    if err := transaction.WithContext(ctx).
        Unscoped().
        Where("id IN ?", inviteIDs).
        Delete(&types.Invitation{}).Error; err != nil {
        ir.log.Error("Failed to FULL delete invites by IDs", "error", err)
        return err
    }
    ir.log.Info("Full deleted invitations by IDs", "count", len(inviteIDs))
    return nil
}

func (ir *invitationRepo) FullDeleteByInvitations(ctx context.Context, tx *gorm.DB, invites []*types.Invitation) error {
    if len(invites) == 0 {
        return nil
    }
    var ids []uuid.UUID
    for _, iv := range invites {
        ids = append(ids, iv.ID)
    }
    return ir.FullDeleteByIDs(ctx, tx, ids)
}

func (ir *invitationRepo) BulkExpireInvitations(ctx context.Context, tx *gorm.DB) (int64, error) {
    ir.log.Info("InvitationRepo.BulkExpireInvitations started")

    db := tx
    if db == nil {
        db = ir.db
    }
    now := time.Now()
    result := db.WithContext(ctx).
        Model(&types.Invitation{}).
        Where("status = ? AND expires_at <= ?", types.InvitationStatusPending, now).
        Updates(map[string]interface{}{
            "status": types.InvitationStatusExpired,
            "expired_at": now,
        })
    if result.Error != nil {
        ir.log.Error("Failed to bulk expire invitations", "error", result.Error)
        return 0, result.Error
    }
    rowsAffected := result.RowsAffected
    ir.log.Info("BulkExpireInvitations updated invitations", "count", rowsAffected)
    return rowsAffected, nil
}


