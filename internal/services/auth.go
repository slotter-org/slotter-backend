package services

import (
  "context"
  "fmt"
  "time"

  "gorm.io/gorm"
  "golang.org/x/crypto/bcrypt"

  "github.com/golang-jwt/jwt/v5"
  "github.com/google/uuid"

  "github.com/yungbote/slotter/backend/internal/normalization"
  "github.com/yungbote/slotter/backend/internal/logger"
  "github.com/yungbote/slotter/backend/internal/types"
  "github.com/yungbote/slotter/backend/internal/repos"
  "github.com/yungbote/slotter/backend/internal/requestdata"
  "github.com/yungbote/slotter/backend/internal/utils"
)

type JWTClaims struct {
  jwt.RegisteredClaims
  UserType    string      `json:"user_type,omitempty"`
  WmsID       string      `json:"wms_id,omitempty"`
  CompanyID   string      `json:"company_id,omitempty"`
  RoleID      string      `json:"role_id,omitempty"`
}

type AuthService interface {
  RegisterUser(ctx context.Context, user *types.User, newCompanyName, newWmsName string) error
  Login(ctx context.Context, email, password string) (string, string, error)
  Refresh(ctx context.Context) (string, string, error)
  Logout(ctx context.Context) error

  handleWmsRegistration(ctx context.Context, tx *gorm.DB, user *types.User, newWmsName string) error
  handleCompanyRegistration(ctx context.Context, tx *gorm.DB, user *types.User, newCompanyName string) error
  createFinalUser(ctx context.Context, tx *gorm.DB, user *types.User) error

  generateAccessToken(ctx context.Context, tx *gorm.DB, user *types.User) (string, error)

  SetContextFromToken(ctx context.Context, tokenString string) (context.Context, error)

  GetAccessTTL() time.Duration
}

type authService struct {
  db                *gorm.DB
  log               *logger.Logger
  userRepo          repos.UserRepo
  wmsRepo           repos.WmsRepo
  companyRepo       repos.CompanyRepo
  roleRepo          repos.RoleRepo
  permissionRepo    repos.PermissionRepo
  avatarService     AvatarService
  userTokenRepo     repos.UserTokenRepo
  jwtSecretKey      string
  accessTTL         time.Duration
  refreshTTL        time.Duration
}

func NewAuthService(
  db                *gorm.DB,
  log               *logger.Logger,
  userRepo          repos.UserRepo,
  wmsRepo           repos.WmsRepo,
  companyRepo       repos.CompanyRepo,
  roleRepo          repos.RoleRepo,
  permissionRepo    repos.PermissionRepo,
  avatarService     AvatarService,
  userTokenRepo     repos.UserTokenRepo,
  jwtSecretKey      string,
  accessTTL         time.Duration,
  refreshTTL        time.Duration,
) AuthService {
  serviceLog := log.With("service", "AuthService")
  return &authService{
    db:             db,
    log:            serviceLog,
    userRepo:       userRepo,
    wmsRepo:        wmsRepo,
    companyRepo:    companyRepo,
    roleRepo:       roleRepo,
    permissionRepo: permissionRepo,
    avatarService:  avatarService,
    userTokenRepo:  userTokenRepo,
    jwtSecretKey:   jwtSecretKey,
    accessTTL:      accessTTL,
    refreshTTL:     refreshTTL,
  }
}

//----------------------------------------------------------------------------------------------------------------------
// RegisterUser, handleWmsRegistration, handleCompanyRegistration, createFinalUser
//----------------------------------------------------------------------------------------------------------------------

func (as *authService) RegisterUser(ctx context.Context, user *types.User, newCompanyName, newWmsName string) error {
  as.log.Info("Starting Register User now...")
  as.log.Debug("User:", "user", *user)
  //1) Normalize User Fields
  utils.NormalizeUserFields(ctx, user)

  //2) Checks on user fields
  if vErr := utils.InputValidation(ctx, "registration", as.userRepo, as.log, user, "", ""); vErr != nil {
    return vErr
  }

  //3) Hash Password
  if hErr := utils.HashPassword(ctx, as.log, user); hErr != nil {
    return hErr
  }

  //4) Transaction Body
  return as.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error{
    switch user.UserType {
    case "wms":
      if err := as.handleWmsRegistration(ctx, tx, user, newWmsName); err != nil {
        return err
      }
    case "company":
      if err := as.handleCompanyRegistration(ctx, tx, user, newCompanyName); err != nil {
        return err
      }
    default:
      as.log.Warn("Invalid user type. Should be either 'wms' or 'company', Cannot proceed further. Returning error.")
      return fmt.Errorf("Invalid user type. Should be either 'wms' or 'company': '%s'", user.UserType)
    }

    //5) Create Final User
    if fuErr := as.createFinalUser(ctx, tx, user); fuErr != nil {
      return fuErr
    }
    return nil
  })
}

