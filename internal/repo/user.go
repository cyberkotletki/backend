package repo

import (
	"backend/internal/entity"
	"context"
)

type UserRepository interface {
	Register(ctx context.Context, user *entity.User) (string, error)
	Update(ctx context.Context, user *entity.User) error
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error)
}
