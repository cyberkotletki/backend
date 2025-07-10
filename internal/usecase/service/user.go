package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
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
	// Валидация входных данных
	if err := s.validateRegisterRequest(req); err != nil {
		return "", err
	}

	// Проверяем, не зарегистрирован ли уже пользователь с таким Telegram ID
	existingUser, err := s.userRepo.GetByTelegramID(ctx, req.TelegramID)
	if err != nil {
		return "", fmt.Errorf("ошибка проверки существующего пользователя: %w", err)
	}
	if existingUser != nil {
		return "", fmt.Errorf("пользователь с таким Telegram ID уже зарегистрирован")
	}

	// Создаем нового пользователя
	user := &entity.User{
		UUID:                  uuid.New().String(),
		PolygonWallet:         req.PolygonWallet,
		Name:                  req.Name,
		Topics:                req.Topics,
		Banner:                "",
		Avatar:                "",
		BackgroundColor:       stringPtr("#090909"), // значение по умолчанию
		BackgroundImage:       nil,
		ButtonBackgroundColor: "#7272FD", // значение по умолчанию
		ButtonTextColor:       "#FFFFFF", // значение по умолчанию
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		TelegramID:            req.TelegramID,
	}

	userUUID, err := s.userRepo.Register(ctx, user)
	if err != nil {
		return "", fmt.Errorf("ошибка регистрации пользователя: %w", err)
	}

	return userUUID, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, req entity.UpdateUserRequest) error {
	// Валидация входных данных
	if err := s.validateUpdateRequest(req); err != nil {
		return err
	}

	// Получаем существующего пользователя
	user, err := s.userRepo.GetByUUID(ctx, req.UUID)
	if err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}

	// Проверяем валидность файлов и принадлежность пользователю
	if err := s.validateUserFiles(ctx, req, user.UUID); err != nil {
		return err
	}

	// Обновляем поля пользователя
	user.Banner = req.Banner
	user.Name = req.Name
	user.BackgroundColor = req.BackgroundColor
	user.BackgroundImage = req.BackgroundImage
	user.ButtonBackgroundColor = req.ButtonBackgroundColor
	user.ButtonTextColor = req.ButtonTextColor
	user.Avatar = req.Avatar
	user.UpdatedAt = time.Now()

	// Сохраняем изменения
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("ошибка обновления профиля: %w", err)
	}

	return nil
}

func (s *UserService) GetProfile(ctx context.Context, uuid string) (*entity.UserProfileResponse, error) {
	if uuid == "" {
		return nil, fmt.Errorf("UUID пользователя не может быть пустым")
	}

	user, err := s.userRepo.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %w", err)
	}

	response := &entity.UserProfileResponse{
		Banner:                s.buildImageURL(user.Banner),
		Name:                  user.Name,
		BackgroundColor:       user.BackgroundColor,
		ButtonBackgroundColor: user.ButtonBackgroundColor,
		ButtonTextColor:       user.ButtonTextColor,
		Avatar:                s.buildImageURL(user.Avatar),
		Topics:                user.Topics,
	}

	// Обрабатываем BackgroundImage если есть
	if user.BackgroundImage != nil && *user.BackgroundImage != "" {
		backgroundImageURL := s.buildImageURL(*user.BackgroundImage)
		response.BackgroundImage = &backgroundImageURL
	}

	return response, nil
}

func (s *UserService) GetHistory(ctx context.Context, uuid string, page int, pageSize int) (*entity.UserHistoryResponse, error) {
	if uuid == "" {
		return nil, fmt.Errorf("UUID пользователя не может быть пустым")
	}

	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // значение по умолчанию
	}

	// Проверяем, что пользователь существует
	_, err := s.userRepo.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %w", err)
	}

	// Получаем историю из репозитория
	historyItems, err := s.historyRepo.GetByStreamerUUID(ctx, uuid, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории: %w", err)
	}

	// Конвертируем в response формат
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
	return s.userRepo.GetByTelegramID(ctx, telegramID)
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
	if err != nil {
		return fmt.Errorf("файл не найден")
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
