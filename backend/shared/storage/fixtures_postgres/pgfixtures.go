// Package pgfixtures は per-context postgres アダプタのテストが共通で使う
// FK 親フィクスチャ (Tenant / User / Group / OAuth2Client) を提供する (wi-172)。
//
// shared/db_postgres 自身の内部テスト (package postgres) は、
// 本パッケージを import すると postgres -> pgfixtures -> postgres の import cycle に
// なるため使えず、引き続き自前の unexported ヘルパーを持つ。本パッケージは
// per-context (application 等) の postgres テストパッケージ専用。
package fixtures_postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	grouppg "github.com/ambi/idmagic/backend/idmanagement/group/db_postgres"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthpg "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/db_postgres"
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
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		tb.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

// SeedUser は指定テナントにユーザを作成して返す。
func SeedUser(tb testing.TB, db sharedpg.DB, tenantID string) *userdomain.User {
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
	if err := (&userpg.UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		tb.Fatalf("seed user: %v", err)
	}
	return user
}

// SeedGroup は指定テナントにグループを作成して返す。
func SeedGroup(tb testing.TB, db sharedpg.DB, tenantID string) *groupdomain.Group {
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
	if err := (&grouppg.GroupRepository{Pool: db}).Save(context.Background(), group); err != nil {
		tb.Fatalf("seed group: %v", err)
	}
	return group
}

// SeedClient は指定テナントに OAuth2 クライアントを作成して返す。
func SeedClient(tb testing.TB, db sharedpg.DB, tenantID string) *oauthdomain.OAuth2Client {
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
	if err := (&oauthpg.OAuth2ClientRepository{Pool: db}).Save(context.Background(), client); err != nil {
		tb.Fatalf("seed client: %v", err)
	}
	return client
}
