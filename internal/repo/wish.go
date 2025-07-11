package repo

import (
	"backend/internal/entity"
	"context"
	"errors"
)

var (
	ErrWishNotFound = errors.New("wish not found")
)

type WishRepository interface {
	Add(ctx context.Context, wish *entity.Wish) (string, error)
	Update(ctx context.Context, wish *entity.Wish) error
	GetByUUID(ctx context.Context, uuid string) (*entity.Wish, error)
	GetByStreamerUUID(ctx context.Context, streamerUUID string) ([]*entity.Wish, error)
}
