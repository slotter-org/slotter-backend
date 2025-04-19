package services

import (
  "context"
  "fmt"

  "gorm.io/gorm"
  "github.com/google/uuid"

  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/repos"
)

type MeService interface {
  GetMe(ctx context.Context, tx *gorm.DB) (types.User, error)
  GetMeWithTransaction(ctx context.Context, tx *gorm.DB) (*types.User, error)

  GetMyWms(ctx context.Context, tx *gorm.DB) (types.Wms, error)
  GetMyWmsWithTransaction(ctx context.Context, tx *gorm.DB) (*types.Wms, error)

  GetMyCompany(ctx context.Context, tx *gorm.DB) (types.Company, error)
  GetMyCompanyWithTransaction(ctx context.Context, tx *gorm.DB) (*types.Company, error)

  GetMyRole(ctx context.Context, tx *gorm.DB) (types.Role, error)
}

type meService struct {
  db          *gorm.DB
  log         *logger.Logger
  userRepo    repos.UserRepo
  wmsRepo     repos.WmsRepo
  companyRepo repos.CompanyRepo
  roleRepo    repos.RoleRepo
}

func NewMeService(
  db *gorm.DB,
  log *logger.Logger,
  userRepo repos.UserRepo,
  wmsRepo repos.WmsRepo,
  companyRepo repos.CompanyRepo,
  roleRepo repos.RoleRepo,
) MeService {
  serviceLog := log.With("service", "MeService")
  return &meService{
    db: db,
    log: serviceLog,
    userRepo: userRepo,
    wmsRepo: wmsRepo,
    companyRepo: companyRepo,
    roleRepo: roleRepo,
  }
}

func (ms *meService) GetMe(ctx context.Context, tx *gorm.DB) (types.User, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ms.log.Warn("Request Data is not set in context.")
    return types.User{}, fmt.Errorf("Request Data is not set in context.")
  }
  if rd.UserID == uuid.Nil {
    ms.log.Warn("User ID not set in Request Data.")
    return types.User{}, fmt.Errorf("User ID not set in Request Data.")
  }

  var theUser types.User
  if tx == nil {
    // Wrap in a transaction ourselves
    if err := ms.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
      foundUsers, fErr := ms.userRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.UserID})
      if fErr != nil {
        return fmt.Errorf("error fetching user: %w", fErr)
      }
      if len(foundUsers) == 0 {
        return fmt.Errorf("user does not exist")
      }
      theUser = *foundUsers[0]
      return nil
    }); err != nil {
      ms.log.Warn("GetMe transaction error:", "error", err)
      return types.User{}, err
    }
  } else {
    // If they already have a transaction, use GetMeWithTransaction
    return types.User{}, fmt.Errorf("use GetMeWithTransaction for an existing tx")
  }
  return theUser, nil
}

func (ms *meService) GetMeWithTransaction(ctx context.Context, tx *gorm.DB) (*types.User, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ms.log.Warn("Request Data is not set in context.")
    return &types.User{}, fmt.Errorf("Request Data is not set in context.")
  }
  if rd.UserID == uuid.Nil {
    ms.log.Warn("User ID not set in Request Data.")
    return &types.User{}, fmt.Errorf("User ID not set in Request Data.")
  }

  foundUsers, fErr := ms.userRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.UserID})
  if fErr != nil {
    ms.log.Warn("Error fetching user in GetMeWithTransaction:", "error", fErr)
    return &types.User{}, fErr
  }
  if len(foundUsers) == 0 {
    return &types.User{}, fmt.Errorf("user does not exist")
  }
  return foundUsers[0], nil
}

func (ms *meService) GetMyWms(ctx context.Context, tx *gorm.DB) (types.Wms, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    return types.Wms{}, fmt.Errorf("Request Data is not set in context.")
  }
  if rd.WmsID == uuid.Nil && rd.CompanyID == uuid.Nil {
    return types.Wms{}, fmt.Errorf("User does not belong to any Wms or Company, cannot fetch Wms.")
  }

  var theWms types.Wms
  if tx == nil {
    if err := ms.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
      // If the user has WmsID
      if rd.WmsID != uuid.Nil {
        foundWms, wErr := ms.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.WmsID})
        if wErr != nil {
          return wErr
        }
        if len(foundWms) == 0 {
          return fmt.Errorf("Wms with that ID not found.")
        }
        theWms = *foundWms[0]
        return nil
      }

      // Otherwise fetch from the user’s Company => Wms
      foundCompanies, cErr := ms.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
      if cErr != nil {
        return cErr
      }
      if len(foundCompanies) == 0 || foundCompanies[0].WmsID == nil {
        return fmt.Errorf("Company not found or no Wms attached.")
      }
      foundWms, wErr := ms.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{*foundCompanies[0].WmsID})
      if wErr != nil {
        return wErr
      }
      if len(foundWms) == 0 {
        return fmt.Errorf("No Wms with that ID found.")
      }
      theWms = *foundWms[0]
      return nil
    }); err != nil {
      return types.Wms{}, err
    }
  } else {
    return types.Wms{}, fmt.Errorf("use GetMyWmsWithTransaction if you already have a transaction")
  }
  return theWms, nil
}

