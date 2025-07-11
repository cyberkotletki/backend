package delivery

import (
	"backend/internal/usecase"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"net/http"
)

type DonationEventSSEHandler struct {
	UC usecase.DonationEventUsecase
}

func NewDonationEventSSEHandler(uc usecase.DonationEventUsecase) *DonationEventSSEHandler {
	return &DonationEventSSEHandler{UC: uc}
}

// Configure настраивает роуты donation event SSE
func (h *DonationEventSSEHandler) Configure(e *echo.Group) {
	g := e.Group("/donation-event")
	g.GET("/stream", h.Handle)
}

func (h *DonationEventSSEHandler) Handle(c echo.Context) error {
	streamerUUID := c.QueryParam("streamer_uuid")
	if streamerUUID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing streamer_uuid")
	}
	ctx := c.Request().Context()
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Flush()

	eventCh, errCh := h.UC.SubscribeDonationEvents(ctx, streamerUUID, "")
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}
			jsonData, _ := json.Marshal(event)
			_, _ = c.Response().Write([]byte("event: donation\ndata: "))
			_, _ = c.Response().Write(jsonData)
			_, _ = c.Response().Write([]byte("\n\n"))
			c.Response().Flush()
		case err, ok := <-errCh:
			if ok && err != nil {
				return err
			}
		}
	}
}
