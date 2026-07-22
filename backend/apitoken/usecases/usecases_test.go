package usecases

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

type fakeRepository struct {
	saved   []*domain.ApiToken
	byHash  map[string]*domain.ApiToken
	deleted [][2]string
}

func (r *fakeRepository) Save(_ context.Context, token *domain.ApiToken) error {
	clone := *token
	clone.Scopes = append(domain.Scopes(nil), token.Scopes...)
	r.saved = append(r.saved, &clone)
	if r.byHash == nil {
		r.byHash = map[string]*domain.ApiToken{}
	}
	r.byHash[token.TokenHash] = &clone
	return nil
}

func (r *fakeRepository) FindByHash(_ context.Context, hash string) (*domain.ApiToken, error) {
	return r.byHash[hash], nil
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

func (r *fakeRepository) Delete(_ context.Context, tenantID, id string) error {
	r.deleted = append(r.deleted, [2]string{tenantID, id})
	return nil
}

// SCL interfaces.IssueApiToken / ListApiTokens / RevokeApiToken.
func TestIssueListAndRevokeApiToken(t *testing.T) {
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	service := New(repo, WithClock(func() time.Time { return now }), WithRandomReader(bytes.NewReader(make([]byte, 32))))

	literal, meta, err := service.Issue(context.Background(), "tenant-1", "SCIM sync", []string{string(domain.ScopeScimUsersRead)}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if literal != "idmagic_pat_0000000000000000000000000000000000000000000000000000000000000000" {
		t.Fatalf("literal = %q", literal)
	}
	if len(repo.saved) != 1 || repo.saved[0].TokenHash == literal || repo.saved[0].TokenHash == "" {
		t.Fatalf("plaintext was persisted or hash missing: %+v", repo.saved)
	}
	if meta.ID == "" || !meta.Scopes.Has(domain.ScopeScimUsersRead) {
		t.Fatalf("metadata = %+v", meta)
	}
	if meta.ExpiresAt == nil || !meta.ExpiresAt.Equal(now.AddDate(0, 0, 7)) {
		t.Fatalf("expires_at = %v", meta.ExpiresAt)
	}

	listed, err := service.List(context.Background(), "tenant-1")
	if err != nil || len(listed) != 1 || listed[0].ID != meta.ID {
		t.Fatalf("list = %+v, err = %v", listed, err)
	}
	if err := service.Revoke(context.Background(), "tenant-1", meta.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != [2]string{"tenant-1", meta.ID} {
		t.Fatalf("deleted = %v", repo.deleted)
	}
}

func TestIssueRejectsInvalidInput(t *testing.T) {
	repo := &fakeRepository{}
	service := New(repo)
	for _, tc := range []struct {
		name       string
		scopes     []string
		expiryDays int
	}{
		{name: "non-positive expiry", scopes: []string{string(domain.ScopeScimUsersRead)}, expiryDays: 0},
		{name: "unknown scope", scopes: []string{"scim:unknown"}, expiryDays: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := service.Issue(context.Background(), "tenant-1", "test", tc.scopes, tc.expiryDays); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	if len(repo.saved) != 0 {
		t.Fatalf("saved invalid tokens: %d", len(repo.saved))
	}
}

// SCL interfaces.AuthenticateApiToken.
func TestAuthenticateApiTokenFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	validLiteral := "idmagic_pat_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	parsed, err := domain.ParseTokenLiteral(validLiteral)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	for _, tc := range []struct {
		name    string
		literal string
		token   *domain.ApiToken
		wantErr bool
	}{
		{name: "valid", literal: validLiteral, token: &domain.ApiToken{TenantID: "tenant-1", TokenHash: parsed.Hash(), Scopes: domain.Scopes{domain.ScopeScimUsersRead}, ExpiresAt: &future}},
		{name: "malformed", literal: "not-a-token", wantErr: true},
		{name: "unknown", literal: validLiteral, wantErr: true},
		{name: "expired", literal: validLiteral, token: &domain.ApiToken{TenantID: "tenant-1", TokenHash: parsed.Hash(), Scopes: domain.Scopes{domain.ScopeScimUsersRead}, ExpiresAt: &past}, wantErr: true},
		{name: "empty scopes", literal: validLiteral, token: &domain.ApiToken{TenantID: "tenant-1", TokenHash: parsed.Hash()}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{byHash: map[string]*domain.ApiToken{}}
			if tc.token != nil {
				repo.byHash[tc.token.TokenHash] = tc.token
			}
			principal, err := New(repo, WithClock(func() time.Time { return now })).Authenticate(context.Background(), tc.literal)
			if tc.wantErr {
				if !errors.Is(err, ErrAccessDenied) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil || principal.TenantID != "tenant-1" || !principal.Scopes.Has(domain.ScopeScimUsersRead) {
				t.Fatalf("principal = %+v, err = %v", principal, err)
			}
		})
	}
}
