package repo

import (
	"backend/internal/entity"
	"context"
)

type HistoryRepository interface {
	Add(ctx context.Context, history *entity.History) error
	GetByStreamerUUID(ctx context.Context, streamerUUID string, page int, pageSize int) ([]*entity.History, error)
}
