package repos

import (
    "context"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
    
    "github.com/slotter-org/slotter-backend/internal/logger"
    "github.com/slotter-org/slotter-backend/internal/types"
)

type ChatSessionRepo interface {
    CreateSession(ctx context.Context, tx *gorm.DB, session *types.ChatSession) (*types.ChatSession, error)
    GetSessionByID(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*types.ChatSession, error)
    GetUserSessions(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*types.ChatSession, error)
    EndSession(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
}

type chatSessionRepo struct {
    db      *gorm.DB
    log     *logger.Logger
}

func NewChatSessionRepo(db *gorm.DB, baseLog *logger.Logger) ChatSessionRepo {
    return &chatSessionRepo{
        db: db,
        log: baseLog.With("repo", "ChatSessionRepo"),
    }
}

func (csr *chatSessionRepo) CreateSession(ctx context.Context, tx *gorm.DB, session *types.ChatSession) (*types.ChatSession, error) {
    if tx == nil {
        tx = csr.db
    }
    if session.ID == uuid.Nil {
        session.ID = uuid.New()
    }
    if err := tx.WithContext(ctx).Create(session).Error; err != nil {
        csr.log.Error("failed to create chat session", "error", err)
        return nil, err
    }
    return session, nil
}

func (csr *chatSessionRepo) GetSessionByID(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*types.ChatSession, error) {
    if tx == nil {
        tx = csr.db
    }
    var s types.ChatSession
    if err := tx.WithContext(ctx).
        Where("id = ?", id).
        First(&s).Error; err != nil {
        return nil, err
    }
    return &s, nil
}

func (csr *chatSessionRepo) GetUserSessions(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*Types.ChatSession, error) {
    if tx == nil {
        tx = csr.db
    }
    var sessions []types.ChatSession
    if err := tx.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("created_at DESC").
        Find(&sessions).Error; err != nil {
        return nil, err
    }
    return sessions, nil
}

func (csr *chatSessionRepo) EndSession(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
    if tx == nil {
        tx = csr.db
    }
    now := time.Now()
    if err := tx.WithContext(ctx).
        Model(&types.ChatSession{}).
        Where("id = ?", id).
        Update("ended_at", &now).Error; err != nil {
        return err
    }
    return nil
}











