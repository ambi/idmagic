// Package pgfixtures は per-context postgres アダプタのテストが共通で使う
// FK 親フィクスチャ (Tenant / User / Group / OAuth2Client) を提供する (wi-172)。
//
// shared/adapters/persistence/postgres 自身の内部テスト (package postgres) は、
// 本パッケージを import すると postgres -> pgfixtures -> postgres の import cycle に
// なるため使えず、引き続き自前の unexported ヘルパーを持つ。本パッケージは
// per-context (application 等) の postgres テストパッケージ専用。
package pgfixtures

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmpg "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/postgres"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	oauthpg "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
)

// TestClock は決定的なタイムスタンプ生成に用いる基準時刻。
func TestClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

// UniqueID はテスト間の行衝突を避けるための一意な識別子を生成する。
func UniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

// NewUUID は UUID 列向けの一意な UUID を生成する。
func NewUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// SeedTenant はテナントを作成して返す。FK 親が必要なテストの前提として使う。
func SeedTenant(t *testing.T, db sharedpg.DB) *tenancydomain.Tenant {
	t.Helper()
	now := TestClock()
	tenant := &tenancydomain.Tenant{
		ID:          NewUUID(t),
		Realm:       UniqueID("tenant"),
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

// SeedUser は指定テナントにユーザを作成して返す。
func SeedUser(t *testing.T, db sharedpg.DB, tenantID string) *idmdomain.User {
	t.Helper()
	now := TestClock()
	user := &idmdomain.User{
		ID:                NewUUID(t),
		TenantID:          tenantID,
		PreferredUsername: UniqueID("username"),
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := (&idmpg.UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

// SeedGroup は指定テナントにグループを作成して返す。
func SeedGroup(t *testing.T, db sharedpg.DB, tenantID string) *idmdomain.Group {
	t.Helper()
	now := TestClock()
	group := &idmdomain.Group{
		ID:        NewUUID(t),
		TenantID:  tenantID,
		Name:      UniqueID("group-name"),
		Roles:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := (&idmpg.GroupRepository{Pool: db}).Save(context.Background(), group); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	return group
}

// SeedClient は指定テナントに OAuth2 クライアントを作成して返す。
func SeedClient(t *testing.T, db sharedpg.DB, tenantID string) *oauthdomain.OAuth2Client {
	t.Helper()
	now := TestClock()
	client := &oauthdomain.OAuth2Client{
		TenantID:                 tenantID,
		ClientID:                 NewUUID(t),
		ClientType:               spec.ClientConfidential,
		ClientSecretHash:         new("secret-hash"),
		RedirectURIs:             []string{"https://client.example/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodClientSecretBasic,
		Scope:                    "openid offline_access",
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := (&oauthpg.OAuth2ClientRepository{Pool: db}).Save(context.Background(), client); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}
