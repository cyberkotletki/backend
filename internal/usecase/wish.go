package usecase

import (
	"backend/internal/entity"
	"context"
)

type WishUsecase interface {
	AddWish(ctx context.Context, req entity.AddWishRequest) (string, error)
	UpdateWish(ctx context.Context, req entity.UpdateWishRequest) error
	GetWishes(ctx context.Context, streamerUUID string) ([]entity.WishResponse, error)
}
