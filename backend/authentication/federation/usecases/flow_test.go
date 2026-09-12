package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type protocolDriverStub struct {
	claims      federationdomain.NormalizedClaims
	startErr    error
	completeErr error
}

func (d protocolDriverStub) Start(
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	return "https://idp.example/authorize", d.startErr
}

func (d protocolDriverStub) Complete(
	_ context.Context,
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ string,
	_ time.Time,
) (federationdomain.NormalizedClaims, error) {
	return d.claims, d.completeErr
}

func TestStartAndCompleteFlowConsumesStateBeforeProtocolValidation(t *testing.T) {
	deps, connection, _, repos := brokerFixture(t)
	deps.Connections = repos.Connections
	deps.Attempts = repos.Attempts
	deps.Drivers = map[federationdomain.Protocol]federationusecases.ProtocolDriver{
		federationdomain.ProtocolOIDC: protocolDriverStub{completeErr: errors.New("invalid token")},
	}
	if err := repos.Connections.Save(context.Background(), &connection); err != nil {
		t.Fatal(err)
	}
	start, err := federationusecases.StartLogin(
		context.Background(), deps, connection.ID, "", "", "https://broker.example/callback", time.Now(),
	)
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	if start.RedirectTo == "" || start.State == "" {
		t.Fatalf("start=%+v", start)
	}
	if _, err := federationusecases.CompleteLogin(
		context.Background(), deps, start.State, "response", "https://broker.example/callback", time.Now(),
	); err == nil {
		t.Fatal("invalid protocol response must fail")
	}
	if _, err := federationusecases.CompleteLogin(
		context.Background(), deps, start.State, "response", "https://broker.example/callback", time.Now(),
	); err == nil {
		t.Fatal("consumed state must not be reusable")
	}
}

// countingDriver は Complete が呼ばれた回数を数える。state の照合が拒否したときに、
// upstream の応答が検証まで到達していないことを観測するためである。
type countingDriver struct {
	claims    federationdomain.NormalizedClaims
	completed int
}

func (d *countingDriver) Start(
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	return "https://idp.example/authorize", nil
}

func (d *countingDriver) Complete(
	_ context.Context,
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ string,
	_ time.Time,
) (federationdomain.NormalizedClaims, error) {
	d.completed++
	return d.claims, nil
}

// 固定する。観測は「値が返ってくること」ではなく「違う値では認可が成立しないこと」で
// ある。攻撃者が仕込んだ callback — 発行されていない state — が拒否されること、同じ
// state の 2 度目が拒否されること、そしてどちらの拒否でもセッションが発行されず upstream
// の応答が検証にすら到達しないこと (拒否が防いだ効果) を観測する。error の戻り値だけを
// 見ると、セッションを作ってから error を返す実装と区別が付かない。
// 正当な state が同じ条件で成立することを最後に置くのは、拒否がすべて別の理由 (設定漏れ
// など) で起きていた場合にそれを検出するためである。
//
//spec:covers OIDC-CORE-CSRF: callback が、login attempt に束縛された単発の state を照合することを
func TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	deps, connection, users, repos := brokerFixture(t)
	deps.Connections, deps.Attempts = repos.Connections, repos.Attempts
	// セッションストアは fixture の内側にあるので、発行されたセッションを数えられるよう
	// テストが握るものへ差し替える。
	sessions := sessionmemory.NewSessionStore()
	deps.Sessions = sessionusecases.NewSessionManager(sessions)
	driver := &countingDriver{claims: federationdomain.NormalizedClaims{
		Subject: "external", Username: "existing@example.com",
	}}
	deps.Drivers = map[federationdomain.Protocol]federationusecases.ProtocolDriver{
		federationdomain.ProtocolOIDC: driver,
	}
	if err := repos.Connections.Save(ctx, &connection); err != nil {
		t.Fatal(err)
	}
	user := activeUser("user-1", "existing@example.com", now)
	users.Seed(user)
	if err := repos.Identities.Create(ctx, &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID,
		ExternalSubject: "external", LocalUserID: user.ID, LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	sessionCount := func(t *testing.T) int {
		t.Helper()
		issued, err := sessions.ListBySub(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(issued)
	}

	start, err := federationusecases.StartLogin(ctx, deps, connection.ID, "", "", "https://broker.example/callback", now)
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	// 不一致: 発行されていない state を持つ callback。
	if _, err := federationusecases.CompleteLogin(
		ctx, deps, "state-the-broker-never-issued", "response", "https://broker.example/callback", now,
	); err == nil {
		t.Fatal("a state the broker never issued completed the login")
	}
	if driver.completed != 0 {
		t.Fatalf("upstream response validated %d time(s) for an unknown state", driver.completed)
	}
	if got := sessionCount(t); got != 0 {
		t.Fatalf("sessions=%d after an unknown state; want none", got)
	}

	// 正当な state は成立する。ここまでの拒否が state の照合によるものであることの対照。
	completion, err := federationusecases.CompleteLogin(
		ctx, deps, start.State, "response", "https://broker.example/callback", now,
	)
	if err != nil {
		t.Fatalf("the issued state was rejected: %v", err)
	}
	if completion.Authentication.SessionID == "" {
		t.Fatalf("completion=%+v", completion)
	}
	if got := sessionCount(t); got != 1 {
		t.Fatalf("sessions=%d after the legitimate callback; want 1", got)
	}

	// 単発: 同じ state の 2 度目。
	if _, err := federationusecases.CompleteLogin(
		ctx, deps, start.State, "response", "https://broker.example/callback", now,
	); err == nil {
		t.Fatal("a consumed state completed the login a second time")
	}
	if driver.completed != 1 {
		t.Fatalf("upstream response validated %d time(s); the replayed state reached it", driver.completed)
	}
	if got := sessionCount(t); got != 1 {
		t.Fatalf("sessions=%d after the replay; want the 1 from the legitimate callback", got)
	}
}
