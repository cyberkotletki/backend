package repo

import (
	"backend/internal/entity"
	"context"
)

// DonationEventRepo описывает методы для отправки событий о донате
// в брокере сообщений (например, Redis).
type DonationEventRepo interface {
	SendDonationEvent(ctx context.Context, event entity.DonationEvent) error
	// SubscribeDonationEvents возвращает канал событий доната для указанного стримера
	SubscribeDonationEvents(ctx context.Context, streamerUUID string, lastID string) (<-chan entity.DonationEvent, <-chan error)
}
