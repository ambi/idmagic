package tokens_saml

import (
	"context"
	"crypto/x509"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestKeyStoreSignerProviderResolvesTenantXMLCredential(t *testing.T) {
	store, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	provider := KeyStoreSignerProvider{KeyStore: store}
	ctxA := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	ctxB := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	signerA, err := provider.Resolve(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	signerB, err := provider.Resolve(ctxB)
	if err != nil {
		t.Fatal(err)
	}
	if signerA.Certificate().SerialNumber.Cmp(signerB.Certificate().SerialNumber) == 0 {
		t.Fatal("different tenants must not resolve the same federation certificate")
	}
	profileSigner, err := provider.Resolve(WithSignerScope(ctxA, "profile-a"))
	if err != nil {
		t.Fatal(err)
	}
	if signerA.Certificate().SerialNumber.Cmp(profileSigner.Certificate().SerialNumber) == 0 {
		t.Fatal("different profiles must not resolve the same federation certificate")
	}
	certs, err := provider.Certificates(ctxA, time.Now().UTC())
	if err != nil || len(certs) != 1 {
		t.Fatalf("certificates: err=%v count=%d", err, len(certs))
	}
	if _, err := x509.ParseCertificate(certs[0].Raw); err != nil {
		t.Fatal(err)
	}
}
