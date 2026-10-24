package usecases

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"maps"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// racingRepository は、Update の前に stale 回だけほかの更新が割り込んだように振る舞う。
type racingRepository struct {
	*db_memory.ConnectorRepository
	stale   int
	updates int
}

func (r *racingRepository) Update(ctx context.Context, c domain.Connector) error {
	r.updates++
	if r.stale > 0 {
		r.stale--
		return ports.ErrStaleConnector
	}
	return r.ConnectorRepository.Update(ctx, c)
}

func newRacingService(t *testing.T) (*Service, *racingRepository, Grant) {
	t.Helper()
	repo := &racingRepository{ConnectorRepository: db_memory.NewConnectorRepository()}
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	service := New(repo, nil, nil, WithClock(func() time.Time { return now }))
	grant, err := service.Create(context.Background(), "tenant", "東京の AD")
	if err != nil {
		t.Fatal(err)
	}
	return service, repo, grant
}

var ecJWK = map[string]any{
	"kty": "EC", "crv": "P-256",
	"x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
	"y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
}

//spec:covers REQ-SOURCING-009
func TestEnrollThatLosesARaceIsAnInvalidToken(t *testing.T) {
	service, repo, grant := newRacingService(t)
	repo.stale = 1
	if _, err := service.Enroll(context.Background(), "tenant", grant.Token, ecJWK); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Enroll error = %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("Enroll retried %d times", repo.updates)
	}
}

//spec:covers REQ-SOURCING-009
func TestEnrollStoresOnlyThePublicMembersOfTheKey(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaJWK := map[string]any{
		"kty": "RSA", "alg": "PS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(private.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(private.E)).Bytes()),
	}
	ecWithExtras := maps.Clone(ecJWK)
	ecWithExtras["alg"], ecWithExtras["kid"] = "ES256", "chosen-by-the-connector"
	for _, tc := range []struct {
		jwk  map[string]any
		want []string
	}{
		{rsaJWK, []string{"e", "kid", "kty", "n"}},
		{ecWithExtras, []string{"crv", "kid", "kty", "x", "y"}},
	} {
		service, _, grant := newRacingService(t)
		enrolled, err := service.Enroll(context.Background(), "tenant", grant.Token, tc.jwk)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Sorted(maps.Keys(enrolled.PublicKey)); !slices.Equal(got, tc.want) {
			t.Fatalf("stored members = %v, want %v", got, tc.want)
		}
		if enrolled.PublicKey["kid"] != enrolled.KeyID || enrolled.KeyID == "chosen-by-the-connector" {
			t.Fatalf("kid = %v, key id = %q", enrolled.PublicKey["kid"], enrolled.KeyID)
		}
	}
}

//spec:covers REQ-SOURCING-012
func TestDisableRereadsAfterAConcurrentUpdate(t *testing.T) {
	service, repo, grant := newRacingService(t)
	repo.stale = maxStaleRetries - 1
	disabled, err := service.Disable(context.Background(), "tenant", grant.Connector.ID)
	if err != nil || disabled.Status != domain.StatusDisabled {
		t.Fatalf("Disable = %+v, %v", disabled, err)
	}
	if repo.updates != maxStaleRetries {
		t.Fatalf("updates = %d, want %d", repo.updates, maxStaleRetries)
	}

	service, repo, grant = newRacingService(t)
	repo.stale = maxStaleRetries
	if _, err := service.Disable(context.Background(), "tenant", grant.Connector.ID); !errors.Is(err, ports.ErrStaleConnector) {
		t.Fatalf("Disable with a repository that is never current = %v", err)
	}
}
