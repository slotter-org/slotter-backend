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
	"github.com/slotter-org/slotter-backend/internal/sse"
	"github.com/slotter-org/slotter-backend/internal/ssedata"
	"github.com/slotter-org/slotter-backend/internal/repos"
	"github.com/slotter-org/slotter-backend/internal/templates"
	"github.com/slotter-org/slotter-backend/internal/types"
)

type InvitationService interface {
	SendInvitation(ctx context.Context, tx *gorm.DB, inv *types.Invitation) error
	sendInvitationLogic(ctx context.Context, tx *gorm.DB, inv *types.Invitation) (*types.Invitation, error)
	sendInvitationOutbound(ctx context.Context, inv *types.Invitation) error
	UpdateInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName,newMessage string) (*types.Invitation, error)
	updateInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID, newName, newMessage string) (*types.Invitation, error) 
	canUpdateInvitation(inv *types.Invitation) bool
	UpdateInvitationRole(ctx context.Context, tx *gorm.DB, invID uuid.UUID, roleID uuid.UUID) (*types.Invitation, error)
	updateInvitationRoleLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID, roleID uuid.UUID) (*types.Invitation, error)
	CancelInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	cancelInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	canCancelInvitation(inv *types.Invitation) bool
	ResendInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	resendInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error)
	DeleteInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error
	deleteInvitationLogic(ctx context.Context, tx *gorm.DB, invID uuid.UUID) error 
	canDeleteInvitation(inv *types.Invitation) bool
	ValidateInvitationToken(ctx context.Context, tx *gorm.DB, token string) (*types.Invitation, error)
	ExpirePendingInvitations(ctx context.Context, tx *gorm.DB) (int64, error)
	expirePendingInvitationsLogic(ctx context.Context, tx *gorm.DB) (int64, error)
	getInvitationSSEChannel(inv *types.Invitation) string
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
	avatarService				AvatarService
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
	avatarService				AvatarService,
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
		avatarService:    avatarService,
		brandLogoPath:		finalLogoBase64,
		frontEndURL:			frontEndURL,
	}
}

func (is *invitationService) SendInvitation(ctx context.Context, tx *gorm.DB, inv *types.Invitation) error {
	// If tx is provided, just use it and do everything inline.
	// Otherwise, create a new transaction block.
	if tx != nil {
		// Do transaction-bound work inside the provided tx
		finalInv, err := is.sendInvitationLogic(ctx, tx, inv)
		if err != nil {
			return err
		}
		// Now that DB changes are done, send the actual invitation (email or text) 
		return is.sendInvitationOutbound(ctx, finalInv)
	}

	// No tx provided. Create our own.
	var finalInv *types.Invitation
	err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
		localInv, logicErr := is.sendInvitationLogic(ctx, innerTx, inv)
		if logicErr != nil {
			return logicErr
		}
		finalInv = localInv
		return nil
	})
	if err != nil {
		return err
	}

	// Post-transaction: send email or text
	return is.sendInvitationOutbound(ctx, finalInv)
}

