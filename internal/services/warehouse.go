package services

import (
  "context"
  "fmt"
  "strings"

  "gorm.io/gorm"
  "github.com/google/uuid"

  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/requestdata"
)

type WarehouseService interface {
  CreateWarehouse(ctx context.Context, newWarehouseName string, companyID uuid.UUID) (*types.Warehouse, error)
  CreateWarehouseWithTransaction(ctx context.Context, tx *gorm.DB, newWarehouseName string, companyID uuid.UUID) (*types.Warehouse, error)
  UpdateWarehouseName(ctx context.Context, warehouse *types.Warehouse, newWarehouseName string) (*types.Warehouse, error)
  UpdateWarehouseNameWithTransaction(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse, newWarehouseName string) (*types.Warehouse, error) 
  DeleteWarehouse(ctx context.Context, warehouse *types.Warehouse) error
  DeleteWarehouseWithTransaction(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse) error
}

type warehouseService struct {
  db              *gorm.DB
  log             *logger.Logger
  userRepo        repos.UserRepo
  wmsRepo         repos.WmsRepo
  companyRepo     repos.CompanyRepo
  roleRepo        repos.RoleRepo
  permissionRepo  repos.PermissionRepo
  warehouseRepo   repos.WarehouseRepo
}

func NewWarehouseService(
  db              *gorm.DB,
  log             *logger.Logger,
  userRepo        repos.UserRepo,
  wmsRepo         repos.WmsRepo,
  companyRepo     repos.CompanyRepo,
  roleRepo        repos.RoleRepo,
  permissionRepo  repos.PermissionRepo,
  warehouseRepo   repos.WarehouseRepo,
) WarehouseService {
  serviceLog := log.With("service", "WarehouseService")
  return &warehouseService{
    db:             db,
    log:            serviceLog,
    userRepo:       userRepo,
    wmsRepo:        wmsRepo,
    companyRepo:    companyRepo,
    roleRepo:       roleRepo,
    permissionRepo: permissionRepo,
    warehouseRepo:  warehouseRepo,
  }
}

func (ws *warehouseService) CreateWarehouse(
  ctx context.Context,
  newWarehouseName string,
  companyID uuid.UUID,
) (*types.Warehouse, error) {
  var theWarehouse *types.Warehouse
  err := ws.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    w, createErr := ws.CreateWarehouseWithTransaction(ctx, tx, newWarehouseName, companyID)
    if createErr != nil {
      return createErr
    }
    theWarehouse = w
    return nil
  })
  if err != nil {
    return nil, err
  }
  return theWarehouse, nil
}

func (ws *warehouseService) CreateWarehouseWithTransaction(
  ctx context.Context,
  tx *gorm.DB,
  newWarehouseName string,
  companyID uuid.UUID,
) (*types.Warehouse, error) {
  if tx == nil {
    ws.log.Warn("CreateWarehouseWithTransaction called with nil transaction")
    return nil, fmt.Errorf("transaction cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return nil, fmt.Errorf("Request Data is not set in context.")
  }
  if rd.UserID == uuid.Nil {
    ws.log.Warn("User ID not set in RequestData.")
    return nil, fmt.Errorf("User ID not set in Request Data.")
  }
  if rd.UserType == "" {
    ws.log.Warn("UserType not set in RequestData.")
    return nil, fmt.Errorf("UserType not set in Request Data.")
  }

  var theWarehouse types.Warehouse
  switch rd.UserType {
  case "wms":
    if companyID == uuid.Nil {
      ws.log.Warn("Wms user is trying to create a Warehouse but companyID is nil || uuid.Nil. Cannot Proceed.")
      return nil, fmt.Errorf("Wms user is trying to create a Warehouse with no attached companyID")
    }
    companies, cgErr := ws.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{companyID})
    if cgErr != nil {
      ws.log.Warn("Error fetching company by company ID", "error", cgErr)
      return nil, cgErr
    }
    if len(companies) == 0 {
      ws.log.Warn("No companies found with the passed in company ID")
      return nil, fmt.Errorf("No found companies with that company ID")
    }
    theCompany := *companies[0]
    if theCompany.WmsID == nil {
      ws.log.Warn("The company given is not associated with any wms")
      return nil, fmt.Errorf("The company given is not associated with any wms")
    }
    if *(theCompany.WmsID) != rd.WmsID {
      ws.log.Warn("The company given is not associated with the same wms as the user making the request")
      return nil, fmt.Errorf("The company given is not associated with the same wms as the user making the request")
    }
    if newWarehouseName == "" {
      ws.log.Warn("Warehouse cannot be created because no new name was given")
      return nil, fmt.Errorf("Warehouse cannot be created because no new name was given")
    }
    exists, weErr := ws.warehouseRepo.NameExistsForCompany(ctx, tx, companyID, newWarehouseName)
    if weErr != nil {
      ws.log.Warn("Failed to check whether warehouse name already exists under company in question", "error", weErr)
      return nil, fmt.Errorf("Failed to check whether warehouse name exists for company: %w", weErr)
    }
    if exists {
      ws.log.Warn("Warehouse name already in use within given company")
      return nil, fmt.Errorf("Warehouse name already in use within given company")
    }
    theWarehouse.Name = strings.ToLower(strings.TrimSpace(newWarehouseName))
    theWarehouse.CompanyID = companyID

  case "company":
    if rd.CompanyID == uuid.Nil {
      ws.log.Warn("User is of type 'company' but no company ID exists in Request Data.")
      return nil, fmt.Errorf("User of type 'company' has no CompanyID in Request Data.")
    }
    if newWarehouseName == "" {
      ws.log.Warn("Warehouse cannot be created because no new name was given.")
      return nil, fmt.Errorf("Warehouse cannot be created because no new name was given.")
    }
    exists, weErr := ws.warehouseRepo.NameExistsForCompany(ctx, tx, rd.CompanyID, newWarehouseName)
    if weErr != nil {
      ws.log.Warn("Failed to check whether warehouse name already exists under company in question", "error", weErr)
      return nil, fmt.Errorf("Failed to check whether warehouse name exists for company: %w", weErr)
    }
    if exists {
      ws.log.Warn("Warehouse name already in use within given company")
      return nil, fmt.Errorf("Warehouse name already in use within given company")
    }
    theWarehouse.Name = strings.ToLower(strings.TrimSpace(newWarehouseName))
    theWarehouse.CompanyID = rd.CompanyID

  default:
    ws.log.Warn("Invalid user type for creating a warehouse", "userType", rd.UserType)
    return nil, fmt.Errorf("Invalid userType '%s' for creating a warehouse", rd.UserType)
  }
  created, cErr := ws.warehouseRepo.Create(ctx, tx, []*types.Warehouse{&theWarehouse})
  if cErr != nil {
    ws.log.Warn("Failed to create new warehouse", "error", cErr)
    return nil, fmt.Errorf("Failed to create new warehouse: %w", cErr)
  }
  if len(created) == 0 {
    ws.log.Warn("No warehouse was actually created, unexpected empty result")
    return nil, fmt.Errorf("warehouse creation returned empty result")
  }
  theWarehouse = *created[0]

  return &theWarehouse, nil
}

