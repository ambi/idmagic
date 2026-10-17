package wsfederation

import (
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/wsfederation/internal/db_memory"
	"github.com/ambi/idmagic/backend/wsfederation/internal/db_postgres"
	wsfedhttp "github.com/ambi/idmagic/backend/wsfederation/internal/handlers_http"
)

type (
	Sessions       = wsfedhttp.Sessions
	SessionRevoker = wsfedhttp.SessionRevoker
)

// NewMemoryModule は WsFederation が所有する RP の保存先をメモリで作る。
func NewMemoryModule() Module {
	return Module{RPRepo: db_memory.NewWsFedRelyingPartyRepository()}
}

// NewPostgresModule は WsFederation が所有する RP の保存先を PostgreSQL で作る。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{RPRepo: &db_postgres.WsFedRelyingPartyRepository{Pool: pool}}
}
