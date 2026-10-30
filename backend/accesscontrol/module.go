// Package accesscontrol は資源へのアクセスと管理権限の認可を組み立てる
// ([[wi-53-rebac-fine-grained-authorization]])。
//
// 組み立て地点に公開するのは、永続層ごとの Module の生成と管理 API の経路の登録だけである。
// それ以外の実装は internal/ に置き、ほかのモジュールへは公開しない。
package accesscontrol

import (
	"github.com/ambi/idmagic/backend/accesscontrol/internal/db_memory"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/db_postgres"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/handlers_http"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/ports"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/principals_idmanagement"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/usecases"
	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/labstack/echo/v5"
)

// Module は AccessControl の管理 API と判定経路が必要とする依存を持つ。
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

// RouteDeps は管理 API の経路が AccessControl の外から受け取る依存である。
type RouteDeps struct {
	support.Deps
	*support.Authenticator
	// Agents と Users は、代行チェーン上のプリンシパルの有効性を判定する記録の正である。
	Agents agentports.AgentRepository
	Users  userports.UserRepository
	// Authorizer はローカルとリモートを差し替えられる AuthZEN ポート。
	Authorizer policyports.Authorizer
}

// AdminRolePolicyResponse は組み立て地点を通した HTTP テストで応答を読むための型である。
type AdminRolePolicyResponse = handlers_http.AdminRolePolicyResponse

// RelationCheckerDeps は RelationChecker が AccessControl の外から受け取る依存である。
type RelationCheckerDeps struct {
	Agents     agentports.AgentRepository
	Users      userports.UserRepository
	Authorizer policyports.Authorizer
	Emit       func(spec.DomainEvent)
}

// RelationChecker は、外部の資源の評価へ関係の判定を渡す入口を返す。
// 保存先を組み立てていない Module は nil を返し、呼び出し側が評価不能として扱う。
func (m Module) RelationChecker(d RelationCheckerDeps) policyports.RelationChecker {
	if m.TupleRepo == nil || m.ModelRepo == nil {
		return nil
	}
	return usecases.RelationChecker{Deps: usecases.Deps{
		Tuples: m.TupleRepo, Models: m.ModelRepo,
		Principals: principals_idmanagement.Resolver{Agents: d.Agents, Users: d.Users},
		Authorizer: d.Authorizer, Emit: d.Emit,
	}}
}

// RegisterRoutes は認可モデル、関係の判定と管理ロール一覧の経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator,
		TupleRepo: m.TupleRepo, ModelRepo: m.ModelRepo,
		Principals: principals_idmanagement.Resolver{Agents: d.Agents, Users: d.Users},
		Authorizer: d.Authorizer,
	})
}
