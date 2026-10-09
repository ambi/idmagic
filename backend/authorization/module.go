// Package authorization は Authorization モジュールを組み立てる
// ([[wi-53-rebac-fine-grained-authorization]])。
//
// 組み立て地点に公開するのは、永続層ごとの Module の生成と管理 API の経路の登録だけである。
// それ以外の実装は internal/ に置き、ほかのモジュールへは公開しない。
package authorization

import (
	"github.com/ambi/idmagic/backend/authorization/internal/db_memory"
	"github.com/ambi/idmagic/backend/authorization/internal/db_postgres"
	"github.com/ambi/idmagic/backend/authorization/internal/handlers_http"
	"github.com/ambi/idmagic/backend/authorization/internal/ports"
	"github.com/ambi/idmagic/backend/authorization/internal/principals_idmanagement"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/labstack/echo/v5"
)

// Module は Authorization の管理 API と判定経路が必要とする依存を持つ。
// bootstrap が永続層 (memory / postgres) ごとに組み立てて渡す。
type Module struct {
	TupleRepo ports.RelationTupleRepository
	ModelRepo ports.AuthorizationModelRepository
}

// NewMemoryModule はプロセス内に保存する Module を作る。
// タプルとモデルの書き込み版は 1 つの Store が共有する。
func NewMemoryModule() Module {
	store := db_memory.NewStore()
	return Module{
		TupleRepo: db_memory.NewRelationTupleRepository(store),
		ModelRepo: db_memory.NewAuthorizationModelRepository(store),
	}
}

// NewPostgresModule は PostgreSQL に保存する Module を作る。
func NewPostgresModule(pool sharedpg.DB) Module {
	return Module{
		TupleRepo: &db_postgres.RelationTupleRepository{Pool: pool},
		ModelRepo: &db_postgres.AuthorizationModelRepository{Pool: pool},
	}
}

// RouteDeps は管理 API の経路が Authorization の外から受け取る依存である。
type RouteDeps struct {
	support.Deps
	*support.Authenticator
	// Agents と Users は、代行チェーン上のプリンシパルの有効性を判定する記録の正である。
	Agents agentports.AgentRepository
	Users  userports.UserRepository
	// Authorizer は OAuth2 が所有する AuthZEN ポート。
	Authorizer oauthports.Authorizer
}

// RegisterRoutes は /api/admin/v1/authorization/* の経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator,
		TupleRepo: m.TupleRepo, ModelRepo: m.ModelRepo,
		Principals: principals_idmanagement.Resolver{Agents: d.Agents, Users: d.Users},
		Authorizer: d.Authorizer,
	})
}
