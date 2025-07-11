package usecase

import (
	"context"
	"errors"
	"io"
)

var (
	ErrStaticInvalidType   = errors.New("invalid static file type")
	ErrStaticImageTooSmall = errors.New("image too small")
	ErrStaticFileNotFound  = errors.New("static file not found")
	ErrStaticFileUpload    = errors.New("static file upload error")
	ErrStaticFileTooLarge  = errors.New("file too large")
	ErrStaticFileEmpty     = errors.New("file is empty")
)

type StaticUsecase interface {
	Upload(ctx context.Context, fileType string, fileData io.ReadSeeker, contentType string, uploaderUUID string) (string, error)
	GetFile(ctx context.Context, id string) (io.ReadSeeker, error)
}
