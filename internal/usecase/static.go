package usecase

import (
	"context"
	"io"
)

type StaticUsecase interface {
	Upload(ctx context.Context, fileType string, fileData io.ReadSeeker, contentType string, uploaderUUID string) (string, error)
	GetFile(ctx context.Context, id string) (io.ReadSeeker, error)
}
