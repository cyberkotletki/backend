package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims структура для хранения uuid пользователя
type Claims struct {
	UUID string `json:"uuid"`
	jwt.RegisteredClaims
}

type JWT struct {
	secretKey []byte
}

func New(secret string) *JWT {
	return &JWT{secretKey: []byte(secret)}
}

// GenerateToken создает JWT-токен для пользователя
func (j *JWT) GenerateToken(uuid string, ttlSeconds int) (string, error) {
	ttl := time.Duration(ttlSeconds) * time.Second
	claims := Claims{
		UUID: uuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ParseToken валидирует токен и возвращает uuid пользователя
func (j *JWT) ParseToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}
	return claims.UUID, nil
}
