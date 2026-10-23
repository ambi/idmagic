package keys_vault_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingpostgres "github.com/ambi/idmagic/backend/signingkeys/internal/db_postgres"
	adaptercrypto "github.com/ambi/idmagic/backend/signingkeys/internal/keys_vault"
	contract "github.com/ambi/idmagic/backend/signingkeys/internal/testing_contract"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

//spec:covers REQ-SIGNINGKEYS-007: PostgreSQL の公開状態から再構築した Vault ストアが、最新バージョンを署名用にせず、有効と準備済みの選択を共有すること。
func TestVaultPreparedSelectionSurvivesRestart(t *testing.T) {
	db := pgtest.Require(t)
	now := pgtest.Now()
	tenant := &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	ctx := signingports.WithKeyUsage(tenantports.WithTenant(context.Background(), tenant, "", ""), signingdomain.KeyUsageXMLFederationSigning)
	engine := newFakeTransit()
	newStore := func() *adaptercrypto.VaultKeyStore {
		return adaptercrypto.NewVaultKeyStoreWithState(engine, "idmagic-signing-", signingpostgres.VaultKeyStateStore{Pool: db})
	}
	first := newStore()
	old, err := first.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := first.Prepare(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Changed || prepared.Next.Active {
		t.Fatalf("prepare: %+v", prepared)
	}
	restarted := newStore()
	current, err := restarted.GetActiveKey(ctx)
	if err != nil || current.Kid != old.Kid {
		t.Fatalf("restart selected new key: %v %+v", err, current)
	}
	state, err := (signingpostgres.VaultKeyStateStore{Pool: db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range state.Records {
		if record.Key.PrivateKey != nil {
			t.Fatal("Vault private key escaped into PostgreSQL")
		}
	}
	again, err := restarted.Prepare(ctx, now.Add(time.Hour))
	if err != nil || again.Changed || again.Next.Kid != prepared.Next.Kid {
		t.Fatalf("restart lost prepared key: %v %+v", err, again)
	}
	activated, err := restarted.Activate(ctx, prepared.Next.Kid, now.Add(24*time.Hour), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	current, err = first.GetActiveKey(ctx)
	if err != nil || current.Kid != activated.Next.Kid {
		t.Fatal("another process retained the old selection")
	}
	digest := sha256.Sum256([]byte("XML signature after restart"))
	signature, err := current.PrivateKey.(crypto.Signer).Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if err := rsa.VerifyPKCS1v15(current.PublicKey.(*rsa.PublicKey), crypto.SHA256, digest[:], signature); err != nil {
		t.Fatal(err)
	}
}

//spec:covers REQ-SIGNINGKEYS-006: Vault の準備と切り替えはメモリと PostgreSQL の提供元と同じ共有契約を満たすこと。
func TestVaultPreparedKeyContract(t *testing.T) {
	contract.Run(t, func(t *testing.T) contract.Fixture {
		t.Helper()
		return contract.Fixture{Store: adaptercrypto.NewVaultKeyStore(newFakeTransit(), "idmagic-signing-"), Context: tenantCtx()}
	})
}

type failingStateStore struct {
	adaptercrypto.KeyStateStore
	fail bool
}

func (s *failingStateStore) Update(ctx context.Context, update func(*adaptercrypto.KeySetState) error) error {
	return s.KeyStateStore.Update(ctx, func(state *adaptercrypto.KeySetState) error {
		if err := update(state); err != nil {
			return err
		}
		if s.fail {
			return errors.New("state persistence failed")
		}
		return nil
	})
}

//spec:covers REQ-SIGNINGKEYS-007: Vault のバージョンを作った後に状態の保存が失敗しても、有効な選択を保つこと。
func TestVaultPreparationPersistenceFailureKeepsTheSigningKey(t *testing.T) {
	state := &failingStateStore{KeyStateStore: adaptercrypto.NewMemoryKeyStateStore()}
	engine := newFakeTransit()
	ctx := signingports.WithKeyUsage(tenantCtx(), signingdomain.KeyUsageXMLFederationSigning)
	store := adaptercrypto.NewVaultKeyStoreWithState(engine, "idmagic-signing-", state)
	old, err := store.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state.fail = true
	if _, err := store.Prepare(ctx, time.Now().UTC()); err == nil {
		t.Fatal("persist failure was ignored")
	}
	state.fail = false
	current, err := adaptercrypto.NewVaultKeyStoreWithState(engine, "idmagic-signing-", state).GetActiveKey(ctx)
	if err != nil || current.Kid != old.Kid {
		t.Fatal("orphaned Vault version became active")
	}
}
