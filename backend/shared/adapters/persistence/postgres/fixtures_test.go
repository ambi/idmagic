package postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

// uniqueID はテスト間の行衝突を避けるための一意な識別子を生成する。
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

// newUUID は UUID 列 (refresh_tokens.id 等) 向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// seedGroup は指定テナントにグループを作成して返す。
func seedGroup(t *testing.T, db DB, tenantID string) *spec.Group {
	t.Helper()
	now := testClock()
	group := &spec.Group{
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

// seedTenant はテナントを作成して返す。FK 親が必要なテストの前提として使う。
func seedTenant(t *testing.T, db DB) *spec.Tenant {
	t.Helper()
	now := testClock()
	tenant := &spec.Tenant{
		ID:          newUUID(t),
		Realm:       uniqueID("tenant"),
		DisplayName: "Test Tenant",
		Status:      spec.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

// seedUser は指定テナントにユーザを作成して返す。
func seedUser(t *testing.T, db DB, tenantID string) *spec.User {
	t.Helper()
	now := testClock()
	user := &spec.User{
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

// seedClient は指定テナントに OAuth2 クライアントを作成して返す。
func seedClient(t *testing.T, db DB, tenantID string) *spec.OAuth2Client {
	t.Helper()
	now := testClock()
	client := &spec.OAuth2Client{
		TenantID:                 tenantID,
		ClientID:                 newUUID(t),
		ClientType:               spec.ClientConfidential,
		ClientSecretHash:         new("secret-hash"),
		RedirectURIs:             []string{"https://client.example/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  spec.AuthMethodClientSecretBasic,
		Scope:                    "openid offline_access",
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              spec.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := (&OAuth2ClientRepository{Pool: db}).Save(context.Background(), client); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}