func (as *authService) handleWmsRegistration(ctx context.Context, tx *gorm.DB, user *types.User, newWmsName string) error {
  var theWms *types.Wms
  if user.WmsID == nil || *user.WmsID == uuid.Nil {
    if newWmsName == "" {
      as.log.Warn("User type 'wms' must have a wms id or a new wms name for registration, cannot proceed further. Returning error.")
      return fmt.Errorf("User type 'wms' must have a wms id or a new wms name for successful wms registration.")
    }
    theWms = &types.Wms{
      ID:         uuid.New(),
      Name:       normalization.ParseInputString(newWmsName),
    }
    avatarErr := as.avatarService.CreateAndUploadWmsAvatar(ctx, tx, theWms)
    if avatarErr != nil {
      as.log.Warn("Failure to create and upload avatar for new wms, cannot proceed further. Returning error", "error", avatarErr)
      return fmt.Errorf("Failure to create and upload avatar for new wms: %w", avatarErr)
    }
    createdWs, cWErr := as.wmsRepo.Create(ctx, tx, []*types.Wms{theWms})
    if cWErr != nil {
      as.log.Warn("Failed to create new wms, Cannot proceed further. Returning error.", "error", cWErr)
      return fmt.Errorf("Failed to create new Wms: %w", cWErr)
    }
    theWms = createdWs[0]
    user.WmsID = &theWms.ID
  } else {
    foundWs, fWErr := as.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{*user.WmsID})
    if fWErr != nil {
      as.log.Warn("Failed to fetch Wms from id, Cannot proceed further. Returning error.", "error", fWErr)
      return fmt.Errorf("Failure to fetch Wms from id: %w", fWErr)
    }
    theWms = foundWs[0]
  }
  foundUsers, fUErr := as.userRepo.GetByWmsIDs(ctx, tx, []uuid.UUID{theWms.ID})
  if fUErr != nil {
    as.log.Warn("Failed to fetch users by Wms id, Cannot proceed further. Returning error.", "error", fUErr)
    return fmt.Errorf("Failure to fetch users from Wms id: %w", fUErr)
  }
  if len(foundUsers) == 0 {
    adminRole := &types.Role{WmsID: &theWms.ID, Name: "admin"}
    defaultRole := &types.Role{WmsID: &theWms.ID, Name: "default"}
    newRoles, nRErr := as.roleRepo.Create(ctx, tx, []*types.Role{adminRole, defaultRole})
    if nRErr != nil {
      as.log.Warn("Failed to create admin and default roles for new wms, Cannot proceed further. Returning error.", "error", nRErr)
      return fmt.Errorf("Failure to create admin and default roles for new wms: %w", nRErr)
    }
    allPerms, aPErr := as.permissionRepo.GetAll(ctx, tx)
    if aPErr != nil {
      as.log.Warn("Failed to fetch all permissions to associate with new admin and default roles for new wms, Cannot proceed further. Returning error.", "error", aPErr)
      return fmt.Errorf("Failure to fetch all permissions to associate with new admin and default roles for new wms: %w", aPErr)
    }
    if rPAErr := as.roleRepo.AssociatePermissions(ctx, tx, []*types.Role{newRoles[0]}, allPerms); rPAErr != nil {
      as.log.Warn("Failed to associate all permissions with new admin role for new wms, Cannot proceed further. Returning error.", "error", rPAErr)
      return fmt.Errorf("Failure to associate all permissions with new admin role for new wms: %w", rPAErr)
    }
    user.RoleID = &newRoles[0].ID
    theWms.DefaultRoleID = &newRoles[1].ID
  } else {
    user.RoleID = theWms.DefaultRoleID
  }
  _, sWErr := as.wmsRepo.Update(ctx, tx, []*types.Wms{theWms})
  if sWErr != nil {
    as.log.Warn("Failed to update final wms, Cannot proceed further. Returning error.", "error", sWErr)
    return fmt.Errorf("Failure to update final wms: %w", sWErr)
  }
  return nil
}

