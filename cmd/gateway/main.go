package main

import (
	abiDescription "backend/internal/abi"
	"backend/internal/delivery"
	"backend/internal/repo/mongodb"
	redisrepo "backend/internal/repo/redis"
	"backend/internal/repo/s3"
	"backend/internal/usecase/service"
	"backend/pkg/jwt"
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	// Server
	ServerPort string

	// MongoDB
	MongoURL      string
	MongoDatabase string

	// MinIO (S3)
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// Polygon Blockchain
	PolygonRPCURL   string
	ChainID         int64
	ContractAddress string
	PrivateKey      string
	PollInterval    time.Duration

	// Static files
	StaticBaseURL string

	// Telegram Bot
	TelegramBotToken string

	// Redis
	RedisAddr     string
	RedisPassword string

	// CORS
	AllowedOrigins []string
}

func main() {
	ctx := context.Background()

	log.Println("🚀 Запуск Donly Gateway...")

	// Инициализация Consul
	consulClient, err := initConsul()
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к Consul: %v", err)
	}
	log.Println("✅ Consul подключен")

	// Инициализация Vault
	vaultClient, err := initVault()
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к Vault: %v", err)
	}
	log.Println("✅ Vault подключен")

	// Загрузка конфигурации из Vault
	config, err := loadConfigFromVault(vaultClient)
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
	}
	log.Println("✅ Конфигурация загружена из Vault")

	// Подключение к MongoDB
	mongoClient, err := initMongoDB(ctx, config.MongoURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("⚠️ Ошибка отключения от MongoDB: %v", err)
		}
	}()
	log.Println("✅ MongoDB подключена")

	// Подключение к MinIO
	_, err = initMinIO(config)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к MinIO: %v", err)
	}
	log.Println("✅ MinIO подключен")

	// Подключение к блокчейну
	polygonClient, contractABI, contractAddr, err := initPolygon(config)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к Polygon: %v", err)
	}
	log.Printf("✅ Polygon подключен к сети Chain UUID: %d", config.ChainID)
	log.Printf("📋 Контракт: %s", contractAddr.Hex())

	// Инициализация репозиториев
	db := mongoClient.Database(config.MongoDatabase)

	userRepo := mongodb.NewUserRepository(db)
	wishRepo := mongodb.NewWishRepository(db)
	historyRepo := mongodb.NewHistoryRepository(db)
	blockchainRepo := mongodb.NewBlockchainRepository(db)
	minioConfig := s3.Config{
		Endpoint:        config.MinIOEndpoint,
		AccessKeyID:     config.MinIOAccessKey,
		SecretAccessKey: config.MinIOSecretKey,
		BucketName:      config.MinIOBucket,
	}
	fileStorage, err := s3.NewFileStorage(minioConfig)
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации S3 репозитория: %v", err)
	}
	staticRepo := mongodb.NewStaticFileRepository(db)

	// --- Redis для Donation Events ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       0,
	})
	donationEventRepo := redisrepo.NewDonationEventRepo(redisClient, "donation_events")
	donationEventUC := service.NewDonationEventUsecase(donationEventRepo)
	donationEventHandler := delivery.NewDonationEventSSEHandler(donationEventUC)

	log.Println("✅ Репозитории инициализированы")

	// Инициализация JWT сервиса
	jwtService := jwt.New("mega-secret-key") // TODO: взять из конфигурации

	// Инициализация сервисов (usecase слой)
	userService := service.NewUserService(userRepo, historyRepo, staticRepo, config.StaticBaseURL)
	wishService := service.NewWishService(wishRepo, staticRepo, userRepo, blockchainRepo, config.StaticBaseURL, polygonClient, contractAddr, contractABI)
	staticService := service.NewStaticService(staticRepo, fileStorage)

	log.Println("✅ Сервисы инициализированы")

	// Инициализация handlers (delivery слой)
	userHandler := delivery.NewUserHandler(userService, jwtService, config.TelegramBotToken)
	wishHandler := delivery.NewWishlistHandler(wishService)
	staticHandler := delivery.NewStaticHandler(staticService)

	log.Println("✅ Handlers инициализированы")

	// Запуск мониторинга блокчейна
	go func() {
		if err := wishService.StartBlockchainMonitoring(ctx); err != nil {
			log.Printf("⚠️ Ошибка запуска мониторинга блокчейна: %v", err)
		}
	}()
	log.Println("🔍 Мониторинг блокчейна запущен")

	// Инициализация HTTP сервера
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     config.AllowedOrigins,
		AllowCredentials: true,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Requested-With"},
	}))

	// JWT middleware
	jwtMiddleware := delivery.NewJWTMiddleware(jwtService)

	// Группа /api
	api := e.Group("/api")

	// Регистрация маршрутов через методы Configure
	userHandler.Configure(api, jwtMiddleware)
	wishHandler.Configure(api, jwtMiddleware)
	staticHandler.Configure(api, jwtMiddleware)

	// Регистрация SSE endpoint для донатов
	donationEventHandler.Configure(api)

	// Health check для Consul
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "donly-gateway",
		})
	})

	// Регистрация сервиса в Consul
	if err := registerServiceInConsul(consulClient, config.ServerPort); err != nil {
		log.Printf("⚠️ Ошибка регистрации в Consul: %v", err)
	}

	// 13. Запуск сервера
	go func() {
		log.Printf("🌐 HTTP сервер запущен на порту %s", config.ServerPort)
		if err := e.Start(":" + config.ServerPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("❌ Ошибка запуска сервера: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Остановка сервера...")

	// Остановка мониторинга блокчейна
	wishService.StopBlockchainMonitoring()

	// Остановка HTTP сервера
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Ошибка остановки сервера: %v", err)
	}

	log.Println("👋 Сервер остановлен")
}