func (ws *warehouseService) UpdateWarehouseName(ctx context.Context, warehouse *types.Warehouse, newWarehouseName string) (*types.Warehouse, error) {
  var updated *types.Warehouse
  err := ws.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    w, upErr := ws.UpdateWarehouseNameWithTransaction(ctx, tx, warehouse, newWarehouseName)
    if upErr != nil {
      return upErr
    }
    updated = w
    return nil
  })
  if err != nil {
    return nil, err
  }
  return updated, nil
}

func (ws *warehouseService) UpdateWarehouseNameWithTransaction(
  ctx context.Context,
  tx *gorm.DB,
  warehouse *types.Warehouse,
  newWarehouseName string,
) (*types.Warehouse, error) {
  if tx == nil {
    ws.log.Warn("UpdateWarehouseNameWithTransaction called with nil transaction")
    return nil, fmt.Errorf("transaction cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return nil, fmt.Errorf("Request Data is not set in context.")
  }
  if rd.UserID == uuid.Nil {
    ws.log.Warn("User ID not set in RequestData.")
    return nil, fmt.Errorf("User ID not set in Request Data.")
  }
  if rd.UserType == "" {
    ws.log.Warn("UserType not set in RequestData.")
    return nil, fmt.Errorf("UserType not set in Request Data.")
  }
  if warehouse == nil || warehouse.ID == uuid.Nil {
    ws.log.Warn("No valid warehouse object provided")
    return nil, fmt.Errorf("Warehouse object is nil or has no valid UUID")
  }
  if warehouse.CompanyID == uuid.Nil {
    ws.log.Warn("The warehouse object has no associated company, cannot update name.")
    return nil, fmt.Errorf("Cannot update warehouse name with no associated company")
  }
  if strings.TrimSpace(newWarehouseName) == "" {
    ws.log.Warn("New Warehouse name is empty or whitespace")
    return nil, fmt.Errorf("New warehouse name cannot be empty.")
  }
  switch rd.UserType {
  case "wms":
    companies, cErr := ws.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{warehouse.CompanyID})
    if cErr != nil {
      ws.log.Warn("Failed to fetch warehouse's company for update name", "error", cErr)
      return nil, cErr
    }
    if len(companies) == 0 {
      ws.log.Warn("No company found matching warehouse's companyID during update.")
      return nil, fmt.Errorf("no matching company found for warehouse's companyID")
    }
    theCompany := companies[0]
    if theCompany.WmsID == nil || *theCompany.WmsID != rd.WmsID {
      ws.log.Warn("Warehouse's company does not belong to the same WMS as the user")
      return nil, fmt.Errorf("The warehouse's company does not match the user's wms")
    }
  
  case "company":
    if rd.CompanyID != warehouse.CompanyID {
      ws.log.Warn("Company user tried to update a warehouse from another company")
      return nil, fmt.Errorf("Cannot update warehouse belonging to another company")
    }
  
  default:
    ws.log.Warn("Invalid userType for updating a warehouse name", "userType", rd.UserType)
    return nil, fmt.Errorf("invalid userType '%s' for updating a warehouse name", rd.UserType)
  }
  exists, weErr := ws.warehouseRepo.NameExistsForCompany(ctx, tx, warehouse.CompanyID, newWarehouseName)
  if weErr != nil {
    ws.log.Warn("Failed to check if new warehouse name already exists within given company", "error", weErr)
    return nil, fmt.Errorf("Failed checking new warehouse name: %w", weErr)
  }
  if exists {
    ws.log.Warn("Attempting to rename warehouse but that name is already in use for the given company.")
    return nil, fmt.Errorf("Warehouse name is already in use under the given company")
  }
  oldName := warehouse.Name
  warehouse.Name = strings.ToLower(strings.TrimSpace(newWarehouseName))
  updated, uErr := ws.warehouseRepo.Update(ctx, tx, []*types.Warehouse{warehouse})
  if uErr != nil {
    ws.log.Warn("Failed to update warehouse name in DB", "error", uErr)
    return nil, fmt.Errorf("Failed to update warehouse name: %w", uErr)
  }
  if len(updated) == 0 {
    ws.log.Warn("No Warehouses updated, unexpected empty slice.")
    return nil, fmt.Errorf("No warehouse was updated - unexpected empty result")
  }
  ws.log.Info("Warehouse name updated successfully", "oldName", oldName, "newName", updated[0].Name)
  return updated[0], nil
}

