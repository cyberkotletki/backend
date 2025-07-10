package service

import (
	"backend/internal/entity"
	"backend/internal/repo"
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

type WishService struct {
	wishRepo       repo.WishRepository
	staticRepo     repo.StaticFileRepository
	userRepo       repo.UserRepository
	blockchainRepo repo.BlockchainRepository
	staticBaseURL  string

	// Blockchain monitoring
	client       *ethclient.Client
	contractAddr common.Address
	contractABI  abi.ABI
	stopChan     chan struct{}
	isRunning    bool
	pollInterval time.Duration
	lastBlock    uint64
}

// BlockchainEvent представляет событие блокчейна для сохранения в БД
type BlockchainEvent struct {
	BlockNumber uint64    `bson:"block_number" json:"block_number"`
	TxHash      string    `bson:"tx_hash" json:"tx_hash"`
	EventType   string    `bson:"event_type" json:"event_type"`
	ProcessedAt time.Time `bson:"processed_at" json:"processed_at"`
}

// WishAddedEvent событие добавления желания в контракт
type WishAddedEvent struct {
	UserUUID string
	WishUUID string
	Price    *big.Int
}

// WishCompletedEvent событие завершения желания
type WishCompletedEvent struct {
	UserUUID string
	WishUUID string
	Price    *big.Int
}

// WishDeletedEvent событие удаления желания
type WishDeletedEvent struct {
	UserUUID          string
	WishUUID          string
	AccumulatedAmount *big.Int
}

func NewWishService(
	wishRepo repo.WishRepository,
	staticRepo repo.StaticFileRepository,
	userRepo repo.UserRepository,
	blockchainRepo repo.BlockchainRepository,
	staticBaseURL string,
	polygonClient *ethclient.Client,
	contractAddr common.Address,
	contractABI abi.ABI,
) *WishService {
	return &WishService{
		wishRepo:       wishRepo,
		staticRepo:     staticRepo,
		userRepo:       userRepo,
		blockchainRepo: blockchainRepo,
		staticBaseURL:  staticBaseURL,
		client:         polygonClient,
		contractAddr:   contractAddr,
		contractABI:    contractABI,
		stopChan:       make(chan struct{}),
		pollInterval:   10 * time.Second, // опрос каждые 10 секунд
	}
}

// StartBlockchainMonitoring запускает фоновое отслеживание событий блокчейна
func (s *WishService) StartBlockchainMonitoring(ctx context.Context) error {
	if s.isRunning {
		return fmt.Errorf("мониторинг блокчейна уже запущен")
	}

	// Получаем последний обработанный блок из БД или текущий блок
	lastProcessedBlock, err := s.getLastProcessedBlock(ctx)
	if err != nil {
		log.Printf("Ошибка получения последнего обработанного блока: %v", err)
		// Если нет записи, начинаем с текущего блока
		header, err := s.client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("ошибка получения текущего блока: %w", err)
		}
		s.lastBlock = header.Number.Uint64()
	} else {
		s.lastBlock = lastProcessedBlock
	}

	s.isRunning = true
	log.Printf("Запускаем мониторинг событий смарт-контракта с блока %d", s.lastBlock)

	go s.blockchainEventLoop(ctx)
	return nil
}

// StopBlockchainMonitoring останавливает фоновое отслеживание
func (s *WishService) StopBlockchainMonitoring() {
	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
	log.Println("Мониторинг событий блокчейна остановлен")
}

func (s *WishService) AddWish(ctx context.Context, req entity.AddWishRequest) (string, error) {
	// Проверяем, что пользователь существует
	user, err := s.userRepo.GetByUUID(ctx, req.UserUUID)
	if err != nil {
		return "", fmt.Errorf("пользователь не найден: %w", err)
	}

	// Проверяем валидность данных
	if err := s.validateAddWishRequest(req); err != nil {
		return "", err
	}

	// Проверяем, что изображение существует и принадлежит пользователю
	staticFile, err := s.staticRepo.GetByID(ctx, req.Image)
	if err != nil {
		return "", fmt.Errorf("изображение не найдено: %w", err)
	}

	if staticFile.Type != "wish" {
		return "", fmt.Errorf("изображение должно быть типа 'wish'")
	}

	if staticFile.UploaderUUID != req.UserUUID {
		return "", fmt.Errorf("изображение не принадлежит текущему пользователю")
	}

	// Создаем новое желание в статусе pending
	wish := &entity.Wish{
		UUID:         uuid.New().String(),
		StreamerUUID: user.UUID,
		WishURL:      req.WishURL,
		Name:         req.Name,
		Description:  req.Description,
		Image:        req.Image,
		PolTarget:    req.PolTarget,
		PolAmount:    0.0,
		IsPriority:   req.IsPriority,
		Status:       "pending", // начальный статус - ожидание подтверждения в блокчейне
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	wishUUID, err := s.wishRepo.Add(ctx, wish)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения желания: %w", err)
	}

	return wishUUID, nil
}

