package envelope_openbao

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

type fakeTransitEngine struct {
	ensuredKeys map[string]bool
	store       map[string][]byte // keyName -> plaintext of the single stored ciphertext
	healthy     bool
	ensureErr   error
}

func newFakeTransitEngine() *fakeTransitEngine {
	return &fakeTransitEngine{
		ensuredKeys: map[string]bool{},
		store:       map[string][]byte{},
		healthy:     true,
	}
}

func (f *fakeTransitEngine) EnsureKey(_ context.Context, name string) error {
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensuredKeys[name] = true
	return nil
}

func (f *fakeTransitEngine) EncryptDataKey(_ context.Context, name string, plaintext []byte) (string, error) {
	if !f.ensuredKeys[name] {
		return "", errors.New("fake transit: key not ensured")
	}
	f.store[name] = plaintext
	return "vault:v1:" + base64.StdEncoding.EncodeToString(plaintext), nil
}

func (f *fakeTransitEngine) DecryptDataKey(_ context.Context, name, ciphertext string) ([]byte, error) {
	stored, ok := f.store[name]
	if !ok {
		return nil, errors.New("fake transit: nothing stored for key")
	}
	want := "vault:v1:" + base64.StdEncoding.EncodeToString(stored)
	if ciphertext != want {
		return nil, errors.New("fake transit: ciphertext does not match stored value")
	}
	return stored, nil
}

func (f *fakeTransitEngine) Healthy(_ context.Context) bool {
	return f.healthy
}

// TestWrapUnwrapRoundTrip covers scenario
// "テナント初回利用時にDEKがbootstrapされる" (docs/modules/data-keys/) for the
// OpenBao provider: WrapDataKey ensures a per-tenant transit key exists and
// UnwrapDataKey recovers the original plaintext.
func TestWrapUnwrapRoundTrip(t *testing.T) {
	ctx := context.Background()
	engine := newFakeTransitEngine()
	provider := NewOpenBaoMasterKeyProvider(engine, DefaultKeyNamePrefix)

	plaintextDEK := []byte("0123456789abcdef0123456789abcdef")
	wrapped, masterKeyID, err := provider.WrapDataKey(ctx, "tenant-a", plaintextDEK)
	if err != nil {
		t.Fatalf("WrapDataKey failed: %v", err)
	}
	if masterKeyID != "idmagic-datakeys-tenant-a" {
		t.Fatalf("unexpected master key id: %s", masterKeyID)
	}
	if !engine.ensuredKeys[masterKeyID] {
		t.Fatal("expected WrapDataKey to ensure the per-tenant transit key exists")
	}

	unwrapped, err := provider.UnwrapDataKey(ctx, "tenant-a", wrapped, masterKeyID)
	if err != nil {
		t.Fatalf("UnwrapDataKey failed: %v", err)
	}
	if !bytes.Equal(unwrapped, plaintextDEK) {
		t.Fatal("UnwrapDataKey did not return the original plaintext DEK")
	}
}

// TestUnwrapRejectsMismatchedMasterKeyID ensures a masterKeyID referencing a
// different tenant's transit key is rejected fail-closed, even if
// the caller supplies otherwise-valid ciphertext.
func TestUnwrapRejectsMismatchedMasterKeyID(t *testing.T) {
	ctx := context.Background()
	engine := newFakeTransitEngine()
	provider := NewOpenBaoMasterKeyProvider(engine, DefaultKeyNamePrefix)

	wrapped, _, err := provider.WrapDataKey(ctx, "tenant-a", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("WrapDataKey failed: %v", err)
	}

	if _, err := provider.UnwrapDataKey(ctx, "tenant-b", wrapped, "idmagic-datakeys-tenant-a"); err == nil {
		t.Fatal("expected UnwrapDataKey to fail-closed for a masterKeyID/tenant mismatch")
	} else if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("expected ErrDataKeyUnavailable, got %v", err)
	}
}

// TestWrapPropagatesProviderUnavailable ensures a transit engine error (e.g.
// OpenBao unreachable) surfaces as an error rather than a silent fallback
// (fail-closed on provider outage).
func TestWrapPropagatesProviderUnavailable(t *testing.T) {
	ctx := context.Background()
	engine := newFakeTransitEngine()
	engine.ensureErr = errors.New("connection refused")
	provider := NewOpenBaoMasterKeyProvider(engine, DefaultKeyNamePrefix)

	if _, _, err := provider.WrapDataKey(ctx, "tenant-a", []byte("0123456789abcdef0123456789abcdef")); err == nil {
		t.Fatal("expected WrapDataKey to fail when the transit engine is unavailable")
	}
}

func TestHealthyDelegatesToEngine(t *testing.T) {
	engine := newFakeTransitEngine()
	provider := NewOpenBaoMasterKeyProvider(engine, DefaultKeyNamePrefix)

	if !provider.Healthy(context.Background()) {
		t.Fatal("expected Healthy to be true when engine is healthy")
	}
	engine.healthy = false
	if provider.Healthy(context.Background()) {
		t.Fatal("expected Healthy to be false when engine is unhealthy")
	}
}

// 接頭辞は Transit の鍵名の規則に合うものだけを受け付ける。OpenBao の記録で受け付けた鍵名の接頭辞も受け付ける。
func TestValidKeyNamePrefix(t *testing.T) {
	tests := []struct {
		prefix string
		want   bool
	}{
		{DefaultKeyNamePrefix, true},
		{"idmagic_datakeys.v2@prod", true},
		{"idmagic/datakeys", false},
		{"", false},
		{"-datakeys", false},
		{"data keys", false},
	}
	for _, tc := range tests {
		if got := ValidKeyNamePrefix(tc.prefix); got != tc.want {
			t.Errorf("ValidKeyNamePrefix(%q) = %v, want %v", tc.prefix, got, tc.want)
		}
	}
}

func TestImplementsMasterKeyProviderPort(t *testing.T) {
	var _ envelope_crypto.MasterKeyProvider = (*OpenBaoMasterKeyProvider)(nil)
}
