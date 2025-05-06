package services

import (
  "context"
  "bytes"
  "fmt"
  "image"
  "image/color"
  "encoding/json"
  "io/ioutil"
  "math"
  "math/rand"
  "os"
  "path/filepath"
  "strings"
  "time"

  "github.com/disintegration/imaging"
  "github.com/fogleman/gg"
  "github.com/golang/freetype/truetype"
  "golang.org/x/image/font"
  "gorm.io/gorm"
  

  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/types"
  "github.com/slotter-org/slotter-backend/internal/repos"
)

type AvatarService interface {
  CreateAndUploadWmsAvatar(ctx context.Context, tx *gorm.DB, wms *types.Wms) error
  CreateAndUploadCompanyAvatar(ctx context.Context, tx *gorm.DB, company *types.Company) error
  CreateAndUploadUserAvatar(ctx context.Context, tx *gorm.DB, user *types.User) error
  CreateAndUploadWarehouseAvatar(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse) error
  CreateAndUploadRoleAvatar(ctx context.Context, tx *gorm.DB, role *types.Role) (*types.Role, error)

  GenerateUserAvatar(ctx context.Context, tx *gorm.DB, user *types.User) (bytes.Buffer, error)
  GenerateCompanyAvatar(ctx context.Context, tx *gorm.DB, company *types.Company) (bytes.Buffer, error)
  GenerateWmsAvatar(ctx context.Context, tx *gorm.DB, wms *types.Wms) (bytes.Buffer, error)
  GenerateWarehouseAvatar(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse) (bytes.Buffer, error)
  GenerateRoleAvatar(ctx context.Context, tx *gorm.DB, role *types.Role) (bytes.Buffer, error)
}

type avatarService struct {
  db		  *gorm.DB
  log             *logger.Logger
  wmsRepo	  repos.WmsRepo
  companyRepo	  repos.CompanyRepo
  warehouseRepo	  repos.WarehouseRepo
  userRepo	  repos.UserRepo
  roleRepo	  repos.RoleRepo
  permissionRepo  repos.PermissionRepo
  bucketService	  BucketService
  companyIcons    []string
  wmsIcons        []string
  warehouseIcons  []string
  roleIcons	  []string
  bgColors        []color.NRGBA
  fontFace        font.Face
}

func NewAvatarService(db *gorm.DB, log *logger.Logger, wmsRepo repos.WmsRepo, companyRepo repos.CompanyRepo, warehouseRepo repos.WarehouseRepo, userRepo repos.UserRepo, roleRepo repos.RoleRepo, permissionRepo repos.PermissionRepo, bucketService BucketService) (AvatarService, error) {
  serviceLog := log.With("service", "AvatarService")


  rand.Seed(time.Now().UnixNano())

  //1) Gather list of icons in company folder && wms folder
  companyDir := os.Getenv("COMPANY_ASSET_DIR_PATH")
  if companyDir == "" {
    companyDir = "./assets/company"
  }
  companyFiles, err := findFiles(companyDir)
  if err != nil {

    return nil, fmt.Errorf("Failed scanning company icons: %w", err)
  }
  if len(companyFiles) == 0 {
    return nil, fmt.Errorf("No company icons found: %s", companyDir)
  }

  wmsDir := os.Getenv("WMS_ASSET_DIR_PATH")
  if wmsDir == "" {
    wmsDir = "./assets/wms"
  }
  wmsFiles, err := findFiles(wmsDir)
  if err != nil {
    return nil, fmt.Errorf("Failed scanning wms icons: %w", err)
  }
  if len(wmsFiles) == 0 {
    return nil, fmt.Errorf("No wms icons found: %s", wmsDir)
  }

  warehouseDir := os.Getenv("WAREHOUSE_ASSET_DIR_PATH")
  if warehouseDir == "" {
    warehouseDir = "./assets/warehouse"
  }
  warehouseFiles, err := findFiles(warehouseDir)
  if err != nil {
    return nil, fmt.Errorf("Failed scanning warehouse icons: %w", err)
  }
  if len(warehouseFiles) == 0 {
    return nil, fmt.Errorf("No warehouse icons found: %s", warehouseDir)
  }

  roleDir := os.Getenv("ROLE_ASSET_DIR_PATH")
  if roleDir == "" {
    roleDir = "./assets/role"
  }
  roleFiles, err := findFiles(roleDir)
  if err != nil {
    return nil, fmt.Errorf("Failed scanning role icons: %w", err)
  }
  if len(roleFiles) == 0 {
    return nil, fmt.Errorf("No role icons found: %s", roleDir)
  }

  //2) Get Avatar Colors
  colorsJSONPath := os.Getenv("AVATAR_COLORS_JSON_PATH")
  if colorsJSONPath == "" {
    return nil, fmt.Errorf("env var AVATAR_COLORS_JSON_PATH is empty")
  }
  serviceLog.Info("Loading avatar colors from JSON file", "path", colorsJSONPath)
  bgColors, err := loadColorsFromFile(colorsJSONPath)
  if err != nil {
    return nil, fmt.Errorf("could not load avatar colors: %w", err)
  }

  //3) Get Font
  fontPath := os.Getenv("AVATAR_FONT")
  if fontPath == "" {
    return nil, fmt.Errorf("env var AVATAR_FONT is empty")
  }
  serviceLog.Info("Loading avatar font from TTF file", "font", fontPath)
  face, err := loadFontFace(fontPath, 206)
  if err != nil {
    return nil, fmt.Errorf("could not load avatar font: %w", err)
  }

  service := &avatarService{
    db:		    db,
    log:            serviceLog,
    wmsRepo:	    wmsRepo,
    companyRepo:    companyRepo,
    warehouseRepo:  warehouseRepo,
    userRepo:	    userRepo,
    roleRepo:	    roleRepo,
    permissionRepo: permissionRepo,
    bucketService:  bucketService,
    companyIcons:   companyFiles,
    wmsIcons:       wmsFiles,
    warehouseIcons: warehouseFiles,
    roleIcons:	    roleFiles,
    bgColors:       bgColors,
    fontFace:       face,
  }
  return service, nil
}

