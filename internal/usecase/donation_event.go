package usecase

import (
	"backend/internal/entity"
	"context"
)

type DonationEventUsecase interface {
	SubscribeDonationEvents(ctx context.Context, streamerUUID string, lastID string) (<-chan entity.DonationEvent, <-chan error)
}
