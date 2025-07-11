package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"backend/internal/usecase"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Импортируем для поддержки декодирования PNG
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type StaticService struct {
	staticRepo  repo.StaticFileRepository
	fileStorage repo.FileStorage
}

func NewStaticService(staticRepo repo.StaticFileRepository, fileStorage repo.FileStorage) *StaticService {
	return &StaticService{
		staticRepo:  staticRepo,
		fileStorage: fileStorage,
	}
}

// Минимальные размеры для разных типов изображений (в пикселях)
var minSizes = map[string]struct{ width, height int }{
	"avatar":     {128, 128},
	"banner":     {800, 200},
	"background": {800, 800},
	"wish":       {128, 128},
}

const maxFileSize = 10 * 1024 * 1024 // 10 MB

func (s *StaticService) Upload(ctx context.Context, fileType string, fileData io.ReadSeeker, _ string, uploaderUUID string) (string, error) {
	if !isValidFileType(fileType) {
		return "", usecase.ErrStaticInvalidType
	}
	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		return "", usecase.ErrStaticFileUpload
	}
	if len(fileBytes) > maxFileSize {
		return "", usecase.ErrStaticFileTooLarge
	}
	if len(fileBytes) == 0 {
		return "", usecase.ErrStaticFileEmpty
	}
	actualContentType := http.DetectContentType(fileBytes)
	if !isSupportedImageType(actualContentType) {
		return "", usecase.ErrStaticInvalidType
	}
	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return "", usecase.ErrStaticFileUpload
	}
	if format != "jpeg" && format != "png" {
		return "", usecase.ErrStaticInvalidType
	}
	minSize, exists := minSizes[fileType]
	if !exists {
		return "", usecase.ErrStaticInvalidType
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < minSize.width || height < minSize.height {
		return "", usecase.ErrStaticImageTooSmall
	}
	processedImg := img
	if fileType == "avatar" || fileType == "wish" {
		processedImg = makeSquare(img)
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, processedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", usecase.ErrStaticFileUpload
	}
	fileID := uuid.New().String()
	filePath := fmt.Sprintf("%s/%s.jpg", fileType, fileID)
	_, err = s.fileStorage.Upload(ctx, filePath, bytes.NewReader(buf.Bytes()), "image/jpeg")
	if err != nil {
		return "", usecase.ErrStaticFileUpload
	}
	staticFile := &entity.StaticFile{
		ID:           fileID,
		Type:         fileType,
		UploaderUUID: uploaderUUID,
	}
	_, err = s.staticRepo.Upload(ctx, staticFile)
	if err != nil {
		_ = s.fileStorage.Delete(ctx, filePath)
		return "", usecase.ErrStaticFileUpload
	}
	return fileID, nil
}

func (s *StaticService) GetFile(ctx context.Context, id string) (io.ReadSeeker, error) {
	staticFile, err := s.staticRepo.GetByID(ctx, id)
	if err != nil {
		return nil, usecase.ErrStaticFileNotFound
	}
	filePath := fmt.Sprintf("%s/%s.jpg", staticFile.Type, staticFile.ID)
	file, _, err := s.fileStorage.Get(ctx, filePath)
	if err != nil {
		return nil, usecase.ErrStaticFileNotFound
	}
	return file, nil
}

// isValidFileType проверяет, является ли тип файла поддерживаемым
func isValidFileType(fileType string) bool {
	validTypes := []string{"avatar", "banner", "background", "wish"}
	for _, vt := range validTypes {
		if vt == fileType {
			return true
		}
	}
	return false
}

// isSupportedImageType проверяет, поддерживается ли тип изображения
func isSupportedImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/jpeg") ||
		strings.HasPrefix(contentType, "image/png")
}

// makeSquare обрезает изображение до квадратного формата с помощью встроенных функций Go
func makeSquare(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Определяем размер квадрата (меньшая из сторон)
	size := width
	if height < width {
		size = height
	}

	// Вычисляем координаты для обрезки (центрируем)
	x := (width - size) / 2
	y := (height - size) / 2

	// Создаем новое изображение нужного размера
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	// Копируем пиксели из исходного изображения
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			srcX := bounds.Min.X + x + dx
			srcY := bounds.Min.Y + y + dy
			dst.Set(dx, dy, img.At(srcX, srcY))
		}
	}

	return dst
}
