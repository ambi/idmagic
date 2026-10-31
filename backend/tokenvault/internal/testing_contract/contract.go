// Package testing_contract はメモリと PostgreSQL の保存先が共有する契約を定める。
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// Fixture は契約の入力である。TenantA と UserA は実在し、ConnectorID は UUID である。
type Fixture struct {
	Connectors              ports.ConnectorRepository
	Connections             ports.ConnectionRepository
	Attempts                ports.AttemptStore
	TenantA, TenantB, UserA string
	NewID                   func() string
	Now                     time.Time
}

func connector(f Fixture, name string) *domain.Connector {
	return &domain.Connector{
		ID: f.NewID(), TenantID: f.TenantA, Name: name, Status: domain.ConnectorActive,
		AuthorizationEndpoint: "https://idp.example.com/a", TokenEndpoint: "https://idp.example.com/t",
		ClientID: "client", ClientSecret: "client-secret", TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		AllowedScopes: []string{"read"},
		Rules:         []domain.ExecutionRule{{ID: "r", Method: "GET", Origin: "https://api.example.com", PathPrefix: "/v1", RequiredScopes: []string{"read"}}},
		Version:       1, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
}

func linked(f Fixture, connectorID, access, refresh string) *domain.Connection {
	expires := f.Now.Add(time.Hour)
	return &domain.Connection{
		ID: f.NewID(), TenantID: f.TenantA, ConnectorID: connectorID, UserID: f.UserA, Status: domain.ConnectionActive,
		GrantedScopes: []string{"read"}, AccessToken: access, RefreshToken: refresh, ExpiresAt: &expires, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
}

// Run は契約を検査する。
func Run(t *testing.T, newFixture func(*testing.T) Fixture) {
	t.Helper()
	t.Run("connector tenant isolation and optimistic update", func(t *testing.T) { connectorContract(t, newFixture(t)) })
	t.Run("connection relink keeps identity", func(t *testing.T) { relinkContract(t, newFixture(t)) })
	t.Run("refresh lease and version guard", func(t *testing.T) { leaseContract(t, newFixture(t)) })
	t.Run("reauthorization clears tokens", func(t *testing.T) { reauthorizationContract(t, newFixture(t)) })
	t.Run("attempt is bound and consumed once", func(t *testing.T) { attemptContract(t, newFixture(t)) })
	t.Run("connector delete removes connections", func(t *testing.T) { deleteContract(t, newFixture(t)) })
}

func connectorContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	created := connector(f, "c1")
	if err := f.Connectors.Create(ctx, created); err != nil {
		t.Fatal(err)
	}
	duplicate := connector(f, "c1")
	if err := f.Connectors.Create(ctx, duplicate); err != ports.ErrConnectorConflict { //nolint:errorlint // 契約は値そのものを返すこと。
		t.Fatalf("duplicate name: %v", err)
	}
	found, err := f.Connectors.Find(ctx, f.TenantA, created.ID)
	if err != nil || found == nil || found.ClientSecret != "client-secret" || len(found.Rules) != 1 {
		t.Fatalf("find: %+v, %v", found, err)
	}
	if other, err := f.Connectors.Find(ctx, f.TenantB, created.ID); err != nil || other != nil {
		t.Fatalf("other tenant read: %+v, %v", other, err)
	}
	if missing, err := f.Connectors.Find(ctx, f.TenantA, "not-a-uuid"); err != nil || missing != nil {
		t.Fatalf("malformed id: %+v, %v", missing, err)
	}
	stale := *found
	found.Name = "c1 renamed"
	if err := f.Connectors.Update(ctx, found); err != nil {
		t.Fatal(err)
	}
	if err := f.Connectors.Update(ctx, &stale); err != ports.ErrConnectorConflict { //nolint:errorlint // 同上。
		t.Fatalf("stale update: %v", err)
	}
}

func relinkContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	c := connector(f, "relink")
	if err := f.Connectors.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a1", "r1")); err != nil {
		t.Fatal(err)
	}
	first, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || first == nil || first.AccessToken != "a1" || first.RefreshToken != "r1" || first.TokenVersion != 1 {
		t.Fatalf("first: %+v, %v", first, err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a2", "")); err != nil {
		t.Fatal(err)
	}
	second, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || second.ID != first.ID || second.AccessToken != "a2" || second.RefreshToken != "" || second.TokenVersion != 2 {
		t.Fatalf("second: %+v, %v", second, err)
	}
	mine, err := f.Connections.ListByUser(ctx, f.TenantA, f.UserA)
	if err != nil || len(mine) != 1 {
		t.Fatalf("list: %d, %v", len(mine), err)
	}
	deleted, err := f.Connections.Delete(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || deleted == nil || deleted.AccessToken != "a2" {
		t.Fatalf("delete: %+v, %v", deleted, err)
	}
	if again, err := f.Connections.Delete(ctx, f.TenantA, c.ID, f.UserA); err != nil || again != nil {
		t.Fatalf("second delete: %+v, %v", again, err)
	}
}

func leaseContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	c := connector(f, "lease")
	if err := f.Connectors.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a1", "r1")); err != nil {
		t.Fatal(err)
	}
	row, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil {
		t.Fatal(err)
	}
	until := f.Now.Add(30 * time.Second)
	acquire := func(at time.Time, version int64) bool {
		t.Helper()
		ok, err := f.Connections.AcquireRefreshLease(ctx, f.TenantA, row.ID, version, at, until)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	if !acquire(f.Now, row.TokenVersion) {
		t.Fatal("first lease refused")
	}
	if acquire(f.Now, row.TokenVersion) {
		t.Fatal("second holder acquired a held lease")
	}
	if !acquire(until, row.TokenVersion) {
		t.Fatal("expired lease was not reacquirable")
	}
	if err := f.Connections.ReleaseRefreshLease(ctx, f.TenantA, row.ID, row.TokenVersion); err != nil {
		t.Fatal(err)
	}
	if !acquire(f.Now, row.TokenVersion) {
		t.Fatal("released lease was not acquirable")
	}
	updated := *row
	updated.AccessToken, updated.RefreshToken = "a2", "r2"
	stored, err := f.Connections.StoreRefreshed(ctx, &updated, row.TokenVersion)
	if err != nil || !stored {
		t.Fatalf("store: %v, %v", stored, err)
	}
	if stored, err := f.Connections.StoreRefreshed(ctx, &updated, row.TokenVersion); err != nil || stored {
		t.Fatalf("stale store: %v, %v", stored, err)
	}
	if acquire(f.Now, row.TokenVersion) {
		t.Fatal("stale version acquired a lease")
	}
	current, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || current.AccessToken != "a2" || current.RefreshToken != "r2" || current.TokenVersion != row.TokenVersion+1 {
		t.Fatalf("current: %+v, %v", current, err)
	}
}

func reauthorizationContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	c := connector(f, "reauth")
	if err := f.Connectors.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a1", "r1")); err != nil {
		t.Fatal(err)
	}
	row, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := f.Connections.MarkReauthorizationRequired(ctx, f.TenantA, row.ID, row.TokenVersion+1, f.Now); err != nil || changed {
		t.Fatalf("stale mark: %v, %v", changed, err)
	}
	if changed, err := f.Connections.MarkReauthorizationRequired(ctx, f.TenantA, row.ID, row.TokenVersion, f.Now); err != nil || !changed {
		t.Fatalf("mark: %v, %v", changed, err)
	}
	current, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || current.Status != domain.ConnectionReauthorizationRequired || current.AccessToken != "" || current.RefreshToken != "" {
		t.Fatalf("current: %+v, %v", current, err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a3", "r3")); err != nil {
		t.Fatal(err)
	}
	relinked, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA)
	if err != nil || relinked.Status != domain.ConnectionActive || relinked.AccessToken != "a3" {
		t.Fatalf("relinked: %+v, %v", relinked, err)
	}
}

func attemptContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	c := connector(f, "attempt")
	if err := f.Connectors.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	attempt := &domain.ConnectionAttempt{
		ID: f.NewID(), TenantID: f.TenantA, ConnectorID: c.ID, UserID: f.UserA, StateHash: []byte("hash-1"),
		CodeVerifier: "verifier", RequestedScopes: []string{"read"}, ExpiresAt: f.Now.Add(time.Minute),
	}
	if err := f.Attempts.Save(ctx, attempt); err != nil {
		t.Fatal(err)
	}
	consume := func(user string, hash []byte, at time.Time) *domain.ConnectionAttempt {
		t.Helper()
		found, err := f.Attempts.Consume(ctx, f.TenantA, c.ID, user, hash, at)
		if err != nil {
			t.Fatal(err)
		}
		return found
	}
	if consume(f.UserA, []byte("other"), f.Now) != nil {
		t.Fatal("another state consumed the attempt")
	}
	if consume(f.NewID(), attempt.StateHash, f.Now) != nil {
		t.Fatal("another user consumed the attempt")
	}
	if consume(f.UserA, attempt.StateHash, attempt.ExpiresAt) != nil {
		t.Fatal("an expired attempt was consumed")
	}
	found := consume(f.UserA, attempt.StateHash, f.Now)
	if found == nil || found.CodeVerifier != "verifier" || len(found.RequestedScopes) != 1 {
		t.Fatalf("consume: %+v", found)
	}
	if consume(f.UserA, attempt.StateHash, f.Now) != nil {
		t.Fatal("the attempt was consumed twice")
	}
}

func deleteContract(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	c := connector(f, "delete")
	if err := f.Connectors.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := f.Connections.SaveLinked(ctx, linked(f, c.ID, "a1", "")); err != nil {
		t.Fatal(err)
	}
	if deleted, err := f.Connectors.Delete(ctx, f.TenantB, c.ID); err != nil || deleted {
		t.Fatalf("other tenant delete: %v, %v", deleted, err)
	}
	if deleted, err := f.Connectors.Delete(ctx, f.TenantA, c.ID); err != nil || !deleted {
		t.Fatalf("delete: %v, %v", deleted, err)
	}
	if found, err := f.Connections.Find(ctx, f.TenantA, c.ID, f.UserA); err != nil || found != nil {
		t.Fatalf("connection survived: %+v, %v", found, err)
	}
}
