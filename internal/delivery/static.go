package delivery

import (
	"backend/internal/entity"
	"backend/internal/usecase"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"strconv"
)

type StaticHandler struct {
	StaticUC usecase.StaticUsecase
}

func NewStaticHandler(staticUC usecase.StaticUsecase) *StaticHandler {
	return &StaticHandler{StaticUC: staticUC}
}

func (h *StaticHandler) Configure(e *echo.Group, jwtMiddleware echo.MiddlewareFunc) {
	g := e.Group("/static")
	g.POST("/upload", h.Upload, jwtMiddleware)
	g.GET(":id", h.GetFile)
}

func (h *StaticHandler) Upload(c echo.Context) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot open file")
	}
	defer func() { _ = file.Close() }()
	// echo.File implements io.ReadSeeker если underlying reader поддерживает
	readSeeker, ok := file.(io.ReadSeeker)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "file must support seeking")
	}
	typeStr := c.FormValue("type")
	if typeStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}
	uploaderUUID := c.Get("user_uuid")
	var uploader string
	if uploaderUUID != nil {
		uploader = uploaderUUID.(string)
	}
	id, err := h.StaticUC.Upload(c.Request().Context(), typeStr, readSeeker, fileHeader.Header.Get("Content-Type"), uploader)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrStaticInvalidType):
			return echo.NewHTTPError(http.StatusBadRequest, "invalid file type")
		case errors.Is(err, usecase.ErrStaticImageTooSmall):
			return echo.NewHTTPError(http.StatusBadRequest, "image too small")
		case errors.Is(err, usecase.ErrStaticFileTooLarge):
			return echo.NewHTTPError(http.StatusBadRequest, "file too large")
		case errors.Is(err, usecase.ErrStaticFileEmpty):
			return echo.NewHTTPError(http.StatusBadRequest, "file is empty")
		case errors.Is(err, usecase.ErrStaticFileUpload):
			return echo.NewHTTPError(http.StatusBadRequest, "file upload error")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	return c.JSON(http.StatusOK, entity.UploadStaticResponse{UUID: id})
}

func (h *StaticHandler) GetFile(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing file id")
	}
	file, err := h.StaticUC.GetFile(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrStaticFileNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}
	defer func() {
		if closer, ok := file.(io.Closer); ok {
			_ = closer.Close()
		}
	}()
	c.Response().Header().Set("Accept-Ranges", "bytes")
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot determine file size")
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot seek to start of file")
	}
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(size, 10))
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, file)
	return err
}
