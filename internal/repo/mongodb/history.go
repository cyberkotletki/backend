package mongodb

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type historyRepository struct {
	col *mongo.Collection
}

func NewHistoryRepository(db *mongo.Database) repo.HistoryRepository {
	return &historyRepository{
		col: db.Collection("history"),
	}
}

func (r *historyRepository) Add(ctx context.Context, history *entity.History) error {
	_, err := r.col.InsertOne(ctx, history)
	return err
}

func (r *historyRepository) GetByStreamerUUID(ctx context.Context, streamerUUID string, page int, pageSize int) ([]*entity.History, error) {
	filter := bson.M{"streamer_uuid": streamerUUID}
	findOptions := options.Find().SetSort(bson.M{"datetime": -1}).SetSkip(int64((page - 1) * pageSize)).SetLimit(int64(pageSize))
	cursor, err := r.col.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var historyList []*entity.History
	for cursor.Next(ctx) {
		var h entity.History
		if err := cursor.Decode(&h); err != nil {
			return nil, err
		}
		historyList = append(historyList, &h)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return historyList, nil
}
