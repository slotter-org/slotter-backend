package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/slotter-org/slotter-backend/internal/logger"
	"github.com/slotter-org/slotter-backend/internal/requestdata"
	"github.com/slotter-org/slotter-backend/internal/repos"
	"github.com/slotter-org/slotter-backend/internal/templates"
	"github.com/slotter-org/slotter-backend/internal/types"
)

type InvitationService interface {
	SendInvitation(ctx context.Context, inv *types.Invitation) error
	UpdateInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName,newMessage string) (*types.Invitation, error)
	updateInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName, newMessage string) (*types.Invitation, error) 
	canUpdateInvitation(inv *types.Invitation) bool
	UpdateInvitationRole(ctx context.Context, tx *gorm.DB, invID uuid.UUID, roleID uuid.UUID) (*types.Invitation, error)
	updateInvitationRoleLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID, roleID uuid.UUID) (*types.Invitation, error)
	canUpdateInvitationRole(inv *types.Invitation) bool
	CancelInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	cancelInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	canCancelInvitation(inv *types.Invitation) bool
	ResendInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	resendInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	canResendInvitation(inv *types.Invitation) bool
	DeleteInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error
	deleteInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error
	canDeleteInvitation(inv *types.Invitation) bool
}

type invitationService struct {
	db									*gorm.DB
	log									*logger.Logger
	invitationRepo			repos.InvitationRepo
	userRepo						repos.UserRepo
	wmsRepo							repos.WmsRepo
	companyRepo					repos.CompanyRepo
	roleRepo						repos.RoleRepo
	permissionRepo			repos.PermissionRepo
	textService					TextService
	emailService				EmailService
	brandLogoPath				string
	frontEndURL					string
}

func NewInvitationService(
	db									*gorm.DB,
	log									*logger.Logger,
	invitationRepo			repos.InvitationRepo,
	userRepo						repos.UserRepo,
	wmsRepo							repos.WmsRepo,
	companyRepo					repos.CompanyRepo,
	roleRepo						repos.RoleRepo,
	permissionRepo			repos.PermissionRepo,
	textService					TextService,
	emailService				EmailService,
) InvitationService {
	serviceLog := log.With("service", "InvitationService")
	rawLogoPath := os.Getenv("SLOTTER_BRAND_LOGO_PATH")
	var finalLogoBase64 string
	if rawLogoPath != "" {
		base64Logo, err := readFileAsBase64(rawLogoPath)
		if err != nil {
			serviceLog.Warn("Failed to read or encode brand logo from SLOTTER_LOGO_URL; using fallback HTTP link", "error", err)
			finalLogoBase64 = "https://slotter.ai/slotter-logo.png"
		} else {
			finalLogoBase64 = base64Logo
			serviceLog.Debug("Using base64-encoded brand logo from SLOTTER_BRAND_LOGO_PATH")
		}
	} else {
		serviceLog.Warn("SLOTTER_BRAND_LOGO_PATH not set; using fallback HTTP link.")
		finalLogoBase64 = "https://slotter.ai/slotter-logo.png"
	}
	frontEndURL := os.Getenv("SLOTTER_FRONT_END_URL")
	if frontEndURL == "" {
		frontEndURL = "http://localhost:3000"
		serviceLog.Warn("SLOTTER_FRONT_END_URL not set; using faillback front end URL.")
	}
	return &invitationService{
		db:								db,
		log:							serviceLog,
		invitationRepo:		invitationRepo,
		userRepo:					userRepo,
		wmsRepo:					wmsRepo,
		companyRepo:			companyRepo,
		roleRepo:					roleRepo,
		permissionRepo:		permissionRepo,
		emailService:			emailService,
		textService:			textService,
		brandLogoPath:		finalLogoBase64,
		frontEndURL:			frontEndURL,
	}
}