func (ws *warehouseService) DeleteWarehouse(ctx context.Context, warehouse *types.Warehouse) error {
  return ws.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    return ws.DeleteWarehouseWithTransaction(ctx, tx, warehouse)
  })
}

func (ws *warehouseService) DeleteWarehouseWithTransaction(
  ctx context.Context,
  tx *gorm.DB,
  warehouse *types.Warehouse,
) error {
  if tx == nil {
    ws.log.Warn("DeleteWarehouseWithTransaction called with nil transaction.")
    return fmt.Errorf("Transaction cannot be nil")
  }
  rd := requestdata.GetRequestData(ctx)
  if rd == nil {
    ws.log.Warn("Request Data is not set in context.")
    return fmt.Errorf("Request Data is not set in context")
  }
  if rd.UserID == uuid.Nil {
    ws.log.Warn("User ID not set in RequestData.")
    return fmt.Errorf("User ID not set in Request Data")
  }
  if rd.UserType == "" {
    ws.log.Warn("UserType not set in RequestData.")
    return fmt.Errorf("UserType not set in Request Data")
  }
  if warehouse == nil || warehouse.ID == uuid.Nil {
    ws.log.Warn("DeleteWarehouse called with nil or invalid warehouse.")
    return fmt.Errorf("Invalid warehouse (nil or missing ID)")
  }
  if warehouse.CompanyID == uuid.Nil {
    ws.log.Warn("Warehouse has no associated company, cannot delete.")
    return fmt.Errorf("Warehouse has no associated company - cannot delete")
  }
  switch rd.UserType {
  case "wms":
    companies, cErr := ws.companyRepo.GetByIDs(ctx, tx, []uuid.UUID{warehouse.CompanyID})
    if cErr != nil {
      ws.log.Warn("Error loading warehouse's company for deletion", "error", cErr)
      return cErr
    }
    if len(companies) == 0 {
      ws.log.Warn("No matching company for warehouse.CompanyID; cannot delete warehouse.")
      return fmt.Errorf("No matching company for warehouse; cannot delete")
    }
    theCompany := companies[0]
    if theCompany.WmsID == nil || *theCompany.WmsID != rd.WmsID {
      ws.log.Warn("User's wms does not match the warehouse's company's WMS.")
      return fmt.Errorf("Cannot delete warehouse that belongs doesnt belong to a company under given wms")
    }
  
  case "company":
    if rd.CompanyID != warehouse.CompanyID {
      ws.log.Warn("Company user tried to delete a warehouse from another company.")
      return fmt.Errorf("Cannot delete a warehouse that belongs to another company")
    }

  default:
    ws.log.Warn("Invalid userType for deleting a warehouse", "userType", rd.UserType)
    return fmt.Errorf("Invalid userType '%s' for deleting a warehouse", rd.UserType)
  }
  delErr := ws.warehouseRepo.FullDeleteByWarehouses(ctx, tx, []*types.Warehouse{warehouse})
  if delErr != nil {
    ws.log.Warn("Failed to delete warehouse from DB", "error", delErr)
    return fmt.Errorf("Failed to delete warehouse: %w", delErr)
  }
  ws.log.Info("Warehouse successfully deleted", "warehouseID", warehouse.ID)
  return nil
}
