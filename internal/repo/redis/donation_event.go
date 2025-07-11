package redis

import (
	"backend/internal/entity"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type DonationEventRepo struct {
	client    *redis.Client
	streamKey string
}

// NewDonationEventRepo создаёт новый репозиторий для отправки событий доната в Redis Stream
func NewDonationEventRepo(client *redis.Client, streamKey string) *DonationEventRepo {
	return &DonationEventRepo{client: client, streamKey: streamKey}
}

func (r *DonationEventRepo) SendDonationEvent(ctx context.Context, event entity.DonationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal donation event: %w", err)
	}
	res := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.streamKey,
		Values: map[string]interface{}{
			"event": data,
		},
	})
	return res.Err()
}

func (r *DonationEventRepo) SubscribeDonationEvents(ctx context.Context, streamerUUID string, lastID string) (<-chan entity.DonationEvent, <-chan error) {
	eventCh := make(chan entity.DonationEvent)
	errCh := make(chan error, 1)
	if lastID == "" {
		lastID = "$"
	}
	go func() {
		defer close(eventCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				res, err := r.client.XRead(ctx, &redis.XReadArgs{
					Streams: []string{r.streamKey, lastID},
					Block:   30 * 1000, // 30 секунд в миллисекундах
					Count:   10,
				}).Result()
				if err != nil && !errors.Is(err, redis.Nil) {
					errCh <- fmt.Errorf("redis XRead error: %w", err)
					return
				}
				for _, stream := range res {
					for _, msg := range stream.Messages {
						lastID = msg.ID
						var data []byte
						if b, ok := msg.Values["event"].([]byte); ok {
							data = b
						} else if s, ok := msg.Values["event"].(string); ok {
							data = []byte(s)
						} else {
							continue
						}
						var event entity.DonationEvent
						if err := json.Unmarshal(data, &event); err != nil {
							continue
						}
						if event.StreamerUUID != streamerUUID {
							continue
						}
						eventCh <- event
					}
				}
			}
		}
	}()
	return eventCh, errCh
}
