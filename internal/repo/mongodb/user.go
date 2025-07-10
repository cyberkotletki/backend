package mongodb

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type userRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) repo.UserRepository {
	return &userRepository{
		col: db.Collection("users"),
	}
}

func (r *userRepository) Register(ctx context.Context, user *entity.User) (string, error) {
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	_, err := r.col.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}
	return user.UUID, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()
	filter := bson.M{"uuid": user.UUID}
	update := bson.M{"$set": user}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *userRepository) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	filter := bson.M{"uuid": uuid}
	var user entity.User
	err := r.col.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error) {
	filter := bson.M{"telegram_id": telegramID}
	var user entity.User
	err := r.col.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
