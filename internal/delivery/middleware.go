package delivery

import (
	"backend/pkg/jwt"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

// NewJWTMiddleware возвращает echo middleware для проверки JWT и установки user_uuid в context
func NewJWTMiddleware(jwtService *jwt.JWT) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}
			// Ожидаем формат "Bearer <token>"
			var jwtToken string
			if strings.HasPrefix(token, "Bearer ") {
				jwtToken = token[7:]
			} else {
				jwtToken = token
			}
			uuid, err := jwtService.ParseToken(jwtToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set("user_uuid", uuid)
			return next(c)
		}
	}
}
