package types

import (
  "time"

  "github.com/google/uuid"
  "gorm.io/gorm"
)

type Warehouse struct {
  gorm.Model

  ID                  uuid.UUID             `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  Name                string                `gorm:"not null;column:name" json:"name"`
  AvatarBucketKey     string                `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                `gorm:"column:avatar_url" json:"avatarURL"`
  CompanyID           uuid.UUID             `gorm:"index;not null" json:"companyID"`
  Company             *Company              `gorm:"contraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID" json:"company,omitempty"`
  CreatedAt           time.Time             `gorm:"not null;default:now()" json:"createdAt"`
  UpdatedAt           time.Time             `gorm:"not null;default:now()" json:"updatedAt"`
}

func (Warehouse) TableName() string {
  return "warehouse"
}
