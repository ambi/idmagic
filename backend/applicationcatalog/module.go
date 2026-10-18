// Package applicationcatalog は Application と各プロトコルの管理とポータルを編成する。
package applicationcatalog

import (
	cataloghttp "github.com/ambi/idmagic/backend/applicationcatalog/internal/handlers_http"
	"github.com/labstack/echo/v5"
)

type Dependencies = cataloghttp.Deps

// Register は解決済みのテナントのグループへ管理とポータルの経路を登録する。
func Register(g *echo.Group, deps Dependencies) {
	cataloghttp.RegisterRoutes(g, deps)
}