func (s *WishService) UpdateWish(ctx context.Context, req entity.UpdateWishRequest) error {
	// Получаем существующее желание
	wish, err := s.wishRepo.GetByUUID(ctx, req.WishUUID)
	if err != nil {
		return fmt.Errorf("желание не найдено: %w", err)
	}

	// Проверяем, что желание принадлежит текущему пользователю
	user, err := s.userRepo.GetByUUID(ctx, req.UserUUID)
	if err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}

	if wish.StreamerUUID != user.UUID {
		return fmt.Errorf("желание не принадлежит текущему пользователю")
	}

	// Проверяем, что желание можно редактировать (не завершено и не удалено)
	if wish.Status == "complete" || wish.Status == "deleted" {
		return fmt.Errorf("нельзя редактировать завершенное или удаленное желание")
	}

	// Проверяем валидность нового изображения
	staticFile, err := s.staticRepo.GetByID(ctx, req.Image)
	if err != nil {
		return fmt.Errorf("изображение не найдено: %w", err)
	}

	if staticFile.Type != "wish" {
		return fmt.Errorf("изображение должно быть типа 'wish'")
	}

	if staticFile.UploaderUUID != req.UserUUID {
		return fmt.Errorf("изображение не принадлежит текущему пользователю")
	}

	// Обновляем поля
	wish.Image = req.Image
	wish.IsPriority = req.IsPriority
	wish.UpdatedAt = time.Now()

	// Сохраняем изменения
	err = s.wishRepo.Update(ctx, wish)
	if err != nil {
		return fmt.Errorf("ошибка обновления желания: %w", err)
	}

	return nil
}

func (s *WishService) GetWishes(ctx context.Context, streamerUUID string) ([]entity.WishResponse, error) {
	// Получаем все желания стримера
	wishes, err := s.wishRepo.GetByStreamerUUID(ctx, streamerUUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения желаний: %w", err)
	}

	// Конвертируем в response формат
	responses := make([]entity.WishResponse, 0, len(wishes))
	for _, wish := range wishes {
		// Показываем только активные желания
		if wish.Status != "active" {
			continue
		}

		response := entity.WishResponse{
			UUID:        wish.UUID,
			WishURL:     wish.WishURL,
			Name:        wish.Name,
			Description: wish.Description,
			Image:       s.buildImageURL(wish.Image),
			PolTarget:   wish.PolTarget,
			PolAmount:   wish.PolAmount,
			IsPriority:  wish.IsPriority,
		}
		responses = append(responses, response)
	}

	// Сортируем: сначала приоритетные, затем по дате создания (новые сначала)
	s.sortWishes(responses, wishes)

	return responses, nil
}

// validateAddWishRequest проверяет валидность запроса на добавление желания
func (s *WishService) validateAddWishRequest(req entity.AddWishRequest) error {
	if req.Name == "" {
		return fmt.Errorf("название желания не может быть пустым")
	}

	if len(req.Name) > 100 {
		return fmt.Errorf("название желания не может быть длиннее 100 символов")
	}

	if req.Description != nil && len(*req.Description) > 500 {
		return fmt.Errorf("описание желания не может быть длиннее 500 символов")
	}

	if req.PolTarget <= 0 {
		return fmt.Errorf("целевая сумма должна быть больше нуля")
	}

	if req.PolTarget > 1000000 {
		return fmt.Errorf("целевая сумма не может превышать 1,000,000 POL")
	}

	if req.Image == "" {
		return fmt.Errorf("изображение обязательно")
	}

	return nil
}

// buildImageURL создает полный URL для изображения
func (s *WishService) buildImageURL(imageID string) string {
	return fmt.Sprintf("%s/static/%s", s.staticBaseURL, imageID)
}

