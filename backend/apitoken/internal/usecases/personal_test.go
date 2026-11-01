package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/internal/db_memory"
)

//spec:covers REQ-APITOKENS-005: 境界の日数を受理し、管理スコープと範囲外の日数は保存しない。
func TestPersonalTokenScopeAndExpiryBounds(t *testing.T) {
	for _, days := range []int{1, 365} {
		repo := db_memory.NewRepository()
		issuer := &fakeTokenIssuer{}
		now := time.Now().UTC()
		service := New(repo, WithTokenIssuer(issuer), WithClock(func() time.Time { return now }))
		_, metadata, err := service.IssuePersonal(t.Context(), "tenant", "user", "personal", []string{"account:read", "account:write", "account:mfa:write", "account:sessions:write", "account:consents:write", "account:password:write"}, days, "")
		if err != nil || metadata.ExpiresAt == nil || !metadata.ExpiresAt.Equal(now.AddDate(0, 0, days)) {
			t.Fatalf("days=%d metadata=%+v error=%v", days, metadata, err)
		}
		for _, input := range []struct {
			days   int
			scopes []string
		}{
			{0, []string{"account:read"}},
			{366, []string{"account:read"}},
			{7, nil},
			{7, []string{"users:write"}},
			{7, []string{"account:read", "scim:users:write"}},
			{7, []string{"account:unknown"}},
		} {
			if _, _, err := service.IssuePersonal(t.Context(), "tenant", "user", "bad", input.scopes, input.days, ""); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("input=%+v error=%v", input, err)
			}
		}
		tokens, err := repo.List(t.Context(), "tenant")
		if err != nil || len(tokens) != 1 {
			t.Fatalf("refused requests changed records: count=%d error=%v", len(tokens), err)
		}
	}
}
