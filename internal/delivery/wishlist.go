package delivery

import (
	"backend/internal/entity"
	"backend/internal/usecase"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
)

type WishlistHandler struct {
	WishUC usecase.WishUsecase
}

func NewWishlistHandler(wishUC usecase.WishUsecase) *WishlistHandler {
	return &WishlistHandler{WishUC: wishUC}
}

// Configure настраивает роуты wishlist
func (h *WishlistHandler) Configure(e *echo.Group, jwtMiddleware echo.MiddlewareFunc) {
	g := e.Group("/wishlist")
	g.POST("", h.AddWish, jwtMiddleware)
	g.PUT("", h.UpdateWish, jwtMiddleware)
	g.GET("", h.GetWishes)
}

func (h *WishlistHandler) AddWish(c echo.Context) error {
	var req entity.AddWishRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	uuid := c.Get("user_uuid").(string)
	req.UserUUID = uuid
	wishUUID, err := h.WishUC.AddWish(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidWish):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid wish")
		case errors.Is(err, usecase.ErrUserNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		case errors.Is(err, usecase.ErrStaticFileNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "static file not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	return c.JSON(http.StatusOK, entity.AddWishResponse{WishUUID: wishUUID})
}

func (h *WishlistHandler) UpdateWish(c echo.Context) error {
	var req entity.UpdateWishRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	uuid := c.Get("user_uuid").(string)
	req.UserUUID = uuid
	if err := h.WishUC.UpdateWish(c.Request().Context(), req); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidWish):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid wish")
		case errors.Is(err, usecase.ErrWishNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "wish not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	return c.NoContent(http.StatusOK)
}

func (h *WishlistHandler) GetWishes(c echo.Context) error {
	streamerUUID := c.QueryParam("streamer_uuid")
	if streamerUUID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing streamer_uuid")
	}
	wishes, err := h.WishUC.GetWishes(c.Request().Context(), streamerUUID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrWishNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "wishes not found")
		case errors.Is(err, usecase.ErrInvalidWish):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid wish")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"wishes": wishes})
}
