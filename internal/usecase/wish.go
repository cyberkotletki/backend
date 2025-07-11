package usecase

import (
	"backend/internal/entity"
	"context"
	"errors"
)

var (
	ErrWishNotFound = errors.New("wish not found")
	ErrInvalidWish  = errors.New("invalid wish")
)

type WishUsecase interface {
	AddWish(ctx context.Context, req entity.AddWishRequest) (string, error)
	UpdateWish(ctx context.Context, req entity.UpdateWishRequest) error
	GetWishes(ctx context.Context, streamerUUID string) ([]entity.WishResponse, error)
}
