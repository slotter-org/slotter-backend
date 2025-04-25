package services

import (
  "context"
  "fmt"
  "time"

  "github.com/google/uuid"
  "gorm.io/gorm"

  "github.com/slotter-org/slotter-backend/internal/requestdata"
  "github.com/slotter-org/slotter-backend/internal/logger"
  "github.com/slotter-org/slotter-backend/internal/repos"
  "github.com/slotter-org/slotter-backend/internal/types"
)

type AiChatService interface {
  //Session Level
  StartNewSession(ctx context.Context, title string) (*types.ChatSession, error)
  EndSession(ctx context.Context, sessionID uuid.UUID) error
  GetUserSessions(ctx context.Context) ([]types.ChatSession, error)
  //Message Level
  SendUserMessage(ctx context.Context, sessionID uuid.UUID, content string, role string) (types.ChatMessage, error)

}
