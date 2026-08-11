package handlers_http_test

// SCL scenario "systemAdminがテナント横断でDEK健全性を一覧する"
// (spec/contexts/data-keys.yaml) を /api/admin/v1/data-keys/health 経由で検証する
// (wi-97 T007)。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeyshttp "github.com/ambi/idmagic/backend/datakeys/handlers_http"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// newDataKeyAdminServer bootstraps a DataEncryptionKey for the
// newSingleTenantRepo() tenant ("acme") so ListTenantDataKeyHealth's
// TenantRepo.FindAll has something to report.
func newDataKeyAdminServer(t *testing.T, actor *userdomain.User) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}
	}

	dataKeysRepo := datakeysmemory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	if _, err := datakeysusecases.BootstrapTenantDataKey(context.Background(), datakeysusecases.Deps{Repository: dataKeysRepo, Crypto: crypto}, "acme", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
			TenantRepo: newSingleTenantRepo(),
		},
		UserRepo:      userRepo,
		AuthnResolver: resolver,
		DataKeys:      datakeys.Module{Repository: dataKeysRepo, Crypto: crypto},
	})
	return e
}

func TestAdminDataKeysHealthListsBootstrappedTenants(t *testing.T) {
	sysAdmin := keyAdminUser("user_sys", tenancydomain.DefaultTenantID, []string{"system_admin"})
	e := newDataKeyAdminServer(t, sysAdmin)
	rec := getAdminKeys(e, "/realms/default/api/admin/v1/data-keys/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Tenants []datakeyshttp.TenantDataKeyHealthResponse `json:"tenants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tenants) != 1 {
		t.Fatalf("expected 1 tenant reported, got %+v", body.Tenants)
	}
	got := body.Tenants[0]
	if got.TenantID != "acme" || got.ActiveVersion != 1 || got.Provider != "tink_cleartext" || !got.ProviderReachable {
		t.Fatalf("unexpected health entry: %+v", got)
	}
}

func TestAdminDataKeysHealthRejectsPlainAdmin(t *testing.T) {
	admin := keyAdminUser("user_admin", tenancydomain.DefaultTenantID, []string{"admin"})
	e := newDataKeyAdminServer(t, admin)
	rec := getAdminKeys(e, "/realms/default/api/admin/v1/data-keys/health")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
