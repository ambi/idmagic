package usecases

import (
	"context"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type staticIntrospector struct {
	result *ports.IntrospectionResult
}

type recordingManagedRevoker struct{ jti string }

func (r *recordingManagedRevoker) RevokeByJTI(_ context.Context, _, jti string, _ time.Time) error {
	r.jti = jti
	return nil
}

func (s staticIntrospector) IntrospectAccessToken(
	context.Context,
	string,
) (*ports.IntrospectionResult, error) {
	return s.result, nil
}

func TestRevokeAccessTokenAddsOwnedJTIToDenylist(t *testing.T) {
	ctx := context.Background()
	denylist := memory.NewAccessTokenDenylist()
	expiresAt := time.Now().Add(time.Minute)
	err := RevokeToken(ctx, RevokeDeps{
		RefreshStore: memory.NewRefreshTokenStore(),
		Introspector: staticIntrospector{result: &ports.IntrospectionResult{
			Active: true, JTI: "jti-1", ClientID: "client", Exp: expiresAt.Unix(),
		}},
		AccessTokenDenylist: denylist,
	}, "client", "header.payload.signature", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := denylist.IsRevoked(ctx, "jti-1")
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("access token jti was not denylisted")
	}
}

func TestRevokeManagedAccessTokenUpdatesLifecycleRecord(t *testing.T) {
	revoker := &recordingManagedRevoker{}
	err := RevokeToken(context.Background(), RevokeDeps{
		RefreshStore:        memory.NewRefreshTokenStore(),
		Introspector:        staticIntrospector{result: &ports.IntrospectionResult{Active: true, Managed: true, JTI: "managed-jti", ClientID: "idmagic-api-token", Exp: time.Now().Add(time.Hour).Unix()}},
		AccessTokenDenylist: memory.NewAccessTokenDenylist(), ManagedTokenRevoker: revoker,
	}, "idmagic-api-token", "jwt", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if revoker.jti != "managed-jti" {
		t.Fatalf("revoked jti = %q", revoker.jti)
	}
}

// ADR-127 §3: session revoke は sid を共有する全 family/client の RefreshTokenRecord を
// 一括で失効させる。
func TestRevokeTokensBySidRevokesAllFamiliesAndClients(t *testing.T) {
	ctx := context.Background()
	store := memory.NewRefreshTokenStore()
	sid := "session-1"
	otherSid := "session-2"

	genA, err := domain.GenerateInitialRefreshToken("client-a", "user-1", []string{"openid"}, nil, &sid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, genA.Record); err != nil {
		t.Fatal(err)
	}
	genB, err := domain.GenerateInitialRefreshToken("client-b", "user-1", []string{"openid"}, nil, &sid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, genB.Record); err != nil {
		t.Fatal(err)
	}
	genOther, err := domain.GenerateInitialRefreshToken("client-a", "user-2", []string{"openid"}, nil, &otherSid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, genOther.Record); err != nil {
		t.Fatal(err)
	}

	if err := RevokeTokensBySid(ctx, RevokeDeps{RefreshStore: store}, sid, time.Now()); err != nil {
		t.Fatal(err)
	}

	recA, err := store.FindByHash(ctx, domain.HashRefreshToken(genA.Token))
	if err != nil {
		t.Fatal(err)
	}
	if recA == nil || !recA.Revoked {
		t.Fatal("client-a token was not revoked")
	}
	recB, err := store.FindByHash(ctx, domain.HashRefreshToken(genB.Token))
	if err != nil {
		t.Fatal(err)
	}
	if recB == nil || !recB.Revoked {
		t.Fatal("client-b token was not revoked")
	}
	recOther, err := store.FindByHash(ctx, domain.HashRefreshToken(genOther.Token))
	if err != nil {
		t.Fatal(err)
	}
	if recOther == nil || recOther.Revoked {
		t.Fatal("token belonging to a different sid must not be revoked")
	}

	// idempotent: 2 回目の呼び出しもエラーなく成功する。
	if err := RevokeTokensBySid(ctx, RevokeDeps{RefreshStore: store}, sid, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeAccessTokenIgnoresOtherClient(t *testing.T) {
	ctx := context.Background()
	denylist := memory.NewAccessTokenDenylist()
	err := RevokeToken(ctx, RevokeDeps{
		RefreshStore: memory.NewRefreshTokenStore(),
		Introspector: staticIntrospector{result: &ports.IntrospectionResult{
			Active: true, JTI: "jti-1", ClientID: "owner", Exp: time.Now().Add(time.Minute).Unix(),
		}},
		AccessTokenDenylist: denylist,
	}, "attacker", "header.payload.signature", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := denylist.IsRevoked(ctx, "jti-1")
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Fatal("another client's access token was revoked")
	}
}