// sendInvitationLogic performs all the DB operations in the transaction:
//  1) Validates the requesting user + perms
//  2) Checks for existing invites or existing user phone/email
//  3) Creates the Invitation
//  4) Generates/uploads the Invitation avatar
//  5) Updates the Invitation record with the avatar fields
func (is *invitationService) sendInvitationLogic(ctx context.Context, tx *gorm.DB, inv *types.Invitation) (*types.Invitation, error) {

	// 0) Basic nil checks
	if inv == nil {
		return nil, fmt.Errorf("invitation is nil")
	}

	rd := requestdata.GetRequestData(ctx)
	if rd == nil {
		is.log.Warn("Request Data not set in context, Cannot proceed.")
		return nil, fmt.Errorf("Request Data not set in context.")
	}
	if rd.UserID == uuid.Nil {
		is.log.Warn("UserID not set in request data, Cannot proceed.")
		return nil, fmt.Errorf("UserID not set in request data.")
	}

	// 1) Get user and verify manage_invitations permission
	foundUsers, ufErr := is.userRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.UserID})
	if ufErr != nil {
		is.log.Warn("Error fetching user by ID", "error", ufErr)
		return nil, fmt.Errorf("failed to fetch user by ID: %w", ufErr)
	}
	if len(foundUsers) == 0 {
		is.log.Warn("No user found with that ID", "id", rd.UserID)
		return nil, fmt.Errorf("no user found with that ID")
	}
	user := foundUsers[0]
	if user.RoleID == nil || *user.RoleID == uuid.Nil {
		is.log.Warn("User has no role assigned.", "roleID", user.RoleID)
		return nil, fmt.Errorf("user has no role assigned")
	}
	foundRoles, rfErr := is.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{*user.RoleID})
	if rfErr != nil {
		is.log.Warn("Error fetching user role by ID.", "error", rfErr)
		return nil, fmt.Errorf("error fetching user role by ID: %w", rfErr)
	}
	if len(foundRoles) == 0 {
		is.log.Warn("No role found with the given role ID.", "roleID", *user.RoleID)
		return nil, fmt.Errorf("no role found with that ID")
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
		return nil, fmt.Errorf("user does not have permission to manage invitations")
	}

	// 2) Exactly one of Email or Phone must be set
	var inviteMethod string
	if inv.Email != nil && *inv.Email != "" && inv.PhoneNumber != nil && *inv.PhoneNumber != "" {
		return nil, fmt.Errorf("cannot have both email and phone set for invitation")
	} else if (inv.Email == nil || *inv.Email == "") && (inv.PhoneNumber == nil || *inv.PhoneNumber == "") {
		return nil, fmt.Errorf("must provide either email or phone for invitation")
	} else if inv.Email != nil && *inv.Email != "" {
		inviteMethod = "email"
	} else {
		inviteMethod = "phone"
	}

	// 3) Validate InvitationType against the user type
	switch user.UserType {
	case "wms":
		if inv.InvitationType != types.InvitationTypeJoinWms && 
		   inv.InvitationType != types.InvitationTypeJoinWmsWithNewCompany {
			return nil, fmt.Errorf("invalid invitation type for WMS user: %s", inv.InvitationType)
		}
		inv.WmsID = user.WmsID

	case "company":
		if inv.InvitationType != types.InvitationTypeJoinCompany {
			return nil, fmt.Errorf("invalid invitation type for Company user: %s", inv.InvitationType)
		}
		inv.CompanyID = user.CompanyID

	default:
		return nil, fmt.Errorf("unknown user type: %s", user.UserType)
	}

	// 4) Check for existing invites + user phone/email existence + create record
	if inviteMethod == "email" {
		existingInvs, getErr := is.invitationRepo.GetByEmails(ctx, tx, []string{*inv.Email})
		if getErr != nil {
			is.log.Warn("Error fetching existing invitations by email", "error", getErr)
			return nil, fmt.Errorf("failed checking existing invitations by email: %w", getErr)
		}
		for _, eInv := range existingInvs {
			if eInv.Status == types.InvitationStatusPending && 
			   eInv.WmsID == inv.WmsID && eInv.CompanyID == inv.CompanyID {
				return nil, fmt.Errorf("there is already a pending invitation for that email under this Wms/Company")
			}
		}

		// ensure no user with that email already exists
		emailExists, eErr := is.userRepo.EmailExists(ctx, tx, *inv.Email)
		if eErr != nil {
			is.log.Warn("Error checking email existence", "error", eErr)
			return nil, fmt.Errorf("failed checking email existence: %w", eErr)
		}
		if emailExists {
			return nil, fmt.Errorf("that email is already in use")
		}

	} else { // phone
		existingInvs, getErr := is.invitationRepo.GetByPhoneNumbers(ctx, tx, []string{*inv.PhoneNumber})
		if getErr != nil {
			is.log.Warn("Error fetching existing invitations by phone", "error", getErr)
			return nil, fmt.Errorf("failed checking existing invitations by phone: %w", getErr)
		}
		for _, pInv := range existingInvs {
			if pInv.Status == types.InvitationStatusPending && 
			   pInv.WmsID == inv.WmsID && pInv.CompanyID == inv.CompanyID {
				return nil, fmt.Errorf("there is already a pending invitation for that phone number under this Wms/Company")
			}
		}

		// ensure no user with that phone already exists
		phoneExists, pErr := is.userRepo.PhoneNumberExists(ctx, tx, *inv.PhoneNumber)
		if pErr != nil {
			is.log.Warn("Error checking phone number existence", "error", pErr)
			return nil, fmt.Errorf("failed checking phone number existence: %w", pErr)
		}
		if phoneExists {
			return nil, fmt.Errorf("that phone number is already in use")
		}
	}

	// Fill out invitation fields
	inv.InviteUserID = user.ID
	inv.Status = types.InvitationStatusPending
	if inv.Token == "" {
		inv.Token = uuid.NewString()
	}
	if inv.ExpiresAt.IsZero() {
		inv.ExpiresAt = time.Now().Add(48 * time.Hour)
	}

	// Create in DB
	_, cErr := is.invitationRepo.Create(ctx, tx, []*types.Invitation{inv})
	if cErr != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", cErr)
	}

	// 5) Generate & upload avatar for the Invitation
	invWithAvatar, avErr := is.avatarService.CreateAndUploadInvitationAvatar(ctx, tx, inv)
	if avErr != nil {
		return nil, fmt.Errorf("failed to create/upload invitation avatar: %w", avErr)
	}

	updatedInvSlice, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{invWithAvatar})
	if upErr != nil || len(updatedInvSlice) == 0 {
		return nil, fmt.Errorf("failed to update invitation with avatar: %w", upErr)
	} 
	final := updatedInvSlice[0]
	channel := is.getInvitationSSEChannel(final)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event: sse.SSEEventInvitationCreated,
			})
		}
	}
	return final, nil
}

