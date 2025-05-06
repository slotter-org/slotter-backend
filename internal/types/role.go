package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type Role struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  WmsID               *uuid.UUID                `gorm:"index" json:"wmsID,omitempty"`
  Wms                 *Wms                      `gorm:"constraint:OnDelete:CASCADE;foreignKey:WmsID;references:ID" json:"wms,omitempty"`
  CompanyID           *uuid.UUID                `gorm:"index" json:"companyID,omitempty"`
  Company             *Company                  `gorm:"constraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID" json:"company,omitempty"`
  Users               []*User                   `gorm:"foreignKey:RoleID" json:"users,omitempty"`
  Permissions         []*Permission             `gorm:"many2many:permissions_roles;" json:"permissions,omitempty"`


  Name                string                    `gorm:"column:name" json:"name"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()" json:"createdAt"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()" json:"updatedAt"`
}

func (Role) TableName() string {
  return "role"
}
