package types

import (
  "time"
  
  "gorm.io/gorm"
  "github.com/google/uuid"
)

type ChatSession struct {
  gorm.Model

  ID          uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  UserID      uuid.UUID         `gorm:"index"`
  Title       string            `gorm:"column:title"`
  CreatedAt   time.Time         `gorm:"not null;default:now()"`
  UpdatedAt   time.Time         `gorm:"not null;default:now()"`
}

func (ChatSession) TableName() string {
  return "chat_session"
}