func (as *avatarService) CreateAndUploadWmsAvatar(ctx context.Context, tx *gorm.DB, wms *types.Wms) error {
  buf, err := as.GenerateWmsAvatar(ctx, tx, wms)
  if err != nil {
    return err
  }
  bucketKey := fmt.Sprintf("wms_avatars/%s.png", wms.ID.String())
  if err := as.bucketService.UploadFile(ctx, tx, bucketKey, bytes.NewReader(buf.Bytes())); err != nil {
    return fmt.Errorf("Failed to upload wms avatar: %w", err)
  }
  if wms.AvatarBucketKey != bucketKey {
    wms.AvatarBucketKey = bucketKey
  }
  finalURL := as.bucketService.GetPublicURL(bucketKey)
  if wms.AvatarURL != finalURL {
    wms.AvatarURL = finalURL
  }
  return nil
}

func (as *avatarService) CreateAndUploadCompanyAvatar(ctx context.Context, tx *gorm.DB, company *types.Company) error {
  buf, err := as.GenerateCompanyAvatar(ctx, tx, company)
  if err != nil {
    return err
  }
  bucketKey := fmt.Sprintf("company_avatars/%s.png", company.ID.String())
  if err := as.bucketService.UploadFile(ctx, tx, bucketKey, bytes.NewReader(buf.Bytes())); err != nil {
    return fmt.Errorf("Failed to upload company avatar: %w", err)
  }
  if company.AvatarBucketKey != bucketKey {
    company.AvatarBucketKey = bucketKey
  }
  finalURL := as.bucketService.GetPublicURL(bucketKey)
  if company.AvatarURL != finalURL {
    company.AvatarURL = finalURL
  }
  return nil
}

func (as *avatarService) CreateAndUploadUserAvatar(ctx context.Context, tx *gorm.DB, user *types.User) error {
  buf, err := as.GenerateUserAvatar(ctx, tx, user)
  if err != nil {
    return err
  }
  bucketKey := fmt.Sprintf("user_avatars/%s.png", user.ID.String())
  if err := as.bucketService.UploadFile(ctx, tx, bucketKey, bytes.NewReader(buf.Bytes())); err != nil {
    return fmt.Errorf("Failed to upload user avatar: %w", err)
  }
  if user.AvatarBucketKey != bucketKey {
    user.AvatarBucketKey = bucketKey
  }
  finalURL := as.bucketService.GetPublicURL(bucketKey)
  if user.AvatarURL != finalURL {
    user.AvatarURL = finalURL
  }
  return nil
}

func (as *avatarService) CreateAndUploadWarehouseAvatar(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse) error {
  buf, err := as.GenerateWarehouseAvatar(ctx, tx, warehouse)
  if err != nil {
    return err
  }
  bucketKey := fmt.Sprintf("warehouse_avatars/%s.png", warehouse.ID.String())
  if err := as.bucketService.UploadFile(ctx, tx, bucketKey, bytes.NewReader(buf.Bytes())); err != nil {
    return fmt.Errorf("Failed to upload warehouse avatar: %w", err)
  }
  if warehouse.AvatarBucketKey != bucketKey {
    warehouse.AvatarBucketKey = bucketKey
  }
  finalURL := as.bucketService.GetPublicURL(bucketKey)
  if warehouse.AvatarURL != finalURL {
    warehouse.AvatarURL = finalURL
  }
  return nil
}

