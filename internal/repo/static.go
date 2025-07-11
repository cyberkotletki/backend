package repo

import (
	"backend/internal/entity"
	"context"
	"errors"
	"io"
)

var (
	ErrStaticFileNotFound  = errors.New("static file not found")
	ErrFileStorageNotFound = errors.New("file not found in storage")
	ErrFileStorageUpload   = errors.New("file upload error")
	ErrFileStorageDelete   = errors.New("file delete error")
)

type StaticFileRepository interface {
	Upload(ctx context.Context, file *entity.StaticFile) (string, error)
	GetByID(ctx context.Context, id string) (*entity.StaticFile, error)
}

// FileStorage описывает абстракцию для работы с файловым хранилищем (S3, MinIO, локальное и т.д.)
// filePath — путь/ключ в хранилище, data — содержимое файла
// contentType — MIME-тип файла (например, image/png)
type FileStorage interface {
	Upload(ctx context.Context, filePath string, data io.ReadSeeker, contentType string) (string, error)
	Get(ctx context.Context, filePath string) (io.ReadSeeker, string, error) // возвращает поток и content-type
	Delete(ctx context.Context, filePath string) error
}
