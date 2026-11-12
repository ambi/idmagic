package db_postgres

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-SOURCING-026 REQ-SOURCING-027: 暗号化能力と委譲設定を保存して読み直し、DB から配信した暗号文の束縛を検証してから期限内の結果を認証へ返す。
func TestPostgresDelegationPersistsTheCapabilityAndCompletesTheBoundChallenge(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := tenantports.WithTenant(t.Context(), tenant, "", "")
	users := idmanagement.NewPostgresModule(db).UserRepo
	user := pgfixtures.SeedUser(t, users, tenant.ID)
	if _, err := users.Update(ctx, user.ID, func(u *userdomain.User) (bool, error) { u.PasswordHash = ""; return true, nil }); err != nil {
		t.Fatal(err)
	}
	service := usecases.New(&ConnectorRepository{Pool: db}, nil, nil)
	grant, err := service.Create(ctx, tenant.ID, "AD")
	if err != nil {
		t.Fatal(err)
	}
	signing := map[string]any{"kty": "EC", "crv": "P-256", "x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU", "y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"}
	c, err := service.Enroll(ctx, tenant.ID, grant.Token, signing)
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
	correlations := &CorrelationRepository{Pool: db}
	store := &PasswordChallengeStore{Pool: db}
	if err := correlations.SaveUserCorrelation(ctx, directoryports.UserCorrelation{TenantID: tenant.ID, ConnectorID: c.ID, ExternalID: "directory-identity", UserID: user.ID, LastSeenRunID: uuid.NewV4().String()}); err != nil {
		t.Fatal(err)
	}
	d := &usecases.Delegation{Service: service, Correlations: correlations, Challenges: store, Locks: &IngestionRunRepository{Pool: db}, Users: users, Now: time.Now}
	if _, err := d.RegisterEncryption(ctx, c, jwk); err != nil {
		t.Fatal(err)
	}
	c, err = d.Configure(ctx, c.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !c.DelegatedAuthentication || c.EncryptionKeyID == "" || c.EncryptionPublicKey == nil {
		t.Fatalf("persisted capability=%+v", c)
	}
	done := make(chan usecases.PasswordVerification, 1)
	go func() { got, _ := d.Verify(ctx, user.ID, "directory-password"); done <- got }()
	challenge, err := d.Poll(ctx, c)
	if err != nil || challenge == nil {
		t.Fatalf("poll=%+v %v", challenge, err)
	}
	plain, err := tokens_jose.DecryptCompact(key, c.EncryptionKeyID, challenge.EncryptedRequest)
	if err != nil {
		t.Fatal(err)
	}
	var payload domain.PasswordPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Password != "directory-password" || payload.ID != challenge.ID || payload.TenantID != tenant.ID || !payload.ExpiresAt.Equal(challenge.ExpiresAt) {
		t.Fatal("database transport changed the encrypted binding")
	}
	if err := d.Complete(ctx, c, challenge.ID, domain.PasswordAccepted); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if !got.Delegated || got.Result != domain.PasswordAccepted {
			t.Fatalf("result=%+v", got)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("verification did not finish")
	}
	if err := d.Complete(ctx, c, challenge.ID, domain.PasswordAccepted); err != nil {
		t.Fatalf("idempotent result=%v", err)
	}
	stored, err := store.Find(ctx, tenant.ID, challenge.ID)
	if err != nil || stored == nil || stored.EncryptedRequest != "" {
		t.Fatalf("stored ciphertext=%+v %v", stored, err)
	}
}