func (ms *meService) GetMyWmsWithTransaction(ctx context.Context, tx *gorm.DB) (*types.Wms, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    return nil, fmt.Errorf("request data not set in context")
  }
  if rd.WmsID == uuid.Nil && rd.CompanyID == uuid.Nil {
    return nil, fmt.Errorf("no wms or company available for user")
  }

  // If the user has WmsID directly:
  if rd.WmsID != uuid.Nil {
    foundWms, err := ms.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.WmsID})
    if err != nil {
      return nil, err
    }
    if len(foundWms) == 0 {
      return nil, fmt.Errorf("Wms with that id not found")
    }
    return foundWms[0], nil
  }

  // Otherwise, fetch from Company => Wms
  foundCompanies, cErr := ms.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
  if cErr != nil {
    return nil, cErr
  }
  if len(foundCompanies) == 0 || foundCompanies[0].WmsID == nil {
    return nil, fmt.Errorf("company not found or no Wms attached")
  }
  foundWms, wErr := ms.wmsRepo.GetByIDs(ctx, tx, []uuid.UUID{*foundCompanies[0].WmsID})
  if wErr != nil {
    return nil, wErr
  }
  if len(foundWms) == 0 {
    return nil, fmt.Errorf("Wms not found for that Company")
  }
  return foundWms[0], nil
}

func (ms *meService) GetMyCompany(ctx context.Context, tx *gorm.DB) (types.Company, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    return types.Company{}, fmt.Errorf("Request Data not set in context")
  }
  if rd.CompanyID == uuid.Nil {
    return types.Company{}, fmt.Errorf("company id not found in request data")
  }

  var theCompany types.Company
  if tx == nil {
    if err := ms.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
      foundCompanies, cErr := ms.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
      if cErr != nil {
        return cErr
      }
      if len(foundCompanies) == 0 {
        return fmt.Errorf("no company found with that id")
      }
      theCompany = *foundCompanies[0]
      return nil
    }); err != nil {
      return types.Company{}, err
    }
  } else {
    return types.Company{}, fmt.Errorf("use GetMyCompanyWithTransaction if you have a tx")
  }
  return theCompany, nil
}

func (ms *meService) GetMyCompanyWithTransaction(ctx context.Context, tx *gorm.DB) (*types.Company, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    return &types.Company{}, fmt.Errorf("request data not set in context")
  }
  if rd.CompanyID == uuid.Nil {
    return &types.Company{}, fmt.Errorf("no company id in request data")
  }

  foundCompanies, cErr := ms.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.CompanyID})
  if cErr != nil {
    return &types.Company{}, cErr
  }
  if len(foundCompanies) == 0 {
    return &types.Company{}, fmt.Errorf("no company found")
  }
  return foundCompanies[0], nil
}

func (ms *meService) GetMyRole(ctx context.Context, tx *gorm.DB) (types.Role, error) {
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    return types.Role{}, fmt.Errorf("Request data not set in context.")
  }
  if rd.RoleID == uuid.Nil {
    return types.Role{}, fmt.Errorf("No roleID in request data.")
  }

  var theRole types.Role
  if tx == nil {
    if err := ms.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
      foundRoles, rErr := ms.roleRepo.GetByIDs(ctx, tx, []uuid.UUID{rd.RoleID})
      if rErr != nil {
        return rErr
      }
      if len(foundRoles) == 0 {
        return fmt.Errorf("role not found with that id")
      }
      theRole = *foundRoles[0]
      return nil
    }); err != nil {
      return types.Role{}, err
    }
  } else {
    return types.Role{}, fmt.Errorf("use GetMyRoleWithTransaction if you have a tx")
  }
  return theRole, nil
}