// sortWishes сортирует желания: приоритетные сначала, затем по дате создания
func (s *WishService) sortWishes(responses []entity.WishResponse, wishes []*entity.Wish) {
	// Создаем карту для быстрого поиска даты создания
	wishMap := make(map[string]time.Time)
	for _, wish := range wishes {
		if wish.Status == "active" {
			wishMap[wish.UUID] = wish.CreatedAt
		}
	}

	// Простая сортировка пузырьком (для небольшого количества элементов)
	for i := 0; i < len(responses)-1; i++ {
		for j := 0; j < len(responses)-i-1; j++ {
			// Сначала сравниваем по приоритету
			if !responses[j].IsPriority && responses[j+1].IsPriority {
				responses[j], responses[j+1] = responses[j+1], responses[j]
				continue
			}

			// Если приоритет одинаковый, сравниваем по дате (новые сначала)
			if responses[j].IsPriority == responses[j+1].IsPriority {
				date1 := wishMap[responses[j].UUID]
				date2 := wishMap[responses[j+1].UUID]
				if date1.Before(date2) {
					responses[j], responses[j+1] = responses[j+1], responses[j]
				}
			}
		}
	}
}

// blockchainEventLoop основной цикл отслеживания событий блокчейна
func (s *WishService) blockchainEventLoop(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Контекст отменен, останавливаем мониторинг блокчейна")
			return
		case <-s.stopChan:
			log.Println("Получен сигнал остановки мониторинга")
			return
		case <-ticker.C:
			if err := s.processNewBlocks(ctx); err != nil {
				log.Printf("Ошибка обработки новых блоков: %v", err)
			}
		}
	}
}

// processNewBlocks обрабатывает новые блоки и извлекает события
func (s *WishService) processNewBlocks(ctx context.Context) error {
	header, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка получения текущего блока: %w", err)
	}

	currentBlock := header.Number.Uint64()
	if currentBlock <= s.lastBlock {
		return nil // Нет новых блоков
	}

	// Если lastBlock == 0 или 1, считаем, что только что задеплоили контракт и все старые блоки не нужны
	if s.lastBlock == 0 || s.lastBlock == 1 {
		if err := s.saveLastProcessedBlock(ctx, currentBlock); err != nil {
			log.Printf("Ошибка сохранения последнего обработанного блока: %v", err)
		}
		s.lastBlock = currentBlock
		log.Printf("Пропускаем обработку старых блоков, выставляем lastBlock = %d", currentBlock)
		return nil
	}

	log.Printf("Обрабатываем блоки с %d по %d", s.lastBlock+1, currentBlock)

	const maxBlockRange = 50000
	fromBlock := s.lastBlock + 1
	toBlock := currentBlock

	for from := fromBlock; from <= toBlock; from += maxBlockRange {
		end := from + maxBlockRange - 1
		if end > toBlock {
			end = toBlock
		}
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(from)),
			ToBlock:   big.NewInt(int64(end)),
			Addresses: []common.Address{s.contractAddr},
		}
		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("ошибка получения логов: %w", err)
		}
		for _, vLog := range logs {
			if err := s.processBlockchainLog(ctx, vLog); err != nil {
				log.Printf("Ошибка обработки лога: %v", err)
			}
		}
		// После каждого чанка сохраняем последний обработанный блок
		if err := s.saveLastProcessedBlock(ctx, end); err != nil {
			log.Printf("Ошибка сохранения последнего обработанного блока: %v", err)
		}
		s.lastBlock = end
	}
	return nil
}

// processBlockchainLog обрабатывает отдельный лог события
func (s *WishService) processBlockchainLog(ctx context.Context, vLog types.Log) error {
	if len(vLog.Topics) == 0 {
		return nil
	}

	eventSignature := vLog.Topics[0].Hex()

	switch eventSignature {
	case s.getEventSignature("WishAdded"):
		return s.handleWishAdded(ctx, vLog)
	case s.getEventSignature("WishCompleted"):
		return s.handleWishCompleted(ctx, vLog)
	case s.getEventSignature("WishDeleted"):
		return s.handleWishDeleted(ctx, vLog)
	default:
		// Неизвестное событие, игнорируем
		return nil
	}
}

// handleWishAdded обрабатывает событие добавления желания в контракт
func (s *WishService) handleWishAdded(ctx context.Context, vLog types.Log) error {
	var event WishAddedEvent
	err := s.contractABI.UnpackIntoInterface(&event, "WishAdded", vLog.Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования WishAdded: %w", err)
	}

	// Извлекаем userUUID из индексированного топика
	if len(vLog.Topics) > 1 {
		event.UserUUID = strings.Trim(string(vLog.Topics[1][:]), "\x00")
	}

	log.Printf("Обнаружено новое желание в контракте: пользователь=%s, wishUUID=%s, цена=%s",
		event.UserUUID, event.WishUUID, event.Price.String())

	wish, err := s.wishRepo.GetByUUID(ctx, event.WishUUID)
	if err != nil {
		log.Printf("Не найдено желание с UUID %s: %v", event.WishUUID, err)
		return nil
	}

	// Проверяем, что желание в статусе pending
	if wish.Status != "pending" {
		log.Printf("Желание %s не в статусе pending (текущий статус: %s)", event.WishUUID, wish.Status)
		return nil
	}

	// Переводим в статус active
	wish.Status = "active"
	wish.UpdatedAt = time.Now()

	err = s.wishRepo.Update(ctx, wish)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса желания: %w", err)
	}

	log.Printf("Желание %s переведено в статус 'active'", wish.UUID)
	return nil
}

