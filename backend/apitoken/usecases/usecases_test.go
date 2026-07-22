package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type fakeRepository struct {
	saved   []*domain.ApiToken
	byJTI   map[string]*domain.ApiToken
	revoked [][3]string
}

func (r *fakeRepository) Save(_ context.Context, token *domain.ApiToken) error {
	clone := *token
	clone.Scopes = append(domain.Scopes(nil), token.Scopes...)
	r.saved = append(r.saved, &clone)
	if r.byJTI == nil {
		r.byJTI = map[string]*domain.ApiToken{}
	}
	r.byJTI[token.JTI] = &clone
	return nil
}

func (r *fakeRepository) FindByJTI(_ context.Context, tenantID, jti string) (*domain.ApiToken, error) {
	token := r.byJTI[jti]
	if token == nil || token.TenantID != tenantID {
		return nil, nil //nolint:nilnil // repository absence is represented by a nil aggregate.
	}
	return token, nil
}

func (r *fakeRepository) List(_ context.Context, tenantID string) ([]*domain.ApiToken, error) {
	var result []*domain.ApiToken
	for _, token := range r.saved {
		if token.TenantID == tenantID {
			result = append(result, token)
		}
	}
	return result, nil
}

func (r *fakeRepository) Revoke(_ context.Context, tenantID, id string, at time.Time) error {
	r.revoked = append(r.revoked, [3]string{tenantID, id, at.Format(time.RFC3339Nano)})
	for _, token := range r.saved {
		if token.TenantID == tenantID && token.ID == id {
			token.RevokedAt = &at
		}
	}
	return nil
}

func (r *fakeRepository) RevokeByJTI(_ context.Context, tenantID, jti string, at time.Time) error {
	for _, token := range r.saved {
		if token.TenantID == tenantID && token.JTI == jti {
			token.RevokedAt = &at
		}
	}
	if token := r.byJTI[jti]; token != nil && token.TenantID == tenantID {
		token.RevokedAt = &at
	}
	return nil
}

type fakeTokenIssuer struct{ input ports.AccessTokenInput }

func (f *fakeTokenIssuer) SignAccessToken(_ context.Context, in ports.AccessTokenInput) (string, string, error) {
	f.input = in
	return "header.payload.signature", "jti-1", nil
}

func (*fakeTokenIssuer) SignIDToken(context.Context, ports.IDTokenInput) (string, error) {
	return "", nil
}
func (*fakeTokenIssuer) AccessTokenTTLSeconds() int { return 600 }
func (*fakeTokenIssuer) IDTokenTTLSeconds() int     { return 3600 }

type fakeIntrospector struct{ result *ports.IntrospectionResult }

func (f fakeIntrospector) IntrospectAccessToken(context.Context, string) (*ports.IntrospectionResult, error) {
	return f.result, nil
}

func testService(now time.Time, repo *fakeRepository, issuer *fakeTokenIssuer) *Service {
	return New(repo, WithTokenIssuer(issuer), WithClock(func() time.Time { return now }))
}

func TestManagedTokenIntrospectionUsesLifecycleRecord(t *testing.T) {
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	claims := &ports.IntrospectionResult{
		Active: true, Managed: true, JTI: "jti-1", ClientID: BuiltinClientID,
		Sub: "admin", Scope: "account:read", Aud: []string{"https://idp.test/realms/acme"}, Iat: now.Unix(), Exp: future.Unix(),
	}
	record := &domain.ApiToken{
		ID: "token-1", TenantID: "tenant-1", UserID: "admin", JTI: "jti-1", ClientID: BuiltinClientID,
		Scopes: domain.Scopes{domain.ScopeAccountRead}, Audience: claims.Aud[0], CreatedAt: now, ExpiresAt: &future,
	}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-1"}, claims.Aud[0], "/realms/acme")
	for _, tc := range []struct {
		name   string
		record *domain.ApiToken
		active bool
	}{
		{name: "active record", record: record, active: true}, {name: "missing record", active: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{byJTI: map[string]*domain.ApiToken{}}
			if tc.record != nil {
				repo.byJTI[tc.record.JTI] = tc.record
			}
			service := New(repo, WithTokenIntrospector(fakeIntrospector{result: claims}), WithClock(func() time.Time { return now }))
			got, err := service.IntrospectAccessToken(ctx, "jwt")
			if err != nil || got.Active != tc.active {
				t.Fatalf("result=%+v err=%v", got, err)
			}
			if !tc.active && (got.JTI != "" || got.Sub != "") {
				t.Fatalf("inactive response leaked metadata: %+v", got)
			}
		})
	}
}

