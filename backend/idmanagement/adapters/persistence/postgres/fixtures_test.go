package postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthpg "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
)

// 本パッケージは pgfixtures が依存する User/Group repository 自身を所有するため、
// pgfixtures を import すると postgres -> pgfixtures -> postgres の import cycle に
// なる。shared/adapters/persistence/postgres 自身の内部テストと同じ理由で、
// 引き続き自前の unexported フィクスチャヘルパーを持つ (wi-178)。

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

// uniqueID はテスト間の行衝突を避けるための一意な識別子を生成する。
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

// newUUID は UUID 列向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// seedTenant はテナントを作成して返す。FK 親が必要なテストの前提として使う。
func seedTenant(t *testing.T, db sharedpg.DB) *tenancydomain.Tenant {
	t.Helper()
	now := testClock()
	tenant := &tenancydomain.Tenant{
		ID:          newUUID(t),
		Realm:       uniqueID("tenant"),
		DisplayName: "Test Tenant",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

// seedUser は指定テナントにユーザを作成して返す。
func seedUser(t *testing.T, db sharedpg.DB, tenantID string) *userdomain.User {
	t.Helper()
	now := testClock()
	user := &userdomain.User{
		ID:                newUUID(t),
		TenantID:          tenantID,
		PreferredUsername: uniqueID("username"),
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := (&UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

// seedGroup は指定テナントにグループを作成して返す。
func seedGroup(t *testing.T, db sharedpg.DB, tenantID string) *groupdomain.Group {
	t.Helper()
	now := testClock()
	group := &groupdomain.Group{
		ID:        newUUID(t),
		TenantID:  tenantID,
		Name:      uniqueID("group-name"),
		Roles:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := (&GroupRepository{Pool: db}).Save(context.Background(), group); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	return group
}

// seedClient は指定テナントに OAuth2 クライアントを作成して返す。oauth2/adapters/persistence/postgres
// は本パッケージへ依存しないため、そのまま import して再利用できる。
func seedClient(t *testing.T, db sharedpg.DB, tenantID string) *oauthdomain.OAuth2Client {
	t.Helper()
	now := testClock()
	client := &oauthdomain.OAuth2Client{
		TenantID:                 tenantID,
		ClientID:                 newUUID(t),
		ClientType:               spec.ClientConfidential,
		ClientSecretHash:         new("secret-hash"),
		RedirectURIs:             []string{"https://client.example/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodClientSecretBasic,
		Scope:                    "openid offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := (&oauthpg.OAuth2ClientRepository{Pool: db}).Save(context.Background(), client); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}