// sendInvitationOutbound is called *after* the transaction commits
// to send the actual invitation (email or SMS).
func (is *invitationService) sendInvitationOutbound(ctx context.Context, inv *types.Invitation) error {
	// Build the link
	linkURL := fmt.Sprintf("%s/register?token=%s", is.frontEndURL, inv.Token)

	// Determine which contact method to use
	var inviteMethod string
	if inv.Email != nil && *inv.Email != "" {
		inviteMethod = "email"
	} else if inv.PhoneNumber != nil && *inv.PhoneNumber != "" {
		inviteMethod = "phone"
	} else {
		// Edge case: no contact info
		return fmt.Errorf("invitation has no email or phone set")
	}

	// If using email, build and send
	if inviteMethod == "email" {
		var finalAvatarURL string

		if inv.WmsID != nil && *inv.WmsID != uuid.Nil {
			foundWms, _ := is.wmsRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.WmsID})
			if len(foundWms) > 0 {
				finalAvatarURL = foundWms[0].AvatarURL
			}
		}
		if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
			foundCompany, _ := is.companyRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.CompanyID})
			if len(foundCompany) > 0 {
				finalAvatarURL = foundCompany[0].AvatarURL
			}
		}

		templateData := templates.InvitationEmailData{
			Logo:           is.brandLogoPath,
			InvitationLink: linkURL,
			AvatarURL:      finalAvatarURL,
			InvitationType: templates.InvitationType(string(inv.InvitationType)),
			WmsName:        "",
			CompanyName:    "",
		}
		// Optionally fill WmsName/CompanyName for better email rendering
		if inv.WmsID != nil && *inv.WmsID != uuid.Nil {
			foundWms, _ := is.wmsRepo.GetByIDs(ctx, nil, []uuid.UUID{*inv.WmsID})
			if len(foundWms) > 0 {
				templateData.WmsName = foundWms[0].Name
			}
		}
		if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
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

		if sendErr := is.emailService.SendEmail(ctx, *inv.Email, subject, plainText, htmlContent, "invitation"); sendErr != nil {
			is.log.Warn("Failed to send invitation email", "error", sendErr)
			return sendErr
		}
		return nil
	}

	// Otherwise, phone
	textBody := fmt.Sprintf("Slotter invitation! Click here: %s", linkURL)
	if err := is.textService.SendText(ctx, *inv.PhoneNumber, textBody); err != nil {
		is.log.Warn("Failed to send invitation text", "error", err)
		return err
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
		err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
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
	if newName != "" {
		inv.Name = &newName
	}
	if newMessage != "" {
		inv.Message = &newMessage
	}
	updated, err := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if err != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation: %w", err)
	}
	final := updated[0]
	channel := is.getInvitationSSEChannel(final)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event: sse.SSEEventInvitationUpdated,
			})
		}
	}
	return final, nil
}

