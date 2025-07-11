package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"backend/internal/usecase"
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo      repo.UserRepository
	historyRepo   repo.HistoryRepository
	staticRepo    repo.StaticFileRepository
	staticBaseURL string
}

func NewUserService(
	userRepo repo.UserRepository,
	historyRepo repo.HistoryRepository,
	staticRepo repo.StaticFileRepository,
	staticBaseURL string,
) *UserService {
	return &UserService{
		userRepo:      userRepo,
		historyRepo:   historyRepo,
		staticRepo:    staticRepo,
		staticBaseURL: staticBaseURL,
	}
}

func (s *UserService) Register(ctx context.Context, req entity.RegisterUserRequest) (string, error) {
	if err := s.validateRegisterRequest(req); err != nil {
		return "", errors.Join(usecase.ErrInvalidRegisterRequest, err)
	}
	user := &entity.User{
		UUID:                  uuid.New().String(),
		PolygonWallet:         req.PolygonWallet,
		Name:                  req.Name,
		Topics:                req.Topics,
		Banner:                "",
		Avatar:                "",
		BackgroundColor:       stringPtr("#090909"),
		BackgroundImage:       nil,
		ButtonBackgroundColor: "#7272FD",
		ButtonTextColor:       "#FFFFFF",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		TelegramID:            req.TelegramID,
	}
	userUUID, err := s.userRepo.Register(ctx, user)
	if err != nil {
		if errors.Is(err, repo.ErrUserAlreadyExists) {
			return "", usecase.ErrUserAlreadyExists
		}
		return "", err
	}
	return userUUID, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, req entity.UpdateUserRequest) error {
	if err := s.validateUpdateRequest(req); err != nil {
		return errors.Join(usecase.ErrInvalidUpdateRequest, err)
	}
	user, err := s.userRepo.GetByUUID(ctx, req.UUID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return usecase.ErrUserNotFound
		}
		return err
	}
	if err := s.validateUserFiles(ctx, req, user.UUID); err != nil {
		return errors.Join(usecase.ErrInvalidUpdateRequest, err)
	}
	user.Banner = req.Banner
	user.Name = req.Name
	user.BackgroundColor = req.BackgroundColor
	user.BackgroundImage = req.BackgroundImage
	user.ButtonBackgroundColor = req.ButtonBackgroundColor
	user.ButtonTextColor = req.ButtonTextColor
	user.Avatar = req.Avatar
	user.UpdatedAt = time.Now()
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return usecase.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *UserService) GetProfile(ctx context.Context, uuid string) (*entity.UserProfileResponse, error) {
	if uuid == "" {
		return nil, usecase.ErrUserNotFound
	}
	user, err := s.userRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	response := &entity.UserProfileResponse{
		Banner:                s.buildImageURL(user.Banner),
		Name:                  user.Name,
		BackgroundColor:       user.BackgroundColor,
		ButtonBackgroundColor: user.ButtonBackgroundColor,
		ButtonTextColor:       user.ButtonTextColor,
		Avatar:                s.buildImageURL(user.Avatar),
		Topics:                user.Topics,
		PolygonWallet:         user.PolygonWallet,
	}
	if user.BackgroundImage != nil && *user.BackgroundImage != "" {
		backgroundImageURL := s.buildImageURL(*user.BackgroundImage)
		response.BackgroundImage = &backgroundImageURL
	}
	return response, nil
}

