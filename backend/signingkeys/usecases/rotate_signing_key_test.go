package usecases

// SCL シナリオ "鍵回転後も旧 kid の JWKS エントリは保持される" を担保する。
// SigningKeyMinJwksOverlap (7d) により、回転後しばらく旧鍵で署名された JWT を
// 検証できる必要がある。

import (
	"context"
	"testing"
	"time"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/adapters/crypto"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRotateSigningKeyKeepsPreviousKidInJWKS(t *testing.T) {
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	prev, err := keyStore.GetActiveKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var emitted []spec.DomainEvent
	deps := RotateSigningKeyDeps{
		KeyStore: keyStore,
		Emit:     func(e spec.DomainEvent) { emitted = append(emitted, e) },
	}
	next, err := RotateSigningKey(context.Background(), deps, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if next.Kid == prev.Kid {
		t.Fatal("rotation must produce a fresh kid")
	}

	// JWKS は旧 kid と新 kid 両方を保持する。
	all, err := keyStore.GetAllKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, k := range all {
		seen[k.Kid] = true
	}
	if !seen[prev.Kid] || !seen[next.Kid] {
		t.Fatalf("JWKS missing kids: %v (want both prev=%s next=%s)", seen, prev.Kid, next.Kid)
	}

	// SigningKeyRotated イベントが旧 / 新 kid を伴って発火する。
	if len(emitted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitted))
	}
	ev, ok := emitted[0].(*signingdomain.SigningKeyRotated)
	if !ok || ev.PreviousKID != prev.Kid || ev.NewKID != next.Kid {
		t.Fatalf("unexpected event: %+v", emitted[0])
	}
}

// SCL objective SigningKeyLifecycle: cadence 未到来の batch 評価は鍵を増やさない。
func TestRotateSigningKeyIfDueSkipsFreshActiveKey(t *testing.T) {
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	rotated, err := RotateSigningKeyIfDue(context.Background(), RotateSigningKeyDeps{
		KeyStore: keyStore,
		Grace:    7 * 24 * time.Hour,
	}, now, 90*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if rotated != nil {
		t.Fatalf("fresh active key must not rotate: %+v", rotated)
	}
}
