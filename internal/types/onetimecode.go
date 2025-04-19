package types

import (
  "time"

  "gorm.io/gorm"
  "github.com/google/uuid"
)

type OneTimeCode struct {
  gorm.Model
  ID                  uuid.UUID                 `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  UserID              uuid.UUID                 `gorm:"index;not null"`
  User                *User                     `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`

  Code                string                    `gorm:"uniqueIndex;not null;column:code"`
  ExpiresAt           time.Time                 `gorm:"column:expires_at"`
  Used                bool                      `gorm:"not null;default:false"`

  CreatedAt           time.Time                 `gorm:"not null;default:now()"`
  UpdatedAt           time.Time                 `gorm:"not null;default:now()"`
}

func (OneTimeCode) TableName() string {
  return "one_time_code"
}