func (is *invitationService) SendInvitation(ctx context.Context, inv *types.Invitation) error {
	if inv == nil {
		return fmt.Errorf("invitation is nil")
	}
	rd := requestdata.GetRequestData(ctx)
	if rd == nil {
		is.log.Warn("Request Data not set in context, Cannot proceed.")
		return fmt.Errorf("Request Data not set in context.")
	}
	if rd.UserID == uuid.Nil {
		is.log.Warn("UserID not set in request data, Cannot proceed.")
		return fmt.Errorf("UserID not set in request data.")
	}

	//1) Get user and ensure that they have manage_invitations permission
	foundUsers, ufErr := is.userRepo.GetByIDs(ctx, nil, []uuid.UUID{rd.UserID})
	if ufErr != nil {
		is.log.Warn("Error fetching user by ID", "error", ufErr)
		return fmt.Errorf("failed to fetch userr by ID: %w", ufErr)
	}
	if len(foundUsers) == 0 {
		is.log.Warn("No user found with that ID", "id", rd.UserID)
		return fmt.Errorf("no user found with that ID.")
	}
	user := foundUsers[0]
	if user.RoleID == nil || *user.RoleID == uuid.Nil {
		is.log.Warn("User has no role assigned.", "roleID", user.RoleID)
		return fmt.Errorf("User has no role assigned.")
	}
	foundRoles, rfErr := is.roleRepo.GetByIDs(ctx, nil, []uuid.UUID{*user.RoleID})
	if rfErr != nil {
		is.log.Warn("Error fetching user role by ID.", "error", rfErr)
		return fmt.Errorf("Error fetching user role by ID: %w", rfErr)
	}
	if len(foundRoles) == 0 {
		is.log.Warn("No role found with the given role ID.", "count", len(foundRoles))
		return fmt.Errorf("No role found with that ID.")
	}
	role := foundRoles[0]
	var hasPermission bool
	for _, perm := range role.Permissions {
		if perm.PermissionType == "manage_invitations" {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		is.log.Warn("User does not have permission to manage invitations.")
		return fmt.Errorf("User does not have permission to mangage invitations.")
	}

	//2) Determine whether inviting via phone or email. Exactly one must be set.
	var inviteMethod string
	if inv.Email != nil && *inv.Email != "" && inv.PhoneNumber != nil && *inv.PhoneNumber != "" {
		return fmt.Errorf("Cannot have both email and phone number set for an invitation.")
	} else if (inv.Email == nil || *inv.Email == "") && (inv.PhoneNumber == nil || *inv.PhoneNumber == "") {
		return fmt.Errorf("Must provide either an email or phone number for invitation.")
	} else if inv.Email != nil && *inv.Email != "" {
		inviteMethod = "email"
	} else {
		inviteMethod = "phone"
	}

	//3) Validate InvitationType against user type
	switch user.UserType {
	case "wms":
		if inv.InvitationType != types.InvitationTypeJoinWms && inv.InvitationType != types.InvitationTypeJoinWmsWithNewCompany {
			return fmt.Errorf("Invalid invitation type for Wms User: %s", inv.InvitationType)
		}
		inv.WmsID = user.WmsID
	case "company":
		if inv.InvitationType != types.InvitationTypeJoinCompany {
			return fmt.Errorf("Invalid invitation type for Company User: %s", inv.InvitationType)
		}
		inv.CompanyID = user.CompanyID
	default:
		return fmt.Errorf("Unknown user type: %s", user.UserType)
	}

	//4) Transaction to check if user/email/phone already exist or if there is a pending invite for that email/phone with the same wms/company
	err := is.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//A) Check if there is already a pending invitation for the same Wms/Company + email/phone
		if inviteMethod == "email" {
			existingInvs, getErr := is.invitationRepo.GetByEmails(ctx, tx, []string{*inv.Email})
			if getErr != nil {
				is.log.Warn("Error fetching existing invitations by email", "error", getErr)
				return fmt.Errorf("Failed checking existing invitations by email: %w", getErr)
			}
			for _, eInv := range existingInvs {
				if eInv.Status == types.InvitationStatusPending && eInv.WmsID == inv.WmsID && eInv.CompanyID == inv.CompanyID {
					return fmt.Errorf("There is already a pending invitation for that email under this Wms/Company")
				}
			}
		} else {
			existingInvs, getErr := is.invitationRepo.GetByPhoneNumbers(ctx, tx, []string{*inv.PhoneNumber})
			if getErr != nil {
				is.log.Warn("Error fetching existing invitations by phone number", "error", getErr)
				return fmt.Errorf("Failed checking existing invitations by phone number: %w", getErr)
			}
			for _, pInv := range existingInvs {
				if pInv.Status == types.InvitationStatusPending && pInv.WmsID == inv.WmsID && pInv.CompanyID == inv.CompanyID {
					return fmt.Errorf("There is already a pending invitation for that phone number under this Wms/Company")
				}
			}
		}
		//B) If inviting by email, ensure no user with that email already exists
		if inviteMethod == "email" {
			emailExists, eErr := is.userRepo.EmailExists(ctx, tx, *inv.Email)
			if eErr != nil {
				is.log.Warn("Error checking email existence", "error", eErr)
				return fmt.Errorf("Failed checking email existence: %w", eErr)
			}
			if emailExists {
				return fmt.Errorf("That email is already in use.")
			}
		} else {
			//C) If inviting by phone, ensure no user with that phone exists
			phoneExists, pErr := is.userRepo.PhoneNumberExists(ctx, tx, *inv.PhoneNumber)
			if pErr != nil {
				is.log.Warn("Error checking phone number existence", "error", pErr)
				return fmt.Errorf("Failed checking phone number existence: %w", pErr)
			}
			if phoneExists {
				return fmt.Errorf("That phone number is already in use.")
			}
		}

		//5) Fill out invitation fields
		inv.InviteUserID = user.ID
		inv.Status = types.InvitationStatusPending
		if inv.Token == "" {
			inv.Token = uuid.NewString()
		}
		if inv.ExpiresAt.IsZero() {
			inv.ExpiresAt = time.Now().Add(48 * time.Hour)
		}

		//6) Create in DB
		_, cErr := is.invitationRepo.Create(ctx, tx, []*types.Invitation{inv})
		if cErr != nil {
			return fmt.Errorf("Failed to create invitation: %w", cErr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	//7) Post-transaction send (so the DB changes are not rolled back if sending fails)
	linkURL := fmt.Sprintf("%s/register?token=%s", is.frontEndURL, inv.Token)
	if inviteMethod == "email" {
		var finalAvatarURL string
		if inv.WmsID != nil || *inv.WmsID != uuid.Nil {
			foundWms, _ := is.wmsRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.WmsID})
			if len(foundWms) > 0 {
				finalAvatarURL = foundWms[0].AvatarURL
			}
		}
		if inv.CompanyID != nil || *inv.CompanyID != uuid.Nil {
			foundCompany, _ := is.companyRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.CompanyID})
			if len(foundCompany) > 0 {
				finalAvatarURL = foundCompany[0].AvatarURL
			}
		}
		templateData := templates.InvitationEmailData{
			Logo:						is.brandLogoPath,
			InvitationLink: linkURL,
			AvatarURL:			finalAvatarURL,
			InvitationType:	templates.InvitationType(string(inv.InvitationType)),
			WmsName:				"",
			CompanyName:		"",
		}
		if inv.WmsID != nil || *inv.WmsID != uuid.Nil {
			foundWms, _ := is.wmsRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.WmsID})
			if len(foundWms) > 0 {
				templateData.WmsName = foundWms[0].Name
			}
		}
		if inv.CompanyID != nil || *inv.CompanyID != uuid.Nil {
			foundCompany, _ := is.companyRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.CompanyID})
			if len(foundCompany) > 0 {
				templateData.CompanyName = foundCompany[0].Name
			}
		}
		htmlContent, tplErr := templates.RenderInvitationHTML(templateData)
		if tplErr != nil {
			is.log.Warn("Failed to render invitation HTML template", "error", tplErr)
			return tplErr
		}
		plainText := fmt.Sprintf("You have been invited to join Slotter! Click here: %s", linkURL)
		subject := "You've Been Invited to Slotter!"

		if err := is.emailService.SendEmail(ctx, *inv.Email, subject, plainText, htmlContent, "invitation"); err != nil {
			is.log.Warn("Failed to send invitation email", "error", err)
			return err
		}
	} else {
		textBody := fmt.Sprintf("Slotter invitation! Click here: %s", linkURL)
		if err := is.textService.SendText(ctx, *inv.PhoneNumber, textBody); err != nil {
			is.log.Warn("Failed to send invitation text", "error", err)
			return err
		}
	}
	return nil
}