func (s *UserService) GetHistory(ctx context.Context, uuid string, page int, pageSize int) (*entity.UserHistoryResponse, error) {
	if uuid == "" {
		return nil, usecase.ErrUserNotFound
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	_, err := s.userRepo.GetByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	historyItems, err := s.historyRepo.GetByStreamerUUID(ctx, uuid, page, pageSize)
	if err != nil {
		return nil, err
	}
	history := make([]entity.HistoryItem, 0, len(historyItems))
	for _, item := range historyItems {
		historyItem := entity.HistoryItem{
			Type:     item.Type,
			Username: item.Username,
			Datetime: item.Datetime.Format(time.RFC3339),
			Amount:   item.Amount,
			WishUUID: item.WishUUID,
			Message:  item.Message,
		}
		history = append(history, historyItem)
	}
	response := &entity.UserHistoryResponse{
		Page:    page,
		History: history,
	}
	return response, nil
}

func (s *UserService) GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error) {
	user, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// Вспомогательные методы

func (s *UserService) validateRegisterRequest(req entity.RegisterUserRequest) error {
	if req.TelegramID == "" {
		return fmt.Errorf("telegram ID не может быть пустым")
	}

	if req.Name == "" {
		return fmt.Errorf("имя не может быть пустым")
	}

	if len(req.Name) > 50 {
		return fmt.Errorf("имя не может быть длиннее 50 символов")
	}

	if req.PolygonWallet == "" {
		return fmt.Errorf("polygon кошелек не может быть пустым")
	}

	// Проверяем формат Ethereum адреса
	if !isValidPolygonAddress(req.PolygonWallet) {
		return fmt.Errorf("некорректный формат Polygon кошелька")
	}

	if len(req.Topics) == 0 {
		return fmt.Errorf("необходимо указать хотя бы одну тему")
	}

	if len(req.Topics) > 10 {
		return fmt.Errorf("нельзя указывать больше 10 тем")
	}

	// Проверяем валидность тем
	validTopics := map[string]bool{
		"IRL":      true,
		"Gaming":   true,
		"Music":    true,
		"ASMR":     true,
		"Creative": true,
		"Esports":  true,
		"18+":      true,
		"Animals":  true,
		"Other":    true,
	}

	for _, topic := range req.Topics {
		if !validTopics[topic] {
			return fmt.Errorf("недопустимая тема: %s", topic)
		}
	}

	return nil
}

func (s *UserService) validateUpdateRequest(req entity.UpdateUserRequest) error {
	if req.UUID == "" {
		return fmt.Errorf("UUID пользователя не может быть пустым")
	}

	if req.Name == "" {
		return fmt.Errorf("имя не может быть пустым")
	}

	if len(req.Name) > 50 {
		return fmt.Errorf("имя не может быть длиннее 50 символов")
	}

	// Проверяем валидность цветов
	if !isValidHexColor(req.ButtonBackgroundColor) {
		return fmt.Errorf("некорректный формат цвета фона кнопки")
	}

	if !isValidHexColor(req.ButtonTextColor) {
		return fmt.Errorf("некорректный формат цвета текста кнопки")
	}

	if req.BackgroundColor != nil && !isValidHexColor(*req.BackgroundColor) {
		return fmt.Errorf("некорректный формат цвета фона")
	}

	// Проверяем, что указан либо цвет фона, либо изображение фона
	if (req.BackgroundColor == nil || *req.BackgroundColor == "") &&
		(req.BackgroundImage == nil || *req.BackgroundImage == "") {
		return fmt.Errorf("необходимо указать либо цвет фона, либо изображение фона")
	}

	return nil
}

func (s *UserService) validateUserFiles(ctx context.Context, req entity.UpdateUserRequest, userUUID string) error {
	// Проверяем banner
	if req.Banner != "" {
		if err := s.validateStaticFile(ctx, req.Banner, "banner", userUUID); err != nil {
			return fmt.Errorf("ошибка валидации баннера: %w", err)
		}
	}

	// Проверяем avatar
	if req.Avatar != "" {
		if err := s.validateStaticFile(ctx, req.Avatar, "avatar", userUUID); err != nil {
			return fmt.Errorf("ошибка валидации аватара: %w", err)
		}
	}

	// Проверяем background image, если указан
	if req.BackgroundImage != nil && *req.BackgroundImage != "" {
		if err := s.validateStaticFile(ctx, *req.BackgroundImage, "background", userUUID); err != nil {
			return fmt.Errorf("ошибка валидации изображения фона: %w", err)
		}
	}

	return nil
}

func (s *UserService) validateStaticFile(ctx context.Context, fileID, expectedType, userUUID string) error {
	staticFile, err := s.staticRepo.GetByID(ctx, fileID)
	if errors.Is(err, repo.ErrStaticFileNotFound) {
		return fmt.Errorf("файл с ID '%s' не найден", fileID)
	}

	if staticFile.Type != expectedType {
		return fmt.Errorf("файл должен быть типа '%s'", expectedType)
	}

	if staticFile.UploaderUUID != userUUID {
		return fmt.Errorf("файл не принадлежит текущему пользователю")
	}

	return nil
}

func (s *UserService) buildImageURL(imageID string) string {
	if imageID == "" {
		return ""
	}
	return fmt.Sprintf("%s/static/%s", s.staticBaseURL, imageID)
}

// Вспомогательные функции для валидации

func isValidPolygonAddress(address string) bool {
	// Простая проверка формата Polygon адреса (совместим с Ethereum)
	re := regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	return re.MatchString(address)
}

func isValidHexColor(color string) bool {
	// Проверка формата HEX цвета
	re := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	return re.MatchString(color)
}

func stringPtr(s string) *string {
	return &s
}
