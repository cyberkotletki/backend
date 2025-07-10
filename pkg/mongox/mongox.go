package mongox

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoConfig содержит параметры подключения
type MongoConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// MongoClient обертка для клиента и базы
type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoClient создает подключение и проверяет его
func NewMongoClient(cfg MongoConfig) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("mongo connect error: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping error: %w", err)
	}

	db := client.Database(cfg.Database)
	return &MongoClient{Client: client, Database: db}, nil
}

// EnsureCollection проверяет существование коллекции и создает её при необходимости
func (mc *MongoClient) EnsureCollection(ctx context.Context, name string, opts ...*options.CreateCollectionOptions) error {
	collections, err := mc.Database.ListCollectionNames(ctx, bson.M{"name": name})
	if err != nil {
		return fmt.Errorf("list collections error: %w", err)
	}
	for _, c := range collections {
		if c == name {
			return nil // уже существует
		}
	}
	return mc.Database.CreateCollection(ctx, name, opts...)
}