func (as *avatarService) CreateAndUploadRoleAvatar(ctx context.Context, tx *gorm.DB, role *types.Role) (*types.Role, error) {
  buf, err := as.GenerateRoleAvatar(ctx, tx, role)
  if err != nil {
    return err
  }
  bucketKey := fmt.Sprintf("role_avatar/%s.png", role.ID.String())
  if err := as.bucketService.UploadFile(ctx, tx, bucketKey, bytes.NewReader(buf.Bytes())); err != nil {
    return fmt.Errorf("failed to upload role avatar: %w", err)
  }
  if role.AvatarBucketKey != bucketKey {
    role.AvatarBucketKey = bucketKey
  }
  finalURL := as.bucketService.GetPublicURL(bucketKey)
  if role.AvatarURL != finalURL {
    role.AvatarURL = finalURL
  }
  return role, nil
}


func (as *avatarService) GenerateUserAvatar(ctx context.Context, tx *gorm.DB, user *types.User) (bytes.Buffer, error) {
	const size = 512

	// 1) Create drawing context
	dc := gg.NewContext(size, size)

	// 2) Circular mask so final image is round
	dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
	dc.Clip()

	// 3) Use a single solid background color (no gradient)
	base := as.bgColors[rand.Intn(len(as.bgColors))]
	dc.SetColor(base)
	dc.DrawRectangle(0, 0, float64(size), float64(size))
	dc.Fill()

	// 4) Compute user initials
	initials := computeInitials(user.FirstName, user.LastName)

	// 5) Set font & measure text
	dc.SetFontFace(as.fontFace)
	tw, th := dc.MeasureString(initials)
	cx, cy := float64(size)/2, float64(size)/2

	// 6) Draw main white text
	dc.SetColor(color.White)
	dc.DrawString(initials, cx-(tw/2)+5, cy+(th/2)-10)

	// 7) Export to PNG
	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return buf, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf, nil
}

func (as *avatarService) GenerateCompanyAvatar(ctx context.Context, tx *gorm.DB, company *types.Company) (bytes.Buffer, error) {
	const size = 512
	dc := gg.NewContext(size, size)

	// Circular mask
	dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
	dc.Clip()

	// Solid color background
	base := as.bgColors[rand.Intn(len(as.bgColors))]
	dc.SetColor(base)
	dc.DrawRectangle(0, 0, float64(size), float64(size))
	dc.Fill()

	// Load and colorize icon
	iconPath := as.companyIcons[rand.Intn(len(as.companyIcons))]
	iconImg, err := imaging.Open(iconPath)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to open company icon: %w", err)
	}
	whiteIcon := colorizeImageWhite(iconImg)

	// Scale the icon (adjust maxIconSize as you need)
	maxIconSize := float64(size) * 0.5
	whiteIcon = imaging.Fit(whiteIcon, int(maxIconSize), int(maxIconSize), imaging.Lanczos)

	// Draw the icon with no shadow or offset
	dc.DrawImageAnchored(whiteIcon, size/2, size/2, 0.5, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return buf, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf, nil
}

func (as *avatarService) GenerateWmsAvatar(ctx context.Context, tx *gorm.DB, wms *types.Wms) (bytes.Buffer, error) {
	const size = 512
	dc := gg.NewContext(size, size)

	// Circular mask
	dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
	dc.Clip()

	// Solid color background
	base := as.bgColors[rand.Intn(len(as.bgColors))]
	dc.SetColor(base)
	dc.DrawRectangle(0, 0, float64(size), float64(size))
	dc.Fill()

	// Load and colorize icon
	iconPath := as.wmsIcons[rand.Intn(len(as.wmsIcons))]
	iconImg, err := imaging.Open(iconPath)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to open WMS icon: %w", err)
	}
	whiteIcon := colorizeImageWhite(iconImg)

	// Scale the icon (adjust maxIconSize if needed)
	maxIconSize := float64(size) * 0.5
	whiteIcon = imaging.Fit(whiteIcon, int(maxIconSize), int(maxIconSize), imaging.Lanczos)

	// Draw the icon with no shadow or offset
	dc.DrawImageAnchored(whiteIcon, size/2, size/2, 0.5, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return buf, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf, nil
}

