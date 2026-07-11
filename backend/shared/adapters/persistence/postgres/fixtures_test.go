package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

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

// seedUser は指定テナントにユーザを作成して返す。UserRepository は
// identitymanagement/adapters/persistence/postgres へ移設済み (wi-178) で、本パッケージの
// 内部テストから import すると postgres -> identitymanagement/postgres -> postgres の
// import cycle になるため、seedClient 同様 FK 充足専用の最小限フィクスチャとして
// 生 SQL で直接 INSERT する。
func seedUser(t *testing.T, db DB, tenantID string) *idmdomain.User {
	t.Helper()
	now := testClock()
	user := &idmdomain.User{
		ID:                newUUID(t),
		TenantID:          tenantID,
		PreferredUsername: uniqueID("username"),
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	lifecycle, err := json.Marshal(user.Lifecycle)
	if err != nil {
		t.Fatalf("marshal lifecycle: %v", err)
	}
	attributes, err := json.Marshal(user.Attributes)
	if err != nil {
		t.Fatalf("marshal attributes: %v", err)
	}
	_, err = db.Exec(context.Background(), `
INSERT INTO users (id,tenant_id,preferred_username,password_hash,roles,lifecycle,attributes,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		user.ID, user.TenantID, user.PreferredUsername, user.PasswordHash, user.Roles,
		string(lifecycle), string(attributes), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

// seedClient は指定テナントに OAuth2 クライアントを作成して返す。OAuth2ClientRepository は
// oauth2/adapters/persistence/postgres へ移設済み (wi-173) で、本パッケージの内部テストから
// import すると postgres -> oauth2/postgres -> postgres の import cycle になるため、
// FK 充足専用の最小限フィクスチャとして生 SQL で直接 INSERT する。
func seedClient(t *testing.T, db DB, tenantID string) *oauthdomain.OAuth2Client {
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
		IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	redirectURIs, err := json.Marshal(client.RedirectURIs)
	if err != nil {
		t.Fatalf("marshal redirect_uris: %v", err)
	}
	grantTypes, err := json.Marshal(client.GrantTypes)
	if err != nil {
		t.Fatalf("marshal grant_types: %v", err)
	}
	responseTypes, err := json.Marshal(client.ResponseTypes)
	if err != nil {
		t.Fatalf("marshal response_types: %v", err)
	}
	_, err = db.Exec(context.Background(), `
INSERT INTO clients (
 tenant_id,client_id,client_secret_hash,client_type,redirect_uris,grant_types,response_types,
 token_endpoint_auth_method,scope,id_token_signed_response_alg,fapi_profile,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		client.TenantID, client.ClientID, client.ClientSecretHash, client.ClientType,
		string(redirectURIs), string(grantTypes), string(responseTypes),
		client.TokenEndpointAuthMethod, client.Scope, client.IDTokenSignedResponseAlg,
		client.FapiProfile, client.CreatedAt, client.UpdatedAt)
	if err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}