// handleWishCompleted обрабатывает событие завершения желания
func (s *WishService) handleWishCompleted(ctx context.Context, vLog types.Log) error {
	var event WishCompletedEvent
	err := s.contractABI.UnpackIntoInterface(&event, "WishCompleted", vLog.Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования WishCompleted: %w", err)
	}

	if len(vLog.Topics) > 1 {
		event.UserUUID = strings.Trim(string(vLog.Topics[1][:]), "\x00")
	}

	log.Printf("Желание завершено в контракте: пользователь=%s, wishUUID=%s",
		event.UserUUID, event.WishUUID)

	wish, err := s.wishRepo.GetByUUID(ctx, event.WishUUID)
	if err != nil {
		log.Printf("Не найдено желание с UUID %s: %v", event.WishUUID, err)
		return nil
	}

	// Проверяем, что желание в статусе active
	if wish.Status != "active" {
		log.Printf("Желание %s не в статусе active (текущий статус: %s)", event.WishUUID, wish.Status)
		return nil
	}

	wish.Status = "complete"
	wish.UpdatedAt = time.Now()

	err = s.wishRepo.Update(ctx, wish)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса желания на complete: %w", err)
	}

	log.Printf("Желание %s переведено в статус 'complete'", wish.UUID)
	return nil
}

// handleWishDeleted обрабатывает событие удаления желания
func (s *WishService) handleWishDeleted(ctx context.Context, vLog types.Log) error {
	var event WishDeletedEvent
	err := s.contractABI.UnpackIntoInterface(&event, "WishDeleted", vLog.Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования WishDeleted: %w", err)
	}

	if len(vLog.Topics) > 1 {
		event.UserUUID = strings.Trim(string(vLog.Topics[1][:]), "\x00")
	}

	log.Printf("Желание удалено в контракте: пользователь=%s, wishUUID=%s",
		event.UserUUID, event.WishUUID)

	// Эффективно находим желание напрямую по UUID
	wish, err := s.wishRepo.GetByUUID(ctx, event.WishUUID)
	if err != nil {
		log.Printf("Не найдено желание с UUID %s: %v", event.WishUUID, err)
		return nil
	}

	// Проверяем, что желание в статусе active
	if wish.Status != "active" {
		log.Printf("Желание %s не в статусе active (текущий статус: %s)", event.WishUUID, wish.Status)
		return nil
	}

	wish.Status = "deleted"
	wish.UpdatedAt = time.Now()

	err = s.wishRepo.Update(ctx, wish)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса желания на deleted: %w", err)
	}

	log.Printf("Желание %s переведено в статус 'deleted'", wish.UUID)
	return nil
}

// Вспомогательные методы для работы с блокчейном

// getEventSignature возвращает хеш сигнатуры события
func (s *WishService) getEventSignature(eventName string) string {
	event, exists := s.contractABI.Events[eventName]
	if !exists {
		return ""
	}
	return event.ID.Hex()
}

// comparePrices сравнивает цену в float64 с big.Int из контракта
func (s *WishService) comparePrices(floatPrice float64, bigIntPrice *big.Int) bool {
	// Конвертируем float64 в wei (умножаем на 10^18)
	weiFloat := floatPrice * 1e18
	weiBigInt := big.NewInt(int64(weiFloat))

	return weiBigInt.Cmp(bigIntPrice) == 0
}

// weiToFloat конвертирует wei в float64
func (s *WishService) weiToFloat(wei *big.Int) float64 {
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	polValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow(10, 18)))
	result, _ := polValue.Float64()
	return result
}

// getLastProcessedBlock получает последний обработанный блок из БД
func (s *WishService) getLastProcessedBlock(ctx context.Context) (uint64, error) {
	return s.blockchainRepo.GetLastProcessedBlock(ctx)
}

// saveLastProcessedBlock сохраняет последний обработанный блок в БД
func (s *WishService) saveLastProcessedBlock(ctx context.Context, blockNumber uint64) error {
	return s.blockchainRepo.SaveLastProcessedBlock(ctx, blockNumber)
}
