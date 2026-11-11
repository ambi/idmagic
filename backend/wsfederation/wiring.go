package wsfederation

import (
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/wsfederation/internal/db_memory"
	"github.com/ambi/idmagic/backend/wsfederation/internal/db_postgres"
	wsfedhttp "github.com/ambi/idmagic/backend/wsfederation/internal/handlers_http"
	"github.com/ambi/idmagic/backend/wsfederation/internal/realm_http"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

type (
	Sessions       = wsfedhttp.Sessions
	SessionRevoker = wsfedhttp.SessionRevoker
)

// DefaultEntraRealmDiscoveryURL は、Microsoft の商用クラウドのレルム探索である。
const DefaultEntraRealmDiscoveryURL = realm_http.DefaultURL

// NewEntraRealmDiscovery は、運用者が設定した URL から Microsoft のレルム探索を作る。通信はしない。
// https に限り、試験のために loopback の http だけを許す。
func NewEntraRealmDiscovery(rawURL string) (wsfedports.EntraRealmDiscovery, error) {
	discovery, err := realm_http.New(rawURL)
	if err != nil {
		return nil, err
	}
	return discovery, nil
}

// NewMemoryModule は WsFederation が所有する RP と Entra の観測記録の保存先をメモリで作る。
func NewMemoryModule() Module {
	return Module{RPRepo: db_memory.NewWsFedRelyingPartyRepository(), ObservationRepo: db_memory.NewEntraFederationObservationRepository()}
}

// NewPostgresModule は WsFederation が所有する RP と Entra の観測記録の保存先を PostgreSQL で作る。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		RPRepo:          &db_postgres.WsFedRelyingPartyRepository{Pool: pool},
		ObservationRepo: &db_postgres.EntraFederationObservationRepository{Pool: pool},
	}
}
