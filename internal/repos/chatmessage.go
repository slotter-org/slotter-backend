package repos

import (
    "context"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
    
    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type ChatMessageRepo interface {
    CreateMessages(ctx context.Context, tx *gorm.DB, msgs []*types.ChatMessage) ([]*types.ChatMessage, error)
    GetBySessionID(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) ([]*types.ChatMessage, error)
}

type chatMessageRepo struct {
    db      *gorm.DB
    log     *logger.Logger
}

func NewChatMessageRepo(db *gorm.DB, baseLog *logger.Logger) ChatMessageRepo {
    return &chatMessageRepo{
        db:     db,
        log:    baseLog.With("repo", "ChatMessageRepo"),
    }
}

func (cmr *chatMessageRepo) CreateMessages(ctx context.Context, tx *gorm.DB, msgs []*types.ChatMessage) ([]*types.ChatMessage, error) {
    if tx == nil {
        tx = cr.db
    }
    if len(msgs) == 0 {
        return msgs, nil
    }
    if err := tx.WithContext(ctx).Create(&msgs).Error; err != nil {
        cr.log.Error("failed to create chat messages", "error", err)
        return nil, err
    }
    return msgs, nil
}

func (cmr *chatMessageRepo) GetBySessionID(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) ([]*types.ChatMessage, error) {
    if tx == nil {
        tx = cr.db
    }
    var msgs []*types.ChatMessage
    if err := tx.WithContext(ctx).
        Where("session_id = ?", sessionID).
        Order("created_at ASC").
        Find(&msgs).Error; err != nil {
        cr.log.Error("failed to get chat messages by sessionID", "error", err)
        return nil, err
    }
    return msgs, nil
}