func (as *avatarService) GenerateWarehouseAvatar(ctx context.Context, tx *gorm.DB, warehouse *types.Warehouse) (bytes.Buffer, error) {
  const size = 512
  dc := gg.NewContext(size, size)

  dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
  dc.Clip()

  base := as.bgColors[rand.Intn(len(as.bgColors))]
  dc.SetColor(base)
  dc.DrawRectangle(0, 0, float64(size), float64(size))
  dc.Fill()

  iconPath := as.warehouseIcons[rand.Intn(len(as.warehouseIcons))]
  iconImg, err := imaging.Open(iconPath)
  if err != nil {
    return bytes.Buffer{}, fmt.Errorf("failed to open warehouse icon: %w", err)
  }
  whiteIcon := colorizeImageWhite(iconImg)
  maxIconSize := float64(size) * 0.5
  whiteIcon = imaging.Fit(whiteIcon, int(maxIconSize), int(maxIconSize), imaging.Lanczos)

  dc.DrawImageAnchored(whiteIcon, size/2, size/2, 0.5, 0.5)

  var buf bytes.Buffer
  if err := dc.EncodePNG(&buf); err != nil {
    return buf, fmt.Errorf("failed to encode PNG: %w", err)
  }
  return buf, nil
}

func (as *avatarService) GenerateRoleAvatar(ctx context.Context, tx *gorm.DB, role *types.Role) (bytes.Buffer, error) {
  iconPath := as.roleIcons[rand.Intn(len(as.roleIcons))]

  img, err := imaging.Open(iconPath)
  if err != nil {
    return bytes.Buffer{}, fmt.Errorf("failed to open role icon %q: %w", iconPath, err)
  }

  img = imaging.Fit(img, 256, 256, imaging.Lanczos)

  var buf bytes.Buffer
  if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
    return buf, fmt.Errorf("failed to encode role avatar PNG: %w", err)
  }
  return buf, nil
}



//----------------------------------------------------------------------------------------
// Helpers
//----------------------------------------------------------------------------------------
func colorizeImageWhite(img image.Image) image.Image {
  bounds := img.Bounds()
  out := image.NewNRGBA(bounds)
  for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
      _, _, _, a := img.At(x, y).RGBA()
      alpha8 := uint8(a >> 8)
      out.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: alpha8})
    }
  }
  return out
}

func computeInitials(first, last string) string {
  fInit := "?"
  if len(first) > 0 {
    fInit = strings.ToUpper(first[:1])
  }
  lInit := "?"
  if len(last) > 0 {
    lInit = strings.ToUpper(last[:1])
  }
  return fInit + lInit
}

func lightenOrDarken(c color.NRGBA, fraction float64) color.NRGBA {
  clamp := func(v float64) uint8 {
    return uint8(math.Max(0, math.Min(255, v)))
  }
  rf := float64(c.R)
  gf := float64(c.G)
  bf := float64(c.B)
  delta := 255.0 * fraction
  rf = rf + delta
  gf = gf + delta
  bf = bf + delta
  return color.NRGBA{
    R: clamp(rf),
    G: clamp(gf),
    B: clamp(bf),
    A: c.A,
  }
}

func findFiles(dir string) ([]string, error) {
  entries, err := os.ReadDir(dir)
  if err != nil {
    return nil, err
  }
  var paths []string
  for _, e := range entries {
    if e.IsDir() {
      continue
    }
    name := e.Name()
    if strings.HasSuffix(strings.ToLower(name), ".png") {
      fullPath := filepath.Join(dir, name)
      paths = append(paths, fullPath)
    }
  }
  return paths, nil
}

func loadColorsFromFile(jsonPath string) ([]color.NRGBA, error) {
  data, err := ioutil.ReadFile(jsonPath)
  if err != nil {
    return nil, fmt.Errorf("read file error: %w", err)
  }
  var colors []color.NRGBA
  if err := json.Unmarshal(data, &colors); err != nil {
    return nil, fmt.Errorf("json unmarshal error: %w", err)
  }
  return colors, nil
}

func loadFontFace(fontPath string, size float64) (font.Face, error) {
  fontBytes, err := ioutil.ReadFile(fontPath)
  if err != nil {
    return nil, fmt.Errorf("failed to read font file: %w", err)
  }
  parsedFont, err := truetype.Parse(fontBytes)
  if err != nil {
    return nil, fmt.Errorf("failed to parse TTF: %w", err)
  }
  face := truetype.NewFace(parsedFont, &truetype.Options{
    Size:     size,
    DPI:      72,
    Hinting:  font.HintingNone,
  })
  return face, nil
}
