package keys_jose

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

// NewFederationCertificate creates the public X.509 wrapper used by SAML and WS-* metadata.
func NewFederationCertificate(
	tenantID, kid string,
	signer crypto.Signer,
	_ time.Time,
) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("federation certificate signer is required")
	}
	sum := sha256.Sum256([]byte(tenantID + "\x00" + kid))
	serial := new(big.Int).SetBytes(sum[:20])
	// The certificate wrapper must be reproducible from a durable Vault key after a
	// process restart. Rotation, not PKIX expiry, governs federation trust lifetime.
	notBefore := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "idmagic " + tenantID + " federation signing"},
		NotBefore:             notBefore,
		NotAfter:              notBefore.AddDate(30, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		return nil, fmt.Errorf("create federation certificate: %w", err)
	}
	return der, nil
}
