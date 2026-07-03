package server

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"idmagic/internal/shared/version"
)

func (d Deps) handleVersion(c *echo.Context) error {
	info := version.Get()
	return c.JSON(http.StatusOK, info)
}
