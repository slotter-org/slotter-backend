package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type Company struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  WmsID               *uuid.UUID                `gorm:"index" json:"wmsID,omitempty"`
  Wms                 *Wms                      `gorm:"constraint:OnDelete:CASCADE;foreignKey:WmsID;references:ID" json:"wms,omitempty"`
  DefaultRoleID       *uuid.UUID                 `gorm:"index" json:"defaultRoleID,omitempty"`
  DefaultRole         *Role                     `gorm:"constraint:OnDelete:SET NULL;foreignKey:DefaultRoleID;references:ID" json:"defaultRole,omitempty"`
  Users               []*User                   `gorm:"foreignKey:CompanyID" json:"users,omitempty"`

  Name                string                    `gorm:"column:name" json:"name"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()" json:"createdAt"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()" json:"updatedAt"`
}

func (Company) TableName() string {
  return "company"
}