func (as *authService) handleCompanyRegistration(ctx context.Context, tx *gorm.DB, user *types.User, newCompanyName string) error {
  var theCompany *types.Company
  if user.CompanyID == nil || *user.CompanyID == uuid.Nil {
    if newCompanyName == "" {
      as.log.Warn("user of type 'company' must have either a company id or a new company name to register.")
      return fmt.Errorf("user of type 'company' must have either a company id or a new company name to register.")
    }
    theCompany = &types.Company{
      ID:           uuid.New(),
      Name:         normalization.ParseInputString(newCompanyName),
    }
    if user.WmsID != nil  && *user.WmsID != uuid.Nil {
      foundWmss, fWErr := as.wmsRepo.GetByIDs(ctx, tx,  []uuid.UUID{*(user.WmsID)})
      if len(foundWmss) > 0 && fWErr != nil {
        as.log.Warn("Failed to fetch wms from user.WmsID")
        return fmt.Errorf("Failed to fetch wms from user.WmsID")
      }
      if len(foundWmss) == 0 {
        as.log.Warn("No Wms with that id exist.")
        return fmt.Errorf("No Wms with that id exist.")
      }
      foundWms := foundWmss[0]
      theCompany.WmsID = &(foundWms.ID)
      user.WmsID = nil
      _, wErr := as.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{*theCompany.WmsID})
      if wErr != nil {
        as.log.Warn("Failed to find the wms we are assigning to new company, Cannot proceed further. Returning error", "error", wErr)
        return fmt.Errorf("Failure to find wms we are assigning to new company: %w", wErr)
      }
    }
    avatarErr := as.avatarService.CreateAndUploadCompanyAvatar(ctx, tx, theCompany)
    if avatarErr != nil {
      as.log.Warn("Failed to create and upload new company avatar, Cannot proceed further. Returning error", "error", avatarErr)
      return fmt.Errorf("Failure to create and upload new company avatar: %w", avatarErr)
    }
    createdCompanies, cCErr := as.companyRepo.Create(ctx, tx, []*types.Company{theCompany})
    if cCErr != nil {
      as.log.Warn("Failed to create new company, Cannot proceed further. Returning error", "error", cCErr)
      return fmt.Errorf("Failure to create new company: %w", cCErr)
    }
    theCompany = createdCompanies[0]
    user.CompanyID = &theCompany.ID
  } else {
    foundCompanies, fCErr := as.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{*user.CompanyID})
    if fCErr != nil {
      as.log.Warn("Failed to find company by user.CompanyID, Cannot proceed further. Returning error", "error", fCErr)
      return fmt.Errorf("Failure to find company by user.CompanyID: %w", fCErr)
    }
    theCompany = foundCompanies[0]
  }
  foundUsers, fUErr := as.userRepo.GetByCompanyIDs(ctx, tx, []uuid.UUID{theCompany.ID})
  if fUErr != nil {
    as.log.Warn("Failed to get users by company id, Cannot proceed further. Returning error", "error", fUErr)
    return fmt.Errorf("Failure to find users by company id: %w", fUErr)
  }
  if len(foundUsers) == 0 {
    adminRole := &types.Role{CompanyID: &theCompany.ID, Name: "admin"}
    defaultRole := &types.Role{CompanyID: &theCompany.ID, Name: "default"}
    newRoles, nRErr := as.roleRepo.Create(ctx, tx, []*types.Role{adminRole, defaultRole})
    if nRErr != nil {
      as.log.Warn("Failed to create new admin and default roles for new company, Cannot proceed further. Returning error", "error", nRErr)
      return fmt.Errorf("Failure to create new admin and default roles for new company: %w", nRErr)
    }
    allPerms, aPErr := as.permissionRepo.GetAll(ctx, tx)
    if aPErr != nil {
      as.log.Warn("Failed to get all permissions to associate with new admin role for new company, Cannot proceed further. Returning error", "error", aPErr)
      return fmt.Errorf("Failure to get all permissions to associate with new admin role for new company: %w", aPErr)
    }
    if associationErr := as.roleRepo.AssociatePermissions(ctx, tx, []*types.Role{newRoles[0]}, allPerms); associationErr != nil {
      as.log.Warn("Failed to associate all permissions with new admin role for new company, Cannot proceed further. Returning error", "error", associationErr)
      return fmt.Errorf("Failure to associate all permissions with new admin role for new company: %w", associationErr)
    }
    user.RoleID = &newRoles[0].ID
    theCompany.DefaultRoleID = &newRoles[1].ID
  } else {
    user.RoleID = theCompany.DefaultRoleID
  }
  _, uCErr := as.companyRepo.Update(ctx, tx, []*types.Company{theCompany})
  if uCErr != nil {
    as.log.Warn("Failed to save final company, Cannot proceed further. Returning error", "error", uCErr)
    return fmt.Errorf("Failure to save final company: %w", uCErr)
  }
  return nil
}

