package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// deleteFixture は同じテナントに 2 つの接続を置く。連携の有無を接続ごとに変え、
// 拒否が「対象の接続に」連携が残っているかで決まることを読めるようにする。
func deleteFixture(t *testing.T) (federationusecases.BrokerDeps, federationmemory.Repositories) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	for _, id := range []string{"linked", "unlinked"} {
		if err := repos.Connections.Save(context.Background(), &federationdomain.IdentityProviderConnection{
			ID: id, TenantID: tenancydomain.DefaultTenantID, DisplayName: id,
			Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionDisabled,
			Issuer: "https://" + id + ".example", ClientID: "client",
			AuthorizationEndpoint: "https://" + id + ".example/auth",
			TokenEndpoint:         "https://" + id + ".example/token", JWKSURI: "https://" + id + ".example/jwks",
			ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
			LinkingPolicy: federationdomain.LinkingNone,
		}); err != nil {
			t.Fatalf("save connection %s: %v", id, err)
		}
	}
	if err := repos.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: "linked", ExternalSubject: "external-1",
		LocalUserID: "user-1", LinkedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("link identity: %v", err)
	}
	return federationusecases.BrokerDeps{Connections: repos.Connections, Identities: repos.Identities}, repos
}

func connectionExists(t *testing.T, repos federationmemory.Repositories, id string) bool {
	t.Helper()
	connection, err := repos.Connections.Find(context.Background(), tenancydomain.DefaultTenantID, id)
	if err != nil {
		t.Fatalf("find connection %s: %v", id, err)
	}
	return connection != nil
}

//spec:covers EX-AUTHENTICATION-037-02: 連携が残る接続の削除は ErrConnectionInUse で拒否され、接続と連携のどちらも保存先に残ること。
func TestDeleteConnectionRefusesAConnectionWithLinkedIdentities(t *testing.T) {
	deps, repos := deleteFixture(t)

	err := federationusecases.DeleteConnection(context.Background(), deps, tenancydomain.DefaultTenantID, "linked")
	if !errors.Is(err, federationusecases.ErrConnectionInUse) {
		t.Fatalf("err = %v, want ErrConnectionInUse", err)
	}
	if !connectionExists(t, repos, "linked") {
		t.Fatal("the refused connection was deleted")
	}
	identity, err := repos.Identities.FindByUserProvider(context.Background(), tenancydomain.DefaultTenantID, "linked", "user-1")
	if err != nil || identity == nil {
		t.Fatalf("the link was lost: identity=%v err=%v", identity, err)
	}
}

//spec:covers EX-AUTHENTICATION-037-01: 連携の無い接続は削除され、同じテナントの別の接続に連携が残っていても削除を妨げないこと。存在しない接続の削除も成功すること。
func TestDeleteConnectionRemovesAConnectionWithoutLinks(t *testing.T) {
	deps, repos := deleteFixture(t)

	if err := federationusecases.DeleteConnection(context.Background(), deps, tenancydomain.DefaultTenantID, "unlinked"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if connectionExists(t, repos, "unlinked") {
		t.Fatal("the connection without links was not deleted")
	}
	if !connectionExists(t, repos, "linked") {
		t.Fatal("deleting one connection removed another")
	}
	if err := federationusecases.DeleteConnection(context.Background(), deps, tenancydomain.DefaultTenantID, "missing"); err != nil {
		t.Fatalf("deleting a missing connection: %v", err)
	}
}