func readFileAsBase64(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	var mimeType string
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".svg":
		mimeType = "image/svg+xml"
	default:
		mimeType = "image/png"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + mimeType + ";base64," + encoded, nil
}

func (is *invitationService) UpdateInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName, newMessage string) (*types.Invitation, error) {
	if tx == nil {
		var out *types.Invitation
		err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm) error {
			updated, uErr := is.updateInvitationLogic(ctx, innerTx, invID, newName, newMessage)
			if uErr != nil {
				return uErr
			}
			out = updated
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return is.updateInvitationLogic(ctx, tx, invID, newName, newMessage)
}

func (is *invitationService) updateInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName, newMessage string) (*types.Invitation, error) {
	if invID == uuid.Nil {
		return nil, fmt.Errorf("invalid invitation id")
	}
	existing, err := is.invitationRepo.GetByIDs(ctx, tx, []uuid.UUID{invID})
	if err != nil || len(existing) == 0 {
		return nil, fmt.Errorf("invitation not found")
	}
	inv := existing[0]
	if !is.canUpdateInvitation(inv) {
		return nil, fmt.Errorf("invitation cannot be updated in status: %s", inv.Status)
	}
	if newName != nil && newName != "" {
		inv.Name = &newName
	}
	if newMessage != nil && newMessage != "" {
		inv.Message = &newMessage
	}
	updated, err := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if err != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation: %w", err)
	}
	return updated[0], nil
}

