package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"backend/internal/usecase"
	"context"
)

type donationEventUsecase struct {
	repo repo.DonationEventRepo
}

func NewDonationEventUsecase(repo repo.DonationEventRepo) usecase.DonationEventUsecase {
	return &donationEventUsecase{repo: repo}
}

func (u *donationEventUsecase) SubscribeDonationEvents(ctx context.Context, streamerUUID string, lastID string) (<-chan entity.DonationEvent, <-chan error) {
	return u.repo.SubscribeDonationEvents(ctx, streamerUUID, lastID)
}
