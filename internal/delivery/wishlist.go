package delivery

import (
	"backend/internal/entity"
	"backend/internal/usecase"
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
func (h *WishlistHandler) Configure(e *echo.Echo, jwtMiddleware echo.MiddlewareFunc) {
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
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, entity.GetWishesResponse{Wishes: wishes})
}
