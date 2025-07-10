package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
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
	// Проверяем поддерживаемый тип файла
	if !isValidFileType(fileType) {
		return "", fmt.Errorf("неподдерживаемый тип файла: %s. Поддерживаются: avatar, banner, background, wish", fileType)
	}

	// Читаем данные файла для проверки
	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Проверяем размер файла
	if len(fileBytes) > maxFileSize {
		return "", fmt.Errorf("размер файла превышает максимально допустимый (10 МБ)")
	}

	if len(fileBytes) == 0 {
		return "", fmt.Errorf("файл пустой")
	}

	// Определяем тип файла по содержимому
	actualContentType := http.DetectContentType(fileBytes)
	if !isSupportedImageType(actualContentType) {
		return "", fmt.Errorf("неподдерживаемый формат изображения. Поддерживаются только JPEG и PNG")
	}

	// Декодируем изображение
	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования изображения: %w", err)
	}

	// Проверяем формат изображения
	if format != "jpeg" && format != "png" {
		return "", fmt.Errorf("неподдерживаемый формат изображения: %s", format)
	}

	// Проверяем минимальные размеры
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	minSize, exists := minSizes[fileType]
	if !exists {
		return "", fmt.Errorf("неизвестный тип файла: %s", fileType)
	}

	if width < minSize.width || height < minSize.height {
		return "", fmt.Errorf("изображение типа %s должно быть минимум %d на %d пикселей",
			fileType, minSize.width, minSize.height)
	}

	// Обрабатываем изображение в зависимости от типа
	processedImg := img
	if fileType == "avatar" || fileType == "wish" {
		// Для аватара и желания делаем изображение квадратным
		processedImg = makeSquare(img)
	}

	// Конвертируем в JPEG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, processedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", fmt.Errorf("ошибка конвертации в JPEG: %w", err)
	}

	// Генерируем уникальный ID для файла
	fileID := uuid.New().String()
	filePath := fmt.Sprintf("%s/%s.jpg", fileType, fileID)

	// Сохраняем в файловое хранилище
	_, err = s.fileStorage.Upload(ctx, filePath, bytes.NewReader(buf.Bytes()), "image/jpeg")
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	// Сохраняем метаданные в базу данных
	staticFile := &entity.StaticFile{
		ID:           fileID,
		Type:         fileType,
		UploaderUUID: uploaderUUID,
	}

	_, err = s.staticRepo.Upload(ctx, staticFile)
	if err != nil {
		// Пытаемся удалить файл из хранилища в случае ошибки
		_ = s.fileStorage.Delete(ctx, filePath)
		return "", fmt.Errorf("ошибка сохранения м��таданных: %w", err)
	}

	return fileID, nil
}

func (s *StaticService) GetFile(ctx context.Context, id string) (io.ReadSeeker, error) {
	// Получаем метаданные файла
	staticFile, err := s.staticRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("файл не найден: %w", err)
	}

	// Формируем путь к файлу
	filePath := fmt.Sprintf("%s/%s.jpg", staticFile.Type, staticFile.ID)

	// Получаем файл из хранилища
	fileData, _, err := s.fileStorage.Get(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения файла из хранилища: %w", err)
	}

	return fileData, nil
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