func (as *authService) createFinalUser(ctx context.Context, tx *gorm.DB, user *types.User) error {
  user.ID = uuid.New()
  ucaErr := as.avatarService.CreateAndUploadUserAvatar(ctx, tx, user)
  if ucaErr != nil {
    as.log.Warn("Failure from AuthService -> AvatarManager to create and upload user avatar", "error", ucaErr)
    return fmt.Errorf("Failure to create and upload user avatar: %w", ucaErr)
  }
  createdUsers, ucErr := as.userRepo.Create(ctx, tx, []*types.User{user})
  if ucErr != nil {
    as.log.Warn("Failure from AuthService -> UserRepo to create final user", "error", ucErr)
    return fmt.Errorf("Failure to create user: %w", ucErr)
  }
  if len(createdUsers) == 0 {
    as.log.Warn("Failure to actually create user from AuthService")
    return fmt.Errorf("Failure to create user in DB")
  }
  return nil
}

func (as *authService) Login(ctx context.Context, userEmail, userPassword string) (string, string, error) {
  //1) Normalize Input
  email := normalization.ParseInputString(userEmail)
  password := normalization.ParseInputString(userPassword)

  //2) Input Validations
  if vErr := utils.InputValidation(ctx, "login", as.userRepo, as.log, &types.User{}, email, password); vErr != nil {
    return "", "", vErr
  }

  //3) Find User By Email
  users, uSErr := as.userRepo.GetByEmails(ctx, nil, []string{email})
  if uSErr != nil {
    as.log.Warn("Failure to retrieve user by email, Cannot proceed. Returning error.", "error", uSErr)
    return "", "", fmt.Errorf("error retrieving user by email: %w", uSErr)
  }
  if len(users) == 0 {
    as.log.Warn("Invalid email, no users returned", "len(users)", len(users))
    return "", "", fmt.Errorf("invalid email, no users returned")
  }
  user := users[0]
  if hErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); hErr != nil {
    as.log.Warn("Invalid password, user password and hash dont match, Cannot proceed. Returning error.", "error", hErr)
    return "", "", fmt.Errorf("invalid password, user password and hash dont match: %w", hErr)
  }

  //4) Refresh
  var accessToken string
  var refreshToken string
  if err := as.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    foundTokens, fTErr := as.userTokenRepo.GetByUserIDs(ctx, tx, []uuid.UUID{user.ID})
    if fTErr != nil && len(foundTokens) != 0 {
      as.log.Warn("Failed to check whether user already has user tokens, Cannot proceed. Returning error.", "error", fTErr)
      return fmt.Errorf("Failed to check whether user already has user tokens: %w", fTErr)
    }
    if len(foundTokens) > 0 {
      if foundTokens[0] != nil && foundTokens[0].ExpiresAt.After(time.Now()){
        as.log.Warn("User is already logged in, Cannot proceed.")
        return fmt.Errorf("User is already logged in.")
      }
      if foundTokens[0] != nil && foundTokens[0].ExpiresAt.Before(time.Now()) {
        if dTErr := as.userTokenRepo.FullDeleteByTokens(ctx, tx, []*types.UserToken{foundTokens[0]}); dTErr != nil {
          as.log.Warn("Failed to delete expired user token, Cannot proceed. Returning error.", "error", dTErr)
          return fmt.Errorf("Failed to delete expired user token: %w", dTErr)
        }
      }
    }
    tok, genErr := as.generateAccessToken(ctx, tx, user)
    if genErr != nil {
      as.log.Warn("Generate Access Token Error, Cannot proceed. Returning error.", "error", genErr)
      return fmt.Errorf("Generate Access Token Error: %w", genErr)
    }
    accessToken = tok
    refreshToken = uuid.New().String()
    expiresAt := time.Now().Add(as.refreshTTL)
    userToken := types.UserToken{
      ID:               uuid.New(),
      UserID:           user.ID,
      AccessToken:      accessToken,
      RefreshToken:     refreshToken,
      ExpiresAt:        expiresAt,
    }
    _, cTErr := as.userTokenRepo.Create(ctx, tx, []*types.UserToken{&userToken})
    if cTErr != nil {
      as.log.Warn("Create User Token Error, Cannot proceed. Returning error.", "error", cTErr)
      return fmt.Errorf("Create User Token Error: %w", cTErr)
    }
    return nil
  }); err != nil {
    return "", "", err
  }
  return accessToken, refreshToken, nil
}

