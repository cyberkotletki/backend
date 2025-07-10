package delivery

import (
	"backend/pkg/jwt"
	"github.com/labstack/echo/v4"
	"net/http"
)

// NewJWTMiddleware возвращает echo middleware для проверки JWT и установки user_uuid в context
func NewJWTMiddleware(jwtService *jwt.JWT) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("jwt")
			if err != nil || cookie.Value == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}
			uuid, err := jwtService.ParseToken(cookie.Value)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set("user_uuid", uuid)
			return next(c)
		}
	}
}
