package signers_saml

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_memory"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestKeyStoreSignerProviderResolvesTenantXMLCredential(t *testing.T) {
	store, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	provider := KeyStoreSignerProvider{KeyStore: store}
	ctxA := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	ctxB := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
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
	profileSigner, err := provider.Resolve(samltoken.WithSignerScope(ctxA, "profile-a"))
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

type unavailableKeyList struct{ signingports.KeyStore }

func (unavailableKeyList) GetAllKeys(context.Context) ([]*signingdomain.SigningKey, error) {
	return nil, errors.New("key list unavailable")
}

//spec:covers REQ-SIGNINGKEYS-006: 準備済みの証明書は有効な証明書と分け、要求のプロファイルの範囲で読むこと。
func TestPreparedCertificateUsesTheRequestedScope(t *testing.T) {
	store, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	provider := KeyStoreSignerProvider{KeyStore: store}
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	if cert, err := provider.PreparedCertificate(ctx); err != nil || cert != nil {
		t.Fatalf("unexpected candidate: %v %v", cert, err)
	}
	profile := samltoken.WithSignerScope(ctx, "profile-a")
	prepared, err := store.Prepare(provider.xmlContext(profile), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	cert, err := provider.PreparedCertificate(profile)
	if err != nil || cert == nil || !bytes.Equal(cert.Raw, prepared.Next.CertificateDER) {
		t.Fatalf("prepared certificate: %v %v", cert, err)
	}
	if cert, err := provider.PreparedCertificate(ctx); err != nil || cert != nil {
		t.Fatal("profile candidate escaped to the shared scope")
	}
	if _, err := (KeyStoreSignerProvider{KeyStore: unavailableKeyList{}}).PreparedCertificate(ctx); err == nil {
		t.Fatal("failed key read became an empty certificate")
	}
	if _, err := (KeyStoreSignerProvider{}).PreparedCertificate(ctx); err == nil {
		t.Fatal("missing key store was ignored")
	}
}
