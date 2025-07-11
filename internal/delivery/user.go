package delivery

import (
	"backend/internal/entity"
	"backend/internal/usecase"
	"backend/pkg/jwt"
	telegramauth "backend/pkg/telegram-auth"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type UserHandler struct {
	UserUC     usecase.UserUsecase
	JWTService *jwt.JWT
	BotToken   string // для проверки Telegram Mini App
}

func NewUserHandler(userUC usecase.UserUsecase, jwtService *jwt.JWT, botToken string) *UserHandler {
	return &UserHandler{UserUC: userUC, JWTService: jwtService, BotToken: botToken}
}

// Configure настраивает роуты user
func (h *UserHandler) Configure(e *echo.Group, jwtMiddleware echo.MiddlewareFunc) {
	g := e.Group("/user")
	g.POST("/streamer/register", h.Register)
	g.POST("/streamer/login", h.Login)
	g.GET("/me", h.Me, jwtMiddleware)
	g.PUT("", h.UpdateProfile, jwtMiddleware)
	g.GET("", h.GetProfile)
	g.GET("/history", h.GetHistory, jwtMiddleware)
}

func (h *UserHandler) Register(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	user, err := telegramauth.VerifyUser(authHeader, h.BotToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid telegram auth: "+err.Error())
	}
	var req entity.RegisterUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// Сохраняем telegram_id
	req.TelegramID = strconv.FormatInt(user.ID, 10)
	streamerUUID, err := h.UserUC.Register(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidRegisterRequest):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid register request")
		case errors.Is(err, usecase.ErrUserAlreadyExists):
			return echo.NewHTTPError(http.StatusConflict, "user already exists")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	// Получаем пользователя по telegram_id для генерации токена
	dbUser, err := h.UserUC.GetByTelegramID(c.Request().Context(), req.TelegramID)
	if err != nil || dbUser == nil {
		c.Logger().Error("user not found after register")
		return echo.NewHTTPError(http.StatusInternalServerError, "user not found after register")
	}
	token, err := h.JWTService.GenerateToken(dbUser.UUID, 30*24*60*60) // 30 дней
	if err != nil {
		c.Logger().Error("failed to generate JWT token:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}
	cookie := new(http.Cookie)
	cookie.Name = "jwt"
	cookie.Value = token
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.MaxAge = 30 * 24 * 60 * 60
	c.SetCookie(cookie)
	return c.JSON(http.StatusOK, entity.RegisterUserResponse{StreamerUUID: streamerUUID})
}

func (h *UserHandler) UpdateProfile(c echo.Context) error {
	uuid := c.Get("user_uuid").(string)
	var req entity.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	req.UUID = uuid
	if err := h.UserUC.UpdateProfile(c.Request().Context(), req); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidUpdateRequest):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid update request")
		case errors.Is(err, usecase.ErrUserNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *UserHandler) GetProfile(c echo.Context) error {
	uuid := c.QueryParam("streamer_uuid")
	if uuid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing streamer_uuid")
	}
	profile, err := h.UserUC.GetProfile(c.Request().Context(), uuid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profile)
}

func (h *UserHandler) GetHistory(c echo.Context) error {
	uuid := c.Get("user_uuid").(string)
	pageStr := c.QueryParam("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	const pageSize = 20
	history, err := h.UserUC.GetHistory(c.Request().Context(), uuid, page, pageSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, history)
}

func (h *UserHandler) Login(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	user, err := telegramauth.VerifyUser(authHeader, h.BotToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid telegram auth: "+err.Error())
	}
	telegramID := strconv.FormatInt(user.ID, 10)
	dbUser, err := h.UserUC.GetByTelegramID(c.Request().Context(), telegramID)
	if err != nil || dbUser == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	token, err := h.JWTService.GenerateToken(dbUser.UUID, 30*24*60*60) // 30 дней
	if err != nil {
		c.Logger().Error("failed to generate JWT token:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}
	cookie := new(http.Cookie)
	cookie.Name = "jwt"
	cookie.Value = token
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.MaxAge = 30 * 24 * 60 * 60
	c.SetCookie(cookie)
	return c.NoContent(http.StatusNoContent)
}

func (h *UserHandler) Me(c echo.Context) error {
	uuid := c.Get("user_uuid").(string)
	return c.JSON(http.StatusOK, map[string]string{"streamer_uuid": uuid})
}
