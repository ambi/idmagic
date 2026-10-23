// Package signers_saml は、SigningKeys の XML 連携署名の鍵から SAML assertion の署名器を解決する。
package signers_saml

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

// KeyStoreSignerProvider は、要求の鍵のスコープで有効な XML 連携署名の鍵から署名器を作る。
type KeyStoreSignerProvider struct {
	KeyStore signingports.KeyStore
}

var _ tokens_saml.SignerProvider = KeyStoreSignerProvider{}

func (p KeyStoreSignerProvider) PreparedCertificate(ctx context.Context) (*x509.Certificate, error) {
	if p.KeyStore == nil {
		return nil, fmt.Errorf("samltoken: federation key store is required")
	}
	keys, err := p.KeyStore.GetAllKeys(p.xmlContext(ctx))
	if err != nil {
		return nil, err
	}
	key := signingdomain.PreparedKey(keys)
	if key == nil {
		return nil, nil //nolint:nilnil // 準備済みの鍵がないことを表す。
	}
	return x509.ParseCertificate(key.CertificateDER)
}

// xmlContext は、XML 連携署名の鍵の用途と、WithSignerScope で載せた鍵のスコープを文脈に付ける。
func (p KeyStoreSignerProvider) xmlContext(ctx context.Context) context.Context {
	if scopeID, ok := tokens_saml.SignerScope(ctx); ok {
		ctx = signingports.WithKeyScope(ctx, scopeID)
	}
	return signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
}

func (p KeyStoreSignerProvider) Resolve(ctx context.Context) (*tokens_saml.Signer, error) {
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
	return tokens_saml.NewSigner(cert, signer)
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
