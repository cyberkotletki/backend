package telegram_auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// User представляет данные пользователя Telegram
type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	IsPremium    bool   `json:"is_premium,omitempty"`
	PhotoURL     string `json:"photo_url,omitempty"`
}

// VerifyUser верифицирует пользователя Telegram по заголовку авторизации
func VerifyUser(authHeader, botToken string) (*User, error) {
	// Проверяем формат заголовка
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "tma" {
		return nil, fmt.Errorf("invalid authorization header format, expected 'tma <initData>'")
	}

	initDataRaw := parts[1]
	if initDataRaw == "" {
		return nil, fmt.Errorf("empty init data")
	}

	// Парсим init data
	values, err := url.ParseQuery(initDataRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse init data: %v", err)
	}

	// Извлекаем hash для проверки подписи
	hash := values.Get("hash")
	if hash == "" {
		return nil, fmt.Errorf("hash is missing in init data")
	}

	// Удаляем hash из данных для создания строки проверки
	values.Del("hash")

	// Создаем строку данных для проверки подписи
	dataCheckString := createDataCheckString(values)

	// Проверяем подпись
	if !verifySignature(dataCheckString, hash, botToken) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Проверяем время создания данных (не старше 24 часов)
	if err := validateAuthDate(values.Get("auth_date")); err != nil {
		return nil, err
	}

	// Извлекаем и парсим данные пользователя
	user, err := parseUserData(values.Get("user"))
	if err != nil {
		return nil, err
	}

	return user, nil
}

// createDataCheckString создает строку для проверки подписи
func createDataCheckString(values url.Values) string {
	var pairs []string
	for key, valueList := range values {
		if len(valueList) > 0 {
			pairs = append(pairs, key+"="+valueList[0])
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\n")
}

// verifySignature проверяет HMAC подпись
func verifySignature(dataCheckString, hash, botToken string) bool {
	// Создаем секретный ключ из токена бота
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Вычисляем HMAC от строки данных
	mac := hmac.New(sha256.New, secretKey.Sum(nil))
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	// Сравниваем подписи безопасным способом
	return hmac.Equal([]byte(hash), []byte(expectedHash))
}

// validateAuthDate проверяет время создания данных
func validateAuthDate(authDateStr string) error {
	if authDateStr == "" {
		return fmt.Errorf("auth_date is missing")
	}

	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid auth_date format: %v", err)
	}

	// Проверяем, что данные не старше 24 часов
	maxAge := int64(86400) // 24 часа в секундах
	if time.Now().Unix()-authDate > maxAge {
		return fmt.Errorf("init data is too old (max age: 24 hours)")
	}

	return nil
}

// parseUserData парсит JSON данные пользователя
func parseUserData(userDataStr string) (*User, error) {
	if userDataStr == "" {
		return nil, fmt.Errorf("user data is missing")
	}

	var user User
	if err := json.Unmarshal([]byte(userDataStr), &user); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %v", err)
	}

	// Проверяем обязательные поля
	if user.ID == 0 {
		return nil, fmt.Errorf("user ID is missing or invalid")
	}

	if user.FirstName == "" {
		return nil, fmt.Errorf("user first name is missing")
	}

	return &user, nil
}
