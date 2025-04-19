/*package types

import (
  "time"
  
  "github.com/google/uuid"
  "gorm.io/gorm"
)

type AiChatConvo struct {
  gorm.Model
  ID              uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  UserID          uuid.UUID         `gorm:"index;not null"`
  User            *User             `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
  Title           string            `gorm:"column:title"`
  CreatedAt       time.Time         `gorm:"not null;default:now()"`
  UpdatedAt       time.Time         `gorm:"not null;default:now()"`
}

type AiChatMessage struct {
  gorm.Model
  ID              uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  ConvID          uuid.UUID         `gorm:"index;not null"`
  Conv            *AiChatConvo      `gorm:"constraint:OnDelete:CASCADE;foreignKey:ConvID;references:ID"`
  Role            string            `gorm:"column:role;not null"`
  Content         string            `gorm:"column:content;type:text;not null"`
  CreatedAt       time.Time         `gorm:"not null;default:now()"`
  UpdatedAt       time.Time         `gorm:"not null;default:now()"`
}
*/