func (is *invitationService) canUpdateInvitation(inv *types.Invitation) bool {
	switch inv.Status {
	case types.InvitationStatusPending:
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
	final := updated[0]
	channel := is.getInvitationSSEChannel(final)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event: sse.SSEEventInvitationUpdated,
			})
		}
	}
	return final, nil
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
	inv := existing[0]
	if !is.canCancelInvitation(inv) {
		return nil, fmt.Errorf("cannot cancel invitation in status: %s", inv.Status)
	}
	inv.Status = types.InvitationStatusCanceled
	inv.ExpiresAt = time.Time{}
	now := time.Now()
	inv.CanceledAt = &now
	updated, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if upErr != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation as canceled: %w", upErr)
	}
	final := updated[0]
	channel := is.getInvitationSSEChannel(final)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event: sse.SSEEventInvitationCanceled,
			})
		}
	}
	return final, nil
}

func (is *invitationService) canCancelInvitation(inv *types.Invitation) bool {
	if inv.Status == types.InvitationStatusPending {
		return true
	}
	return false
}

func (is *invitationService) ResendInvitation(ctx context.Context, tx *gorm.DB, invID uuid.UUID) (*types.Invitation, error) {
	if tx != nil {
		// Use the provided transaction
		finalInv, err := is.resendInvitationLogic(ctx, tx, invID)
		if err != nil {
			return nil, err
		}
		// After the transaction logic, send the outbound invitation
		outErr := is.sendInvitationOutbound(ctx, finalInv)
		if outErr != nil {
			is.log.Warn("Failed to resend invitation outbound", "error", outErr)
			return finalInv, outErr
		}
		return finalInv, nil
	}

	// No transaction provided, so start our own.
	var finalInv *types.Invitation
	err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
		inv, logicErr := is.resendInvitationLogic(ctx, innerTx, invID)
		if logicErr != nil {
			return logicErr
		}
		finalInv = inv
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Now that DB changes have committed, do the outbound send.
	outErr := is.sendInvitationOutbound(ctx, finalInv)
	if outErr != nil {
		is.log.Warn("Failed to resend invitation outbound", "error", outErr)
		return finalInv, outErr
	}
	return finalInv, nil
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
	if inv.Status == types.InvitationStatusAccepted {
		return nil, fmt.Errorf("cannot resend an already accepted invitation")
	}
	if inv.Status == types.InvitationStatusPending {
		return nil, fmt.Errorf("cannot resend an invitation that is still pending")
	}
	if inv.Status == types.InvitationStatusCanceled || 
	   inv.Status == types.InvitationStatusRejected || 
	   inv.Status == types.InvitationStatusExpired {
		inv.CanceledAt = nil
		inv.RejectedAt = nil
		inv.ExpiredAt = nil
	}
	inv.Status = types.InvitationStatusPending
	inv.Token = uuid.NewString()
	inv.ExpiresAt = time.Now().Add(48 * time.Hour)
	inv.CreatedAt = time.Now()

	updated, upErr := is.invitationRepo.Update(ctx, tx, []*types.Invitation{inv})
	if upErr != nil || len(updated) == 0 {
		return nil, fmt.Errorf("failed to update invitation for resend: %w", upErr)
	}
	final := updated[0]

	// SSE event inside the transaction
	channel := is.getInvitationSSEChannel(final)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event:   sse.SSEEventInvitationResent,
			})
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
	if err := is.invitationRepo.SoftDeleteByInvitations(ctx, tx, []*types.Invitation{inv}); err != nil {
		return err
	}
	channel := is.getInvitationSSEChannel(inv)
	if channel != "" {
		ssd := ssedata.GetSSEData(ctx)
		if ssd != nil {
			ssd.AppendMessage(sse.SSEMessage{
				Channel: channel,
				Event: sse.SSEEventInvitationDeleted,
			})
		}
	}
	return nil
}

