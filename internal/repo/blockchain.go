package repo

import (
	"backend/internal/entity"
	"context"
)

type BlockchainRepository interface {
	GetLastProcessedBlock(ctx context.Context) (uint64, error)
	SaveLastProcessedBlock(ctx context.Context, blockNumber uint64) error
	SaveEvent(ctx context.Context, event *entity.BlockchainEvent) error
	GetEvents(ctx context.Context, fromBlock, toBlock uint64) ([]*entity.BlockchainEvent, error)
}
