// Package fixtures_postgres は PostgreSQL テストの前提を組み立てる。
// 各所有モジュールのアダプターを通じて FK の親行を投入するテスト専用の組み立て地点である。
// storage/db_postgres の内部テストから使うと import が循環するため、そこでの前提は内部で組み立てる。
// User、Group、OAuth2 クライアントは、所有モジュールの内側のテストからも使えるよう、所有モジュールの
// ルートパッケージを import せず、保存先を呼び出し元から受け取る。
package fixtures_postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"

	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
)

// TestClock は決定的なタイムスタンプ生成に用いる基準時刻。
func TestClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

// UniqueID はテスト間の行衝突を避けるための一意な識別子を生成する。
func UniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

// NewUUID は UUID 列向けの一意な UUID を生成する。
func NewUUID(tb testing.TB) string {
	tb.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		tb.Fatalf("new uuid: %v", err)
	}
	return id
}

// SeedTenant はテナントを作成して返す。FK 親が必要なテストの前提として使う。
func SeedTenant(tb testing.TB, db sharedpg.DB) *tenancydomain.Tenant {
	tb.Helper()
	now := TestClock()
	tenant := &tenancydomain.Tenant{
		ID:          NewUUID(tb),
		Realm:       UniqueID("tenant"),
		DisplayName: "Test Tenant",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
		tb.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

// SeedUser は指定テナントにユーザを作成して返す。users は IdManagement の PostgreSQL の保存先である。
func SeedUser(tb testing.TB, users userports.UserRepository, tenantID string) *userdomain.User {
	tb.Helper()
	now := TestClock()
	user := &userdomain.User{
		ID:                NewUUID(tb),
		TenantID:          tenantID,
		PreferredUsername: UniqueID("username"),
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := users.Save(context.Background(), user); err != nil {
		tb.Fatalf("seed user: %v", err)
	}
	return user
}

// SeedGroup は指定テナントにグループを作成して返す。groups は IdManagement の PostgreSQL の保存先である。
func SeedGroup(tb testing.TB, groups groupports.GroupRepository, tenantID string) *groupdomain.Group {
	tb.Helper()
	now := TestClock()
	group := &groupdomain.Group{
		ID:        NewUUID(tb),
		TenantID:  tenantID,
		Name:      UniqueID("group-name"),
		Roles:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := groups.Save(context.Background(), group); err != nil {
		tb.Fatalf("seed group: %v", err)
	}
	return group
}

// SeedClient は指定テナントに OAuth2 クライアントを作成して返す。clients は OAuth2 の
// PostgreSQL の保存先である。OAuth2 の内側のテストがルートパッケージを経ずに渡せるよう、
// 保存先を呼び出し元から受け取る。
func SeedClient(tb testing.TB, clients oauthports.OAuth2ClientRepository, tenantID string) *oauthdomain.OAuth2Client {
	tb.Helper()
	now := TestClock()
	client := &oauthdomain.OAuth2Client{
		TenantID:                 tenantID,
		ClientID:                 NewUUID(tb),
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
	if err := clients.Save(context.Background(), client); err != nil {
		tb.Fatalf("seed client: %v", err)
	}
	return client
}