func (is *invitationService) canDeleteInvitation(inv *types.Invitation) bool {
	switch inv.Status {
	case types.InvitationStatusAccepted,
			 types.InvitationStatusCanceled,
			 types.InvitationStatusRejected,
			 types.InvitationStatusCanceled,
			 types.InvitationStatusExpired:
		return true
	}
	return false
}

func (is *invitationService) ValidateInvitationToken(ctx context.Context, tx *gorm.DB, token string) (*types.Invitation, error) {
	if tx != nil {
		return is.validateInvitationTokenLogic(ctx, tx, token)
	}
	var out *types.Invitation
	err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
		inv, logicErr := is.validateInvitationTokenLogic(ctx, innerTx, token)
		if logicErr != nil {
			return logicErr
		}
		out = inv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (is *invitationService) validateInvitationTokenLogic(ctx context.Context, tx *gorm.DB, token string) (*types.Invitation, error) {
	if token == "" {
		return nil, fmt.Errorf("empty invitation token")
	}
	foundInvs, err := is.invitationRepo.GetByTokens(ctx, tx, []string{token})
	if err != nil {
		is.log.Warn("Failed fetching invitation by token", "error", err)
		return nil, fmt.Errorf("failed fetching invitation by token: %w", err)
	}
	if len(foundInvs) == 0 {
		return nil, fmt.Errorf("no invitation found for that token")
	}
	if len(foundInvs) > 1 {
		return nil, fmt.Errorf("mutliple invitations found for that token")
	}
	inv := foundInvs[0]
	if inv.Status != types.InvitationStatusPending {
		return nil, fmt.Errorf("invitation is not pending (status: %s)", inv.Status)
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("invitation token is expired")
	}
	return inv, nil
}

func (is *invitationService) ExpirePendingInvitations(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx != nil {
		return is.expirePendingInvitationsLogic(ctx, tx)
	}
	var totalExpired int64
	err := is.db.WithContext(ctx).Transaction(func(innerTx *gorm.DB) error {
		expired, innerErr := is.expirePendingInvitationsLogic(ctx, innerTx)
		if innerErr != nil {
			return innerErr
		}
		totalExpired = expired
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalExpired, nil
}

func (is *invitationService) expirePendingInvitationsLogic(ctx context.Context, tx *gorm.DB) (int64, error) {
	return is.invitationRepo.BulkExpireInvitations(ctx, tx)
}

func (is *invitationService) getInvitationSSEChannel(inv *types.Invitation) string {
	if inv.WmsID != nil && *inv.WmsID != uuid.Nil {
		return "wms:" + inv.WmsID.String()
	}
	if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
		return "company:" + inv.CompanyID.String()
	}
	return ""
}
