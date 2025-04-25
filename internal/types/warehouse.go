package types

import (
  "time"

  "github.com/google/uuid"
  "gorm.io/datatypes"
)

type Warehouse struct {
  gorm.Model

  ID              uuid.UUID             `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  Name            string                `gorm:"not null;column:name"`
  CompanyID       uuid.UUID             `gorm:"index;not null"`
  Company         *Company              `gorm:"contraint:OnDelete:CASCADE;foreignKey:CompanyID;references:ID"`
  CreatedAt       time.Time             `gorm:"not null;default:now()"`
  UpdatedAt       time.Time             `gorm:"not null;default:now()"`
}

func (Warehouse) TableName() string {
  return "warehouse"
}
