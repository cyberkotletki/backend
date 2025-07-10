package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	telegram "backend/pkg/telegram-auth"
)

func main() {
	// Создаем Echo инстанс
	e := echo.New()

	// Настраиваем CORS для разработки
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	// Логирование запросов
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	// Получаем путь к статическим файлам
	staticPath := getStaticPath()
	log.Printf("Serving static files from: %s", staticPath)

	// Статические файлы
	e.Static("/static", staticPath)

	e.GET("/index.js", func(c echo.Context) error {
		return c.File(filepath.Join(staticPath, "index.js"))
	})

	// Главная страница
	e.GET("/", func(c echo.Context) error {
		return c.File(filepath.Join(staticPath, "index.html"))
	})

	// API для авторизации с реальной проверкой
	e.POST("/api/auth/telegram", func(c echo.Context) error {
		var req struct {
			InitData string `json:"init_data"`
		}

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		log.Printf("Received init data: %s", req.InitData)

		// Проверяем подпись Telegram
		user, err := telegram.VerifyUser("tma "+req.InitData, botToken)
		if err != nil {
			log.Printf("Telegram validation failed: %v", err)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Telegram data"})
		}

		// Возвращаем данные пользователя (без JWT для простоты тестирования)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"token": "test-token-" + string(rune(user.ID)), // Простой токен для тестирования
			"user": map[string]interface{}{
				"id":         user.ID,
				"username":   user.Username,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"status":     "authenticated",
			},
		})
	})

	// API для тестирования (защищенный endpoint)
	e.GET("/api/profile", func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header required"})
		}

		// Простая проверка токена (для тестирования)
		if !strings.Contains(authHeader, "test-token-") {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id":    123456789,
			"username":   "testuser",
			"first_name": "Test",
			"last_name":  "User",
			"status":     "active",
			"balance":    "0.00 ETH",
			"created_at": time.Now().Format(time.RFC3339),
		})
	})

	// Healthcheck
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Запускаем сервер
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Starting mini-app test server on port %s", port)
	log.Fatal(e.Start(":" + port))
}

// getStaticPath возвращает путь к папке static
func getStaticPath() string {
	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get working directory:", err)
	}

	// Если запускаем из cmd/mini-app-test, поднимаемся на два уровня вверх
	if filepath.Base(wd) == "mini-app-test" {
		staticPath := filepath.Join(filepath.Dir(filepath.Dir(wd)), "static")
		if _, err := os.Stat(staticPath); err == nil {
			return staticPath
		}
	}

	// Если запускаем из корня проекта (go run cmd/mini-app-test/main.go)
	staticPath := filepath.Join(wd, "static")
	if _, err := os.Stat(staticPath); err == nil {
		return staticPath
	}

	// Пробуем найти папку static, поднимаясь по директориям
	currentDir := wd
	for i := 0; i < 5; i++ { // Ограничиваем поиск 5 уровнями
		staticPath := filepath.Join(currentDir, "static")
		if _, err := os.Stat(staticPath); err == nil {
			return staticPath
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Достигли корня файловой системы
		}
		currentDir = parentDir
	}

	log.Fatal("Static folder not found. Make sure you have 'static' folder in your project root")
	return ""
}