func (as *authService) Refresh(ctx context.Context) (string, string, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    as.log.Warn("No Request Data found in context, Cannot proceed", "requestdata", rd)
    return "", "", fmt.Errorf("No Request Data found in context.")
  }
  if rd.TokenString == "" {
    as.log.Warn("TokenString in Request Data in context is an empty string, Cannot proceed", "tokenstring", rd.TokenString)
    return "", "", fmt.Errorf("TokenString in Request Data in context is an empty string.")
  }
  if rd.RefreshToken == "" {
    as.log.Warn("RefreshTokenString in Request Data in context is an empty string, Cannot proceed", "refreshtokenstring", rd.RefreshToken)
    return "", "", fmt.Errorf("RefreshTokenString in Request Data in context is an empty string.")
  }

  var accessToken string
  var newRefreshTokenStr string
  err := as.db.WithContext(ctx).Transaction(func (tx *gorm.DB) error {
    var existingToken *types.UserToken
    foundTokens, fTErr := as.userTokenRepo.GetByRefreshTokens(ctx, tx, []string{rd.RefreshToken})
    if foundTokens[0] != nil && fTErr != nil {
      as.log.Warn("Error fetching refresh token, Cannot proceed. Returning error.", "error", fTErr)
      return fmt.Errorf("Error fetching refresh token: %w", fTErr)
    }
    existingToken = foundTokens[0]

    if existingToken.ExpiresAt.Before(time.Now()) {
      if dTErr := as.userTokenRepo.FullDeleteByTokens(ctx, tx, []*types.UserToken{existingToken}); dTErr != nil {
        as.log.Warn("Refresh token expired, error deleting expired refresh token, Cannot proceed. Returning error.", "error", dTErr)
        return fmt.Errorf("Refresh token expired, error deleting: %w", dTErr)
      }
      as.log.Warn("Refresh Token Expired, Cannot proceed.")
      return fmt.Errorf("Refresh Token Expired.")
    }
    users, uErr := as.userRepo.GetByIDs(ctx, tx, []uuid.UUID{existingToken.UserID})
    if uErr != nil {
      as.log.Warn("Failed to load user for refresh, Cannot proceed. Returning error.", "error", uErr)
      return fmt.Errorf("Failed to load user for refresh: %w", uErr)
    }
    if len(users) == 0 {
      as.log.Warn("No user found for the given refresh token, Cannot proceed.", "len(users)", len(users))
      return fmt.Errorf("No user found for the given refresh token.")
    }
    user := users[0]
    tok, genErr := as.generateAccessToken(ctx, tx, user)
    if genErr != nil {
      as.log.Warn("Failed to generate new access token, Cannot proceed. Returning error.", "error", genErr)
      return fmt.Errorf("Failed to generate new access token: %w", genErr)
    }
    accessToken = tok
    newRefreshTokenStr = uuid.New().String()
    newExpiresAt := time.Now().Add(as.refreshTTL)
    newUserToken := types.UserToken{
      ID:               uuid.New(),
      UserID:           user.ID,
      AccessToken:      tok,
      RefreshToken:     newRefreshTokenStr,
      ExpiresAt:        newExpiresAt,
    }
    _, cErr := as.userTokenRepo.Create(ctx, tx, []*types.UserToken{&newUserToken})
    if cErr != nil {
      as.log.Warn("Failed to create new user token, Cannot proceed. Returning error.", "error", cErr)
      return fmt.Errorf("Failed to create new user token: %w", cErr)
    }
    if dErr := as.userTokenRepo.FullDeleteByTokens(ctx, tx, []*types.UserToken{existingToken}); dErr != nil {
      as.log.Warn("Failed to remove old refresh token, Cannot proceed. Returning error.", "error", dErr)
      return fmt.Errorf("Failed to remove old refresh toke: %w", dErr)
    }
    return nil
  })
  if err != nil {
    as.log.Warn("Failed transaction, Cannot proceed. Returning error.", "error", err)
    return "", "", err
  }
  return accessToken, newRefreshTokenStr, nil
}

