package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type Permission struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  PermissionType      string                    `gorm:"uniqueIndex;not null;column:permission_type" json:"permission_type"`
  Roles               []*Role                   `gorm:"many2many:permissions_roles;"`


  Name                string                    `gorm:"column:name" json:"name"`
  Category            string                    `gorm:"column:category" json:"category"`
  Action              string                    `gorm:"column:action" json:"action"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()"`
}

func (Permission) TableName() string {
  return "permission"
}
