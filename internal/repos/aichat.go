/*package repos

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/yungbote/slotter/backend/internal/logger"
    "github.com/yungbote/slotter/backend/internal/types"
)

type AiChatConvoRepo interface {
    CreateConvos(ctx context.Context, tx *gorm.DB, convs []*types.AiChatConvo, error)
    GetConvosByUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*types.AiChatConvo, error)
    GetConvosByID(ctx context.Context, tx *gorm.DB, convID uuid.UUID) (*types.AiChatConvo, error)
    DeleteConvoByID(ctx context.Context, tx *gorm.DB, convID uuid.UUID) error
}

type AiChatMessageRepo interface {
    CreateMessages(ctx context.Context, tx *gorm.DB, msgs []*types.AiChatMessage) ([]*types.AiChatMessage, error)
    GetMessagesByConvo(ctx context.Context, tx *gorm.DB, convID uuid.UUID) ([]*types.AiChatMessage, error)
}

type aiChatConvoRepo struct {
    db          *gorm.DB
    log         *logger.Logger
}

func NewAiChatConvoRepo(db *gorm.DB, baseLog *logger.Logger) AiChatConvoRepo {
    repoLog := baseLog.With("repo", "AiChatConvoRepo")
    return &aiChatConvoRepo{db: db, log: repoLog}
}

func (r *aiChatConvoRepo) CreateConversations(ctx context.Context, tx *gorm.DB, convs []*types.AiChatConvo) ([]*types.AiChatConvo, error) {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    if err := transaction.WithContext(ctx).Create(&convs).Error; err != nil {
        return nil, fmt.Errorf("Failed creating conversation: %w", err)
    }
    return convs, nil
}

func (r *aiChatConvoRepo) GetConversationsByUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*types.AiChatConvo, error) {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    var results []*types.AiChatConvo
    if err := transaction.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("created_at DESC").
        Find(&results).Error; err != nil {
        return nil, fmt.Errorf("Failed fetching conversations for user: %w", err)
    }
    return results, nil
}

func (r *aiChatConvoRepo) GetConversationByID(ctx context.Context, tx *gorm.DB, convID uuid.UUID) ([]*types.AiChatConvo, error) {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    var conv types.AiChatConvo
    if err := transaction.WithContext(ctx).
        Where("id = ?", convID).
        First(&conv).Error; err != nil {
        return nil, err
    }
    return &conv, nil
}

func (r *aiChatConvoRepo) DeleteConversationByID(ctx context.Context, tx *gorm.DB, convID uuid.UUID) error {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    if err := transaction.WithContext(ctx).
        Where("id = ?", convID).
        Delete(&types.AiChatConvo{}).Error; err != nil {
        return fmt.Errorf("Failed deleting conversation: %w", err)
    }
    return nil
}

type aiChatMessageRepo struct {
    db              *gorm.DB
    log             *logger.Logger
}

func NewAiChatMessageRepo(db *gorm.DB, baseLog *logger.Logger) AiChatMessageRepo {
    repoLog := baseLog.With("repo", "AiChatMessageRepo")
    return &aiChatMessageRepo{db: db, log: repoLog}
}

func (r *aiChatMessageRepo) CreateMessages(ctx context.Context, tx *gorm.DB, msgs []*types.AiChatMessage, error) {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    if err := transaction.WithContext(ctx).Create(&msgs).Error; err != nil {
        return nil, fmt.Errorf("Failed creating messages: %w", err)
    }
    return msgs, nil
}

func (r *aiChatMessageRepo) GetMessagesByConversation(ctx context.Context, tx *gorm.DB, convID uuid.UUID) ([]*types.AiChatMessage, error) {
    transaction := tx
    if transaction == nil {
        transaction = r.db
    }
    var results []*types.AiChatMessage
    if err := transaction.WithContext(ctx).
        Where("conv_id = ?", convID).
        Order("created_at ASC").
        Find(&results).Error; err != nil {
        return nil, fmt.Errorf("Failed fetching messages by conversation: %w", err)
    }
    return results, nil
}*/