func (is *invitationService) canUpdateInvitation(inv *types.Invitation) bool {
	switch inv.Status {
	case is.invitationRepo.InvitationStatusPending:
		return true
	default:
		return false
	}
}

func (is *invitationService) UpdateInvitationRole(ctx context.Context, tx *gorm.DB, invID, roleID uuid.UUID) (*types.Invitation, error) {
	if tx == nil {
		var out *types.Invitation
		err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
			updated, upErr := is.updateInvitationRoleLogic(ctx, innerTx, invID, roleID)
			if upErr != nil {
				return upErr
			}
			out = updated
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return is.updateInvitationRoleLogic(ctx, tx, invID, roleID)
}

func (is *invitationService) updateInvitationRoleLogic(ctx context.Context, tx *gorm.DB, invID, roleID uuid.UUID) (*types.Invitation, error) {
	existing, err := is.invitationRepo.GetByIDs(ctx, tx, []uuid.UUID{invID})
	if err != nil || len(existing) == 0 {
		return nil, fmt.Errorf("invitation not found")
	}
	inv := existing[0]
	if !is.canUpdateInvitation(inv) {
		return nil, fmt.Errorf("invitation cannot update role in status: %s", inv.Status)
	}
	rd := requestdata.GetRequestData(ctx)
	if rd == nil {
		return nil, fmt.Errorf("request data missing in context")
	}
	if roleID == uuid.Nil {
		return nil, fmt.Errorf("invalid role id")
	}
	if inv.WmsID != nil && *inv.WmsID != uuid.Nil {
	roles, rErr := is.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{roleID})
		if rErr != nil || len(roles) == 0 {
			return nil, fmt.Errorf("role not found")
		}
		theRole := roles[0]
		if theRole.WmsID == nil || *theRole.WmsID != *inv.WmsID {
			return nil, fmt.Errorf("role does not belong to the same wms as the invitation")
		}
	} else if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
	roles, rErr := is.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{roleID})
		if rErr != nil || len(roles) == 0 {
			return nil, fmt.Errorf("role not found")
		}
		theRole := roles[0]
		if theRole.CompanyID == nil || *theRole.CompanyID != *inv.CompanyID {
			return nil, fmt.Errorf("role does not belong to the same company as the invitation")
		}
	} else {
		return nil, fmt.Errorf("invitation has no wmsID/companyID, cannot proceed")
	}
	inv.RoleID = &roleID
	updated, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if upErr != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation role: %w", upErr)
	}
	return updated[0], nil
}

