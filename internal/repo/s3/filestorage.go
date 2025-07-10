package s3

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"time"
)

// FileStorage реализует интерфейс repo.FileStorage для MinIO/S3
type FileStorage struct {
	client     *minio.Client
	bucketName string
}

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

func NewFileStorage(cfg Config) (*FileStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket check error: %w", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("bucket create error: %w", err)
		}
	}
	return &FileStorage{client: client, bucketName: cfg.BucketName}, nil
}

func (s *FileStorage) Upload(ctx context.Context, filePath string, data io.ReadSeeker, contentType string) (string, error) {
	// Размер файла определяем через Seek
	cur, _ := data.Seek(0, io.SeekCurrent)
	sz, err := data.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}
	_, err = data.Seek(cur, io.SeekStart)
	if err != nil {
		return "", err
	}
	_, err = s.client.PutObject(ctx, s.bucketName, filePath, data, sz, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func (s *FileStorage) Get(ctx context.Context, filePath string) (io.ReadSeeker, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	stat, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, "", err
	}
	return obj, stat.ContentType, nil
}

func (s *FileStorage) Delete(ctx context.Context, filePath string) error {
	return s.client.RemoveObject(ctx, s.bucketName, filePath, minio.RemoveObjectOptions{})
}
