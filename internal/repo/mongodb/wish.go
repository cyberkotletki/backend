package mongodb

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type wishRepository struct {
	col *mongo.Collection
}

func NewWishRepository(db *mongo.Database) repo.WishRepository {
	return &wishRepository{
		col: db.Collection("wishes"),
	}
}

func (r *wishRepository) Add(ctx context.Context, wish *entity.Wish) (string, error) {
	wish.CreatedAt = time.Now()
	wish.UpdatedAt = wish.CreatedAt
	_, err := r.col.InsertOne(ctx, wish)
	if err != nil {
		return "", err
	}
	return wish.UUID, nil
}

func (r *wishRepository) Update(ctx context.Context, wish *entity.Wish) error {
	wish.UpdatedAt = time.Now()
	filter := bson.M{"uuid": wish.UUID}
	update := bson.M{"$set": wish}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return repo.ErrWishNotFound
	}
	return nil
}

func (r *wishRepository) GetByUUID(ctx context.Context, uuid string) (*entity.Wish, error) {
	filter := bson.M{"uuid": uuid}
	var wish entity.Wish
	err := r.col.FindOne(ctx, filter).Decode(&wish)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repo.ErrWishNotFound
		}
		return nil, err
	}
	return &wish, nil
}

func (r *wishRepository) GetByStreamerUUID(ctx context.Context, streamerUUID string) ([]*entity.Wish, error) {
	filter := bson.M{"streamer_uuid": streamerUUID}
	findOptions := options.Find().SetSort(bson.M{"is_priority": -1, "created_at": -1})
	cursor, err := r.col.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var wishes []*entity.Wish
	for cursor.Next(ctx) {
		var w entity.Wish
		if err := cursor.Decode(&w); err != nil {
			return nil, err
		}
		wishes = append(wishes, &w)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return wishes, nil
}
