package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type Wms struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
  DefaultRoleID       *uuid.UUID                `gorm:"index" json:"defaultRoleID,omitempty"`
  DefaultRole         *Role                     `gorm:"constraint:OnDelete:SET NULL;foreignKey:DefaultRoleID;references:ID" json:"defaultRole,omitempty"`
  Companies           []*Company                `gorm:"foreignKey:WmsID", json:"companies,omitempty"`
  Users               []*User                   `gorm:"foreignKey:WmsID" json:"users,omitempty"`


  Name                string                    `gorm:"column:name" json:"name"`
  AvatarBucketKey     string                    `gorm:"column:avatar_bucket_key" json:"avatarBucketKey,omitempty"`
  AvatarURL           string                    `gorm:"column:avatar_url" json:"avatarURL,omitempty"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()" json:"createdAt"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()" json:"updatedAt"`
}

func (Wms) TableName() string {
  return "wms"
}