func (is *invitationService) CancelInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error) {
	if tx == nil {
		var out *types.Invitation
		err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
			updated, cErr := is.cancelInvitationLogic(ctx, innerTx, invID)
			if cErr != nil {
				return cErr
			}
			out = updated
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return is.cancelInvitationLogic(ctx, tx, invID)
}

func (is *invitationService) cancelInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error) {
	if invID == uuid.Nil {
		return nil, fmt.Errorf("invalid invtitation ID")
	}
	existing, err := is.invitationRepo.GetByIDs(ctx, tx, []uuid.UUID{invID})
	if err != nil || len(existing) == 0 {
		return nil, fmt.Errorf("invitation not found")
	}
	inv := exisiting[0]
	if !is.canCancelInvitation(inv) {
		return nil, fmt.Errorf("cannot cancel invitation in status: %s", inv.Status)
	}
	inv.Status = is.invitationRepo.InvitationStatusCanceled
	inv.ExpiresAt = time.Time{}
	now := time.Now()
	inv.CanceledAt = &now
	updated, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if upErr != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation as canceled: %w", upErr)
	}
	return updated[0], nil
}

func (is *invitationService) canCancelInvitation(inv *types.Invitation) bool {
	if inv.Status == is.invitationRepo.InvitationStatusPending {
		return true
	}
	return false
}

func (is *invitationService) ResendInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error) {
	if tx == nil {
		var out *types.Invitation
		err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
			updated, reErr := is.resendInvitationLogic(ctx, innerTx, invID)
			if reErr != nil {
				return reErr
			}
			out = updated
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return is.resendInvitationLogic(ctx, tx, invID)
}

func (is *invitationService) resendInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error) {
	if invID == uuid.Nil {
		return nil, fmt.Errorf("invalid invitation ID")
	}
	existing, err := is.invitationRepo.GetByIDs(ctx, tx, []uuid.UUID{invID})
	if err != nil || len(existing) == 0 {
		return nil, fmt.Errorf("invitation not found")
	}
	inv := existing[0]
	if inv.Status == is.invitationRepo.InvitationStatusAccepted {
		return nil, fmt.Errorf("cannot resend an already accepted invitation")
	}
	if inv.Status == is.invitationRepo.InvitationStatusPending {
		return nil, fmt.Errorf("cannot resend an invitation that is still pending")
	}
	if inv.Status == is.invitationRepo.InvitationStatusCanceled || inv.Status == is.invitationRepo.InvitationStatusRejected || inv.Status == is.invitationRepo.InvitationStatusExpired {
		inv.CanceledAt = nil
		inv.RejectedAt = nil
		inv.ExpiredAt = nil
	}
	inv.Status = is.invitationRepo.InvitationStatusPending
	inv.Token = uuid.NewString()
	inv.ExpiresAt = time.Now().Add(48 * time.Hour)
	inv.CreatedAt = time.Now()
	updated, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if upErr != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation for resend: %w", upErr)
	}
	final := updated[0]
	linkURL := fmt.Sprintf("%s/register?token=%s", is.frontEndURL, final.Token)
	if final.Email != nil && *final.Email != "" {
		if eErr := is.sendInvitationEmail(ctx, *final.Email, linkURL, final); eErr != nil {
			return nil, eErr
		}
	} else if final.PhoneNumber != nil && *final.PhoneNumber != "" {
		textBody := fmt.Sprintf("Re-sent invitation link: %s", linkURL)
		if tErr := is.textService.SendText(ctx, *final.PhoneNumber, textBody); tErr != nil {
			return nil, tErr
		}
	}
	return final, nil
}

func (is *invitationService) DeleteInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error {
	if tx == nil {
		return is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
			return is.deleteInvitationLogic(ctx, innerTx, invID)
		})
	}
	return is.deleteInvitationLogic(ctx, tx, invID)
}

func (is *invitationService) deleteInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error {
	if invID == uuid.Nil {
		return fmt.Errorf("invalid invitation ID")
	}
	existing, err := is.invitationRepo.GetByIDs(ctx, tx, []uuid.UUID{invID})
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("invitation not found")
	}
	inv := existing[0]
	if !is.canDeleteInvitation(inv) {
		return fmt.Errorf("invitation is not in a deletable status: %s", inv.Status)
	}
	return is.invitationRepo.SoftDeleteByInvitations(ctx, tx, []*types.Invitation{inv})
}

func (is *invitationService) canDeleteInvitation(inv *types.Invitation) bool {
	switch inv.Status {
	case is.invitationRepo.InvitationStatusAccepted,
			 is.invitationRepo.InvitationStatusCanceled,
			 is.invitationRepo.InvitationStatusRejected,
			 is.invitationRepo.InvitationStatusCanceled,
			 is.invitationRepo.InvitationStatusExpired:
		return true
	}
	return false
}
