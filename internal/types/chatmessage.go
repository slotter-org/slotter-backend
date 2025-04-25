package types

import (
  "time"
  
  "gorm.io/gorm"
  "github.com/google/uuid"
)

type ChatMessage struct {
  gorm.Model

  ID          uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
  SessionID   uuid.UUID       `gorm:"index"`
  UserID      *uuid.UUID      `gorm:"index;null"`
  Role        string          `gorm:"column:role"`
  Content     string          `gorm:"column:content"`
  CreatedAt   time.Time       `gorm:"not null;default:now()"`
  UpdatedAt   time.Time       `gorm:"not null;default:now()"`
}

func (ChatMessage) TableName() string {
  return "chat_message"
}
