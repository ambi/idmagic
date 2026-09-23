package handlers_http_test

// 署名鍵の健全性一覧の具体例を観測する。
//
// testing_stack は鍵ストアをメモリ実装に固定し、テナント保存先の読み出しを数えられない。
// KeyProvider の障害と、横断収集が走らないことはその 2 つでしか観測できないため、
// ここでは `Register` を直接組む。

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signinghttp "github.com/ambi/idmagic/backend/signingkeys/handlers_http"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	signingvault "github.com/ambi/idmagic/backend/signingkeys/keys_vault"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const tenantA = "acme"

// countedTenants はテナント横断の収集が始まったかを、全テナントの読み出し回数で観測する。
type countedTenants struct {
	*tenancymemory.TenantRepository
	findAll atomic.Int32
}

func (r *countedTenants) FindAll(ctx context.Context) ([]*tenancydomain.Tenant, error) {
	r.findAll.Add(1)
	return r.TenantRepository.FindAll(ctx)
}

type fixedResolver struct{ sub string }

func (r fixedResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return &authdomain.AuthenticationContext{UserID: r.sub, AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}, nil
}

type healthServer struct {
	echo    *echo.Echo
	tenants *countedTenants
}

// newHealthServer は actor が認証済みのスタックを組む。制御面テナントは default である。
func newHealthServer(t *testing.T, keyStore signingports.KeyStore, groups *groupmemory.GroupRepository, actor *userdomain.User) healthServer {
	t.Helper()
	tenants := &countedTenants{TenantRepository: tenancymemory.NewTenantRepository()}
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive},
		{ID: tenantA, Realm: tenantA, DisplayName: "Acme", Status: tenancydomain.TenantStatusActive},
	} {
		if err := tenants.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	users := usermemory.NewUserRepository()
	users.Seed(actor)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		TenantRepo:    tenants,
		IdManagement:  idmanagement.Module{UserRepo: users, GroupRepo: groups},
		SigningKeys:   signingkeys.Module{KeyStore: keyStore},
		AuthnResolver: fixedResolver{sub: actor.ID},
	})
	return healthServer{echo: e, tenants: tenants}
}

func (h healthServer) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	h.echo.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	return recorder
}

func healthUser(id, tenantID string, roles ...string) *userdomain.User {
	now := time.Now().UTC()
	return &userdomain.User{ID: id, TenantID: tenantID, PreferredUsername: id, Roles: roles, CreatedAt: now, UpdatedAt: now}
}

func tenantContext(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: id, Realm: id}, "", "")
}

// transitDouble は Vault Transit の代役である。到達不能にすると、Vault へ問い合わせる
// 操作はすべて失敗し、VaultKeyStore が取り込み済みの公開鍵だけが手元に残る。
type transitDouble struct {
	mu        sync.Mutex
	keys      map[string]*rsa.PrivateKey
	reachable bool
}

var errVaultUnreachable = errors.New("vault unreachable")

func (f *transitDouble) EnsureKey(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.reachable {
		return errVaultUnreachable
	}
	if f.keys[name] == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		f.keys[name] = key
	}
	return nil
}

func (f *transitDouble) RotateKey(context.Context, string) error {
	return errors.New("rotation is not exercised by the health examples")
}

func (f *transitDouble) LatestPublicKey(_ context.Context, name string) (string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.reachable {
		return "", 0, errVaultUnreachable
	}
	der, err := x509.MarshalPKIXPublicKey(&f.keys[name].PublicKey)
	if err != nil {
		return "", 0, err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), 1, nil
}

func (f *transitDouble) Sign(context.Context, string, int, []byte, crypto.SignerOpts) ([]byte, error) {
	return nil, errVaultUnreachable
}

func (f *transitDouble) Healthy(context.Context) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reachable
}

func (f *transitDouble) setReachable(reachable bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reachable = reachable
}