// SCL interfaces.IssueApiToken / ADR-137: 発行者と選択 client を持つ managed JWT を一度だけ返す。
func TestIssueListAndRevokeApiToken(t *testing.T) {
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	repo, issuer := &fakeRepository{}, &fakeTokenIssuer{}
	service := testService(now, repo, issuer)

	literal, meta, err := service.Issue(context.Background(), "tenant-1", "admin-1", "CLI", []string{string(domain.ScopeAccountRead)}, 7, "")
	if err != nil {
		t.Fatal(err)
	}
	if literal != "header.payload.signature" {
		t.Fatalf("literal = %q", literal)
	}
	if len(repo.saved) != 1 || repo.saved[0].JTI != "jti-1" {
		t.Fatalf("record = %+v", repo.saved)
	}
	if repo.saved[0].UserID != "admin-1" || repo.saved[0].ClientID != BuiltinClientID {
		t.Fatalf("record identity = %+v", repo.saved[0])
	}
	if issuer.input.Sub != "admin-1" || issuer.input.Client.ClientID != BuiltinClientID || !issuer.input.Managed {
		t.Fatalf("issuer input = %+v", issuer.input)
	}
	if issuer.input.ExpiresAt != now.AddDate(0, 0, 7).Unix() {
		t.Fatalf("exp = %d", issuer.input.ExpiresAt)
	}
	if meta.JTI != "jti-1" || meta.UserID != "admin-1" || meta.ClientID != BuiltinClientID {
		t.Fatalf("metadata = %+v", meta)
	}

	listed, err := service.List(context.Background(), "tenant-1")
	if err != nil || len(listed) != 1 || listed[0].ID != meta.ID {
		t.Fatalf("list = %+v, err = %v", listed, err)
	}
	if err := service.Revoke(context.Background(), "tenant-1", meta.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.revoked) != 1 || repo.revoked[0][0] != "tenant-1" {
		t.Fatalf("revoked = %v", repo.revoked)
	}
}

// SCL interfaces.AuthenticateApiToken: JWT claim と管理 record の双方を一致検証する。
func TestAuthenticateManagedTokenFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	future, past := now.Add(time.Hour), now.Add(-time.Hour)
	base := domain.ApiToken{ID: "token-1", TenantID: "tenant-1", UserID: "admin", JTI: "jti-1", ClientID: "cli", Audience: "https://idp/realms/tenant-1", Scopes: domain.Scopes{domain.ScopeAccountRead}, CreatedAt: now, ExpiresAt: &future}
	for _, tc := range []struct {
		name    string
		record  *domain.ApiToken
		claims  ports.IntrospectionResult
		wantErr bool
	}{
		{name: "valid", record: &base, claims: ports.IntrospectionResult{Active: true, Managed: true, JTI: "jti-1", ClientID: "cli", Sub: "admin", Scope: "account:read", Aud: []string{"https://idp/realms/tenant-1"}, Iat: now.Unix(), Exp: future.Unix()}},
		{name: "missing record", claims: ports.IntrospectionResult{Active: true, Managed: true, JTI: "jti-1"}, wantErr: true},
		{name: "expired record", record: func() *domain.ApiToken { x := base; x.ExpiresAt = &past; return &x }(), claims: ports.IntrospectionResult{Active: true, Managed: true, JTI: "jti-1"}, wantErr: true},
		{name: "subject mismatch", record: &base, claims: ports.IntrospectionResult{Active: true, Managed: true, JTI: "jti-1", ClientID: "cli", Sub: "other", Scope: "account:read", Aud: []string{base.Audience}}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{byJTI: map[string]*domain.ApiToken{}}
			if tc.record != nil {
				repo.byJTI[tc.record.JTI] = tc.record
			}
			principal, err := New(repo, WithTokenIssuer(&fakeTokenIssuer{}), WithClock(func() time.Time { return now })).AuthenticateClaims(context.Background(), "tenant-1", tc.claims)
			if tc.wantErr {
				if !errors.Is(err, ErrAccessDenied) {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil || principal.UserID != "admin" || principal.ClientID != "cli" {
				t.Fatalf("principal=%+v err=%v", principal, err)
			}
		})
	}
}
