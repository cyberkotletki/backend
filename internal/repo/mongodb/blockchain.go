package mongodb

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type blockchainRepository struct {
	stateCol  *mongo.Collection
	eventsCol *mongo.Collection
}

func NewBlockchainRepository(db *mongo.Database) repo.BlockchainRepository {
	return &blockchainRepository{
		stateCol:  db.Collection("blockchain_state"),
		eventsCol: db.Collection("blockchain_events"),
	}
}

const lastProcessedBlockID = "last_processed_block"

func (r *blockchainRepository) GetLastProcessedBlock(ctx context.Context) (uint64, error) {
	filter := bson.M{"_id": lastProcessedBlockID}
	var state entity.BlockchainState
	err := r.stateCol.FindOne(ctx, filter).Decode(&state)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Если записи нет, возвращаем 0
			return 0, nil
		}
		return 0, err
	}
	return state.LastProcessedBlock, nil
}

func (r *blockchainRepository) SaveLastProcessedBlock(ctx context.Context, blockNumber uint64) error {
	filter := bson.M{"_id": lastProcessedBlockID}
	update := bson.M{
		"$set": bson.M{
			"last_processed_block": blockNumber,
			"updated_at":           time.Now(),
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.stateCol.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *blockchainRepository) SaveEvent(ctx context.Context, event *entity.BlockchainEvent) error {
	event.ProcessedAt = time.Now()
	_, err := r.eventsCol.InsertOne(ctx, event)
	return err
}

func (r *blockchainRepository) GetEvents(ctx context.Context, fromBlock, toBlock uint64) ([]*entity.BlockchainEvent, error) {
	filter := bson.M{
		"block_number": bson.M{
			"$gte": fromBlock,
			"$lte": toBlock,
		},
	}
	findOptions := options.Find().SetSort(bson.M{"block_number": 1, "processed_at": 1})
	cursor, err := r.eventsCol.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*entity.BlockchainEvent
	for cursor.Next(ctx) {
		var event entity.BlockchainEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