func (as *authService) Logout(ctx context.Context) error {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    as.log.Warn("No Request Data found in context, Cannot proceed.", "requestdata", rd)
    return fmt.Errorf("No Request Data found in context.")
  }
  if rd.TokenString == "" {
    as.log.Warn("TokenString in Request Data is an empty string, Cannot proceed.", "tokenstring", rd.TokenString)
    return fmt.Errorf("TokenString in RequestData is an empty string.")
  }
  return as.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    foundTokens, fTErr := as.userTokenRepo.GetByAccessTokens(ctx, tx, []string{rd.TokenString})
    if len(foundTokens) != 0 && fTErr != nil {
      as.log.Warn("Error finding user token from token string, Cannot proceed. Returning error.", "error", fTErr)
      return fmt.Errorf("Error finding user token from token string: %w", fTErr)
    }
    if tDErr := as.userTokenRepo.FullDeleteByTokens(ctx, tx, []*types.UserToken{foundTokens[0]}); tDErr != nil {
      as.log.Warn("Error deleting user token, Cannot proceed. Returning error.", "error", tDErr)
      return fmt.Errorf("Error deleting user token: %w", tDErr)
    }
    return nil
  })
}

func (as *authService) generateAccessToken(ctx context.Context, tx *gorm.DB, user *types.User) (string, error) {
  var wmsID string
  var companyID string
  var roleID string
  if user.UserType == "wms" && user.WmsID != nil && *user.WmsID != uuid.Nil {
    wmsID = (*user.WmsID).String()
  }
  if user.UserType == "company" && user.CompanyID != nil && *user.CompanyID != uuid.Nil {
    companyID = (*user.CompanyID).String()
  }
  if user.RoleID != nil  && *user.RoleID != uuid.Nil {
    roleID = (*user.RoleID).String()
  }
  claims := JWTClaims{
    RegisteredClaims: jwt.RegisteredClaims{
      Subject: user.ID.String(),
      ExpiresAt: jwt.NewNumericDate(time.Now().Add(as.accessTTL)),
      IssuedAt: jwt.NewNumericDate(time.Now()),
    },
    UserType: user.UserType,
    WmsID: wmsID,
    CompanyID: companyID,
    RoleID: roleID,
  }
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  return token.SignedString([]byte(as.jwtSecretKey))
}


func (as *authService) SetContextFromToken(ctx context.Context, tokenString string) (context.Context, error) {
  if tokenString == "" {
    return ctx, nil
  }
  parsedToken, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
    return []byte(as.jwtSecretKey), nil
  })
  if err != nil {
    return ctx, fmt.Errorf("failed to parse token: %w", err)
  }
  claims, ok := parsedToken.Claims.(*JWTClaims)
  if !ok || !parsedToken.Valid {
    return ctx, fmt.Errorf("invalid or expired JWT token")
  }
  userID, err := uuid.Parse(claims.Subject)
  if err != nil {
    return ctx, fmt.Errorf("invalid user ID in token: %w", err)
  }
  var wmsID uuid.UUID
  if claims.WmsID != "" {
    wmsID, err = uuid.Parse(claims.WmsID)
    if err != nil {
      return ctx, fmt.Errorf("invalid Wms ID in token: %w", err)
    }
  }
  var companyID uuid.UUID
  if claims.CompanyID != "" {
    companyID, err = uuid.Parse(claims.CompanyID)
    if err != nil {
      return ctx, fmt.Errorf("invalid Company ID in token: %w", err)
    }
  }
  var roleID uuid.UUID
  if claims.RoleID != "" {
    roleID, err = uuid.Parse(claims.RoleID)
    if err != nil {
      return ctx, fmt.Errorf("invalid Role ID in token: %w", err)
    }
  }
  var refreshTokenStr string
  foundTokens, fTErr := as.userTokenRepo.GetByAccessTokens(ctx, nil, []string{tokenString})
  if len(foundTokens) != 0 && fTErr != nil {
    as.log.Warn("Error fetching user token by access token, Cannot proceed. Returning error.", "error", fTErr)
    return ctx, fmt.Errorf("Failed to fetch user token by access token: %w", fTErr)
  }
  refreshTokenStr = foundTokens[0].RefreshToken
  rd := &requestdata.RequestData{
    TokenString: tokenString,
    RefreshToken: refreshTokenStr,
    UserType: claims.UserType,
    UserID: userID,
    WmsID: wmsID,
    CompanyID: companyID,
    RoleID: roleID,
  }
  ctx = requestdata.WithRequestData(ctx, rd)
  return ctx, nil
}

func (as *authService) GetAccessTTL() time.Duration {
  return as.accessTTL
}