// initConsul инициализирует подключение к Consul
func initConsul() (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()

	// Для Docker Compose
	if consulAddr := os.Getenv("CONSUL_ADDR"); consulAddr != "" {
		config.Address = consulAddr
	} else {
		config.Address = "localhost:8500" // по умолчанию
	}

	return consulapi.NewClient(config)
}

// initVault инициализирует подключение к Vault
func initVault() (*vaultapi.Client, error) {
	config := vaultapi.DefaultConfig()

	// Для Docker Compose
	if vaultAddr := os.Getenv("VAULT_ADDR"); vaultAddr != "" {
		config.Address = vaultAddr
	} else {
		config.Address = "http://localhost:8200" // по умолчанию
	}

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Токен для разработки
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		client.SetToken(token)
	} else {
		client.SetToken("myroot") // токен из docker-compose
	}

	return client, nil
}

// loadConfigFromVault загружает конфигурацию из Vault
func loadConfigFromVault(client *vaultapi.Client) (*Config, error) {
	config := &Config{}

	// Пытаемся загрузить из Vault, если не получается - используем переменные окружения
	secret, err := client.Logical().Read("secret/data/donly")
	if err != nil || secret == nil {
		log.Println("⚠️ Конфигурация не найдена в Vault, используем переменные окружения")
		return loadConfigFromEnv(), nil
	}

	data := secret.Data["data"].(map[string]interface{})

	config.ServerPort = getStringFromVault(data, "server_port", "8080")
	config.MongoURL = getStringFromVault(data, "mongo_url", "mongodb://localhost:27017")
	config.MongoDatabase = getStringFromVault(data, "mongo_database", "donly")

	config.MinIOEndpoint = getStringFromVault(data, "minio_endpoint", "localhost:9000")
	config.MinIOAccessKey = getStringFromVault(data, "minio_access_key", "minioadmin")
	config.MinIOSecretKey = getStringFromVault(data, "minio_secret_key", "minioadmin")
	config.MinIOBucket = getStringFromVault(data, "minio_bucket", "donly-static")
	config.MinIOUseSSL = false

	// Polygon Amoy testnet конфигурация
	config.PolygonRPCURL = getStringFromVault(data, "polygon_rpc_url", "https://rpc-amoy.polygon.technology")
	config.ChainID = 80002 // Polygon Amoy testnet
	config.ContractAddress = getStringFromVault(data, "contract_address", "")
	config.PrivateKey = getStringFromVault(data, "private_key", "")

	pollInterval := getStringFromVault(data, "poll_interval", "15s")
	if duration, err := time.ParseDuration(pollInterval); err == nil {
		config.PollInterval = duration
	} else {
		config.PollInterval = 15 * time.Second
	}

	config.StaticBaseURL = getStringFromVault(data, "static_base_url", "http://localhost:8080")
	config.TelegramBotToken = getStringFromVault(data, "telegram_bot_token", "")

	// Redis
	config.RedisAddr = getStringFromVault(data, "redis_addr", "localhost:6379")
	config.RedisPassword = getStringFromVault(data, "redis_password", "")

	// CORS
	if origins, ok := data["allowed_origins"]; ok {
		if arr, ok := origins.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					config.AllowedOrigins = append(config.AllowedOrigins, s)
				}
			}
		}
	}
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}

	return config, nil
}

