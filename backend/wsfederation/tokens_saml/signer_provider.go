package tokens_saml

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

type SignerProvider interface {
	Resolve(ctx context.Context) (*Signer, error)
	Certificates(ctx context.Context, now time.Time) ([]*x509.Certificate, error)
}

type KeyStoreSignerProvider struct {
	KeyStore signingports.KeyStore
}

func WithSignerScope(ctx context.Context, scopeID string) context.Context {
	return signingports.WithKeyScope(ctx, scopeID)
}

func (p KeyStoreSignerProvider) xmlContext(ctx context.Context) context.Context {
	return signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
}

func (p KeyStoreSignerProvider) Resolve(ctx context.Context) (*Signer, error) {
	if p.KeyStore == nil || !p.KeyStore.Healthy(ctx) {
		return nil, fmt.Errorf("samltoken: federation key provider unavailable")
	}
	key, err := p.KeyStore.GetActiveKey(p.xmlContext(ctx))
	if err != nil {
		return nil, err
	}
	if key == nil || len(key.CertificateDER) == 0 {
		return nil, fmt.Errorf("samltoken: active federation credential unavailable")
	}
	cert, err := x509.ParseCertificate(key.CertificateDER)
	if err != nil {
		return nil, fmt.Errorf("samltoken: parse federation certificate: %w", err)
	}
	signer, ok := key.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("samltoken: federation private key cannot sign")
	}
	return NewSigner(cert, signer)
}

func (p KeyStoreSignerProvider) Certificates(ctx context.Context, now time.Time) ([]*x509.Certificate, error) {
	if p.KeyStore == nil {
		return nil, fmt.Errorf("samltoken: federation key store is required")
	}
	keys, err := p.KeyStore.ListPublicKeys(p.xmlContext(ctx), now)
	if err != nil {
		return nil, err
	}
	out := make([]*x509.Certificate, 0, len(keys))
	for _, key := range keys {
		if len(key.CertificateDER) == 0 {
			return nil, fmt.Errorf("samltoken: federation credential %s has no certificate", key.Kid)
		}
		cert, err := x509.ParseCertificate(key.CertificateDER)
		if err != nil {
			return nil, fmt.Errorf("samltoken: parse federation certificate %s: %w", key.Kid, err)
		}
		out = append(out, cert)
	}
	if len(out) == 0 {
		if _, err := p.Resolve(ctx); err != nil {
			return nil, err
		}
		return p.Certificates(ctx, now)
	}
	return out, nil
}