func decodeHealth(t *testing.T, recorder *httptest.ResponseRecorder) map[string]signinghttp.TenantKeyHealthResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Tenants []signinghttp.TenantKeyHealthResponse `json:"tenants"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	byTenant := map[string]signinghttp.TenantKeyHealthResponse{}
	for _, row := range body.Tenants {
		byTenant[row.TenantID] = row
	}
	return byTenant
}

//spec:covers EX-SIGNINGKEYS-008-01: tenant-a の Vault Transit が到達不能になると、制御面テナントの system_admin が制御面の経路で取得した健全性一覧は tenant-a の provider_healthy を false として返し、tenant-a の JWKS は取り込み済みの鍵を返し続ける。到達可能な間の一覧が true を返すことを対照にする。
func TestKeyProviderOutageIsVisibleInHealthWhileJWKSServesCachedKeys(t *testing.T) {
	transit := &transitDouble{keys: map[string]*rsa.PrivateKey{}, reachable: true}
	keyStore := signingvault.NewVaultKeyStore(transit, "idmagic-signing-")
	cached, err := keyStore.GetActiveKey(tenantContext(tenantA))
	if err != nil {
		t.Fatalf("load tenant-a key while Vault is reachable: %v", err)
	}
	server := newHealthServer(t, keyStore, groupmemory.NewGroupRepository(), healthUser("sys-operator", tenancydomain.DefaultTenantID, "system_admin"))
	const healthPath = "/realms/" + tenancydomain.DefaultRealm + "/api/admin/v1/keys/health"

	if before := decodeHealth(t, server.get(t, healthPath))[tenantA]; !before.Healthy || before.ActiveKid != cached.Kid {
		t.Fatalf("tenant-a health while reachable = %+v, want healthy with active kid %q", before, cached.Kid)
	}

	transit.setReachable(false)

	during := decodeHealth(t, server.get(t, healthPath))[tenantA]
	if during.TenantID != tenantA || during.Healthy {
		t.Fatalf("tenant-a health during the outage = %+v, want provider_healthy false", during)
	}
	jwks := server.get(t, "/realms/"+tenantA+"/jwks")
	if jwks.Code != http.StatusOK || !strings.Contains(jwks.Body.String(), `"kid":"`+cached.Kid+`"`) {
		t.Fatalf("tenant-a jwks during the outage status=%d body=%s, want the cached kid %q", jwks.Code, jwks.Body.String(), cached.Kid)
	}
}

//spec:covers EX-SIGNINGKEYS-009-01: 制御面テナントで admin だけを持つ operator と、他テナントで Group 経由の system_admin を持つ tenant-operator は、署名鍵ヘルス一覧を access_denied で拒否され、応答は他テナントの識別子、提供元、active_kid、鍵数、到達性を含まず、全テナントの読み出しも起きない。tenant-operator が自テナントの鍵一覧は読めることで、Group 由来のロールが効いていることを対照にする。
func TestSigningKeyHealthRefusesActorsOutsideTheControlPlane(t *testing.T) {
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	var kids []string
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, tenantA} {
		key, err := keyStore.GetActiveKey(tenantContext(tenantID))
		if err != nil {
			t.Fatal(err)
		}
		kids = append(kids, key.Kid)
	}

	groups := groupmemory.NewGroupRepository()
	if err := groups.Save(t.Context(), &groupdomain.Group{
		ID: "ops", TenantID: tenantA, Name: "ops", Roles: []string{"system_admin"},
		MembershipType: groupdomain.GroupMembershipManual,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(t.Context(), &groupdomain.GroupMember{
		GroupID: "ops", UserID: "tenant-operator", Source: groupdomain.MembershipSourceManual,
	}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		actor *userdomain.User
		realm string
	}{
		{"operator", healthUser("operator", tenancydomain.DefaultTenantID, "admin"), tenancydomain.DefaultRealm},
		{"tenant-operator", healthUser("tenant-operator", tenantA), tenantA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newHealthServer(t, keyStore, groups, tc.actor)
			refused := server.get(t, "/realms/"+tc.realm+"/api/admin/v1/keys/health")
			if refused.Code != http.StatusForbidden || !strings.Contains(refused.Body.String(), `"type":"urn:idmagic:error:access_denied"`) {
				t.Fatalf("status=%d body=%s, want 403 access_denied", refused.Code, refused.Body.String())
			}
			leaks := append([]string{"tenant_id", "provider", "active_kid", "jwks_key_count", "acme", "default"}, kids...)
			for _, leaked := range leaks {
				if strings.Contains(refused.Body.String(), leaked) {
					t.Errorf("refusal body carries %q: %s", leaked, refused.Body.String())
				}
			}
			if calls := server.tenants.findAll.Load(); calls != 0 {
				t.Errorf("TenantRepository.FindAll calls = %d, want no cross-tenant collection", calls)
			}
		})
	}

	own := newHealthServer(t, keyStore, groups, healthUser("tenant-operator", tenantA)).get(t, "/realms/"+tenantA+"/api/admin/v1/keys")
	if own.Code != http.StatusOK {
		t.Fatalf("tenant-operator own key list status=%d body=%s, want 200 through the Group-derived role", own.Code, own.Body.String())
	}
}
