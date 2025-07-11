package usecase

import (
	"backend/internal/entity"
	"context"
	"errors"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrInvalidRegisterRequest = errors.New("invalid register request")
	ErrInvalidUpdateRequest   = errors.New("invalid update request")
)

type UserUsecase interface {
	Register(ctx context.Context, req entity.RegisterUserRequest) (string, error)
	UpdateProfile(ctx context.Context, req entity.UpdateUserRequest) error
	GetProfile(ctx context.Context, uuid string) (*entity.UserProfileResponse, error)
	GetHistory(ctx context.Context, uuid string, page int, pageSize int) (*entity.UserHistoryResponse, error)
	GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error)
}
