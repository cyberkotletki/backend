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

type staticFileRepository struct {
	col *mongo.Collection
}

func NewStaticFileRepository(db *mongo.Database) repo.StaticFileRepository {
	return &staticFileRepository{
		col: db.Collection("static_files"),
	}
}

func (r *staticFileRepository) Upload(ctx context.Context, file *entity.StaticFile) (string, error) {
	file.CreatedAt = time.Now()
	res, err := r.col.InsertOne(ctx, file)
	if err != nil {
		return "", err
	}
	id := ""
	if oid, ok := res.InsertedID.(string); ok {
		id = oid
	}
	return id, nil
}

func (r *staticFileRepository) GetByID(ctx context.Context, id string) (*entity.StaticFile, error) {
	filter := bson.M{"_id": id}
	var file entity.StaticFile
	err := r.col.FindOne(ctx, filter).Decode(&file)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repo.ErrStaticFileNotFound
		}
		return nil, err
	}
	return &file, nil
}