// loadConfigFromEnv загружает конфигурацию из переменных окружения (fallback)
func loadConfigFromEnv() *Config {
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		MongoURL:         getEnv("MONGO_URL", "mongodb://admin:password@localhost:27017"),
		MongoDatabase:    getEnv("MONGO_DATABASE", "donly"),
		MinIOEndpoint:    getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:   getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:   getEnv("MINIO_SECRET_KEY", "minioadmin123"),
		MinIOBucket:      getEnv("MINIO_BUCKET", "donly-static"),
		MinIOUseSSL:      false,
		PolygonRPCURL:    getEnv("POLYGON_RPC_URL", "https://rpc-amoy.polygon.technology"), // Polygon Amoy testnet
		ChainID:          80002,                                                            // Polygon Amoy testnet Chain UUID
		ContractAddress:  getEnv("CONTRACT_ADDRESS", ""),
		PrivateKey:       getEnv("PRIVATE_KEY", ""),
		PollInterval:     15 * time.Second,
		StaticBaseURL:    getEnv("STATIC_BASE_URL", "http://localhost:8080"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		AllowedOrigins:   []string{"*"},
	}
}

// initMongoDB инициализирует подключение к MongoDB
func initMongoDB(ctx context.Context, mongoURL string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(mongoURL)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Проверка подключения
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// initMinIO инициализирует подключение к MinIO
func initMinIO(config *Config) (*minio.Client, error) {
	client, err := minio.New(config.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinIOAccessKey, config.MinIOSecretKey, ""),
		Secure: config.MinIOUseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Создаем bucket если не существует
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, config.MinIOBucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(ctx, config.MinIOBucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		log.Printf("✅ Bucket '%s' создан", config.MinIOBucket)
	}

	return client, nil
}

// initPolygon инициализирует подключение к Polygon блокчейну
func initPolygon(config *Config) (*ethclient.Client, abi.ABI, common.Address, error) {
	// Подключение к Polygon RPC
	client, err := ethclient.Dial(config.PolygonRPCURL)
	if err != nil {
		return nil, abi.ABI{}, common.Address{}, fmt.Errorf("ошибка подключения к Polygon RPC: %w", err)
	}

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.ChainID(ctx)
	if err != nil {
		return nil, abi.ABI{}, common.Address{}, fmt.Errorf("ошибка получения Chain UUID: %w", err)
	}

	/*
		if chainID.Int64() != config.ChainID {
			return nil, abi.ABI{}, common.Address{}, fmt.Errorf("неожиданный Chain UUID: получен %d, ожидался %d", chainID.Int64(), config.ChainID)
		}
	*/

	// Загрузка ABI контракта
	contractABI, err := abi.JSON(strings.NewReader(abiDescription.DonatesABI))
	if err != nil {
		return nil, abi.ABI{}, common.Address{}, fmt.Errorf("ошибка парсинга ABI: %w", err)
	}

	// Адрес контракта
	if config.ContractAddress == "" {
		log.Println("⚠️ Адрес контракта не указан. Мониторинг блокчейна будет недоступен")
		return client, contractABI, common.Address{}, nil
	}

	contractAddr := common.HexToAddress(config.ContractAddress)

	// Проверка существования контракта
	code, err := client.CodeAt(ctx, contractAddr, nil)
	if err != nil {
		return nil, abi.ABI{}, common.Address{}, fmt.Errorf("ошибка проверки контракта: %w", err)
	}

	if len(code) == 0 {
		log.Printf("⚠️ Контракт по адресу %s не найден или не задеплоен", contractAddr.Hex())
	} else {
		log.Printf("✅ Контракт найден по адресу %s", contractAddr.Hex())
	}

	return client, contractABI, contractAddr, nil
}

// registerServiceInConsul регистрирует сервис в Consul
func registerServiceInConsul(client *consulapi.Client, port string) error {
	registration := &consulapi.AgentServiceRegistration{
		ID:      "donly-gateway",
		Name:    "donly-gateway",
		Port:    parsePort(port),
		Address: "localhost",
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://localhost:%s/health", port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	return client.Agent().ServiceRegister(registration)
}

// Вспомогательные функции

func getStringFromVault(data map[string]interface{}, key, defaultValue string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parsePort(port string) int {
	if port == "8080" {
		return 8080
	}
	return 8080
}
