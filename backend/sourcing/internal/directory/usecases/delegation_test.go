package usecases

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

func delegationFixture(t *testing.T) (context.Context, *Delegation, domain.Connector, map[string]any) {
	t.Helper()
	ctx := testing_tenant.Default(t.Context())
	tenantID := tenantports.TenantID(ctx)
	service := New(db_memory.NewConnectorRepository(), nil, nil)
	grant, err := service.Create(ctx, tenantID, "AD")
	if err != nil {
		t.Fatal(err)
	}
	c, err := service.Enroll(ctx, tenantID, grant.Token, ecJWK)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	point, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": base64.RawURLEncoding.EncodeToString(point[1:33]), "y": base64.RawURLEncoding.EncodeToString(point[33:65])}
	correlations := db_memory.NewCorrelationRepository()
	d := &Delegation{Service: service, Correlations: correlations, Challenges: db_memory.NewPasswordChallengeStore(), Locks: db_memory.NewIngestionRunRepository(), Users: idmanagement.NewMemoryModule().UserRepo, Now: time.Now}
	if err := d.Users.Save(ctx, &userdomain.User{ID: "directory-user", TenantID: tenantID, PreferredUsername: "directory-user", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := correlations.SaveUserCorrelation(ctx, directoryports.UserCorrelation{TenantID: tenantID, ConnectorID: c.ID, UserID: "directory-user", ExternalID: "external-user"}); err != nil {
		t.Fatal(err)
	}
	return ctx, d, c, jwk
}

//spec:covers REQ-SOURCING-027 REQ-AUTHENTICATION-044: 能力とローカル資格情報の検査を通った設定だけが有効になり、停止時にもローカル認証を許さない。
func TestDelegationConfigurationRejectsLocalCredentialsAndRetainsItsAuthority(t *testing.T) {
	ctx, d, c, jwk := delegationFixture(t)
	if _, err := d.Configure(ctx, c.ID, true); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("enable without capability: %v", err)
	}
	if _, err := d.RegisterEncryption(ctx, c, ecJWK); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("signing key reused: %v", err)
	}
	if _, err := d.RegisterEncryption(ctx, c, jwk); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Users.Update(ctx, "directory-user", func(u *userdomain.User) (bool, error) { u.PasswordHash = "local-hash"; return true, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Configure(ctx, c.ID, true); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("enable with local hash: %v", err)
	}
	if _, err := d.Users.Update(ctx, "directory-user", func(u *userdomain.User) (bool, error) { u.PasswordHash = ""; return true, nil }); err != nil {
		t.Fatal(err)
	}
	c, err := d.Configure(ctx, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if local, err := d.AllowsLocalPassword(ctx, "directory-user"); err != nil || local {
		t.Fatalf("local write allowed=%v err=%v", local, err)
	}
	if local, err := d.AllowsLocalPassword(ctx, "unrelated-user"); err != nil || !local {
		t.Fatalf("unrelated user changed=%v err=%v", local, err)
	}
	if _, err := d.Service.Disable(ctx, c.TenantID, c.ID); err != nil {
		t.Fatal(err)
	}
	got, err := d.Verify(ctx, "directory-user", "password")
	if err != nil || !got.Delegated || got.Result != domain.PasswordConnectorUnavailable {
		t.Fatalf("disabled authority=%+v %v", got, err)
	}
	if local, err := d.AllowsLocalPassword(ctx, "directory-user"); err != nil || local {
		t.Fatalf("disabled authority allowed local write=%v %v", local, err)
	}
}

//spec:covers REQ-SOURCING-026: 返答を受理する前に設定の世代を変えると、旧要求で認証できない。
func TestDelegationRejectsAResultFromAnEarlierConfiguration(t *testing.T) {
	ctx, d, c, jwk := delegationFixture(t)
	if _, err := d.RegisterEncryption(ctx, c, jwk); err != nil {
		t.Fatal(err)
	}
	c, err := d.Configure(ctx, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	wait, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan PasswordVerification, 1)
	go func() { result, _ := d.Verify(wait, "directory-user", "password"); done <- result }()
	challenge, err := d.Poll(wait, c)
	if err != nil || challenge == nil {
		t.Fatalf("poll=%+v %v", challenge, err)
	}
	if _, err := d.Configure(ctx, c.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := d.Complete(ctx, c, challenge.ID, domain.PasswordAccepted); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("old generation accepted: %v", err)
	}
	cancel()
	result := <-done
	if !result.Delegated || result.Result != domain.PasswordConnectorUnavailable {
		t.Fatalf("old result=%+v", result)
	}
	if item, err := d.Challenges.Find(ctx, c.TenantID, challenge.ID); err != nil || item != nil {
		t.Fatalf("cancelled request remains=%+v %v", item, err)
	}
}
