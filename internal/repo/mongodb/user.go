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

type userRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) repo.UserRepository {
	col := db.Collection("users")
	// Создаём уникальные индексы для telegram_id и polygon_wallet
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "telegram_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{Key: "polygon_wallet", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	})
	return &userRepository{
		col: col,
	}
}

func (r *userRepository) Register(ctx context.Context, user *entity.User) (string, error) {
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	_, err := r.col.InsertOne(ctx, user)
	if err != nil {
		// Обработка ошибки уникальности
		if mongo.IsDuplicateKeyError(err) {
			return "", errors.Join(repo.ErrUserAlreadyExists, err)
		}
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
		return repo.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	filter := bson.M{"uuid": uuid}
	var user entity.User
	err := r.col.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repo.ErrUserNotFound
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
			return nil, repo.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
