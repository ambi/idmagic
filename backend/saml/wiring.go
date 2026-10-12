package saml

import (
	"github.com/ambi/idmagic/backend/saml/internal/db_memory"
	"github.com/ambi/idmagic/backend/saml/internal/db_postgres"
	"github.com/ambi/idmagic/backend/saml/internal/handlers_http"
	"github.com/ambi/idmagic/backend/saml/ports"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ServiceProviderRepository は組み立て地点が SP と IdP profile を共有する保存先である。
type ServiceProviderRepository interface {
	ports.SamlServiceProviderRepository
	ports.SamlIdentityProviderProfileRepository
}

type (
	Sessions       = handlers_http.Sessions
	SessionRevoker = handlers_http.SessionRevoker
)

func NewSamlServiceProviderRepository() ServiceProviderRepository {
	return db_memory.NewSamlServiceProviderRepository()
}

func NewAuthnRequestReplayStore() ports.AuthnRequestReplayStore {
	return db_memory.NewAuthnRequestReplayStore()
}

func NewMemoryModule() Module {
	repo := NewSamlServiceProviderRepository()
	return Module{SPRepo: repo, ProfileRepo: repo, ReplayStore: NewAuthnRequestReplayStore()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{SPRepo: &db_postgres.SamlServiceProviderRepository{Pool: pool}, ProfileRepo: &db_postgres.SamlIdentityProviderProfileRepository{Pool: pool}, ReplayStore: &db_postgres.AuthnRequestReplayStore{Pool: pool}}
}
