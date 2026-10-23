package keys_vault

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keysJOSE "github.com/ambi/idmagic/backend/signingkeys/internal/keys_jose"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func (s *VaultKeyStore) stateKeys(ctx context.Context, state KeySetState) ([]*signingdomain.SigningKey, error) {
	keys := make([]*signingdomain.SigningKey, 0, len(state.Records))
	for _, record := range state.Records {
		key := record.Key
		cert, err := x509.ParseCertificate(key.CertificateDER)
		if err != nil {
			return nil, err
		}
		key.PublicKey = cert.PublicKey
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok || record.Version < 1 {
			return nil, errors.New("invalid Vault XML public key metadata")
		}
		key.PrivateKey = vaultSigner{engine: s.engine, name: s.keyName(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx)), version: record.Version, pub: pub}
		keys = append(keys, &key)
	}
	return keys, nil
}

func (s *VaultKeyStore) xmlKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	state, err := s.states.Load(ctx)
	if err != nil {
		return nil, err
	}
	return s.stateKeys(ctx, state)
}

func (s *VaultKeyStore) xmlActive(ctx context.Context) (*signingdomain.SigningKey, error) {
	keys, err := s.xmlKeys(ctx)
	if err != nil {
		return nil, err
	}
	if active := signingdomain.ActiveKey(keys); active != nil {
		return active, nil
	}
	result, err := s.changeXML(ctx, "initialize", "", time.Now().UTC(), 0)
	return result.Next, err
}

func (s *VaultKeyStore) Prepare(ctx context.Context, now time.Time) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	if _, err := s.xmlActive(ctx); err != nil {
		return signingdomain.RotationResult{}, err
	}
	return s.changeXML(ctx, "prepare", "", now, 0)
}

func (s *VaultKeyStore) Activate(ctx context.Context, kid string, now time.Time, grace time.Duration) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	return s.changeXML(ctx, "activate", kid, now, grace)
}

func (s *VaultKeyStore) changeXML(ctx context.Context, action, kid string, now time.Time, grace time.Duration) (signingdomain.RotationResult, error) {
	var result signingdomain.RotationResult
	if grace < 0 {
		return result, errors.New("signing key grace period must not be negative")
	}
	err := s.states.Update(ctx, func(state *KeySetState) error {
		keys, err := s.stateKeys(ctx, *state)
		if err != nil {
			return err
		}
		active := signingdomain.ActiveKey(keys)
		if action == "initialize" && active != nil {
			result.Next = active
			return nil
		}
		if action == "prepare" {
			if key := signingdomain.PreparedKey(keys); key != nil {
				result = signingdomain.RotationResult{Next: key, Previous: active}
				return nil
			}
		}
		if action == "activate" {
			result, err = signingdomain.ActivatePrepared(keys, kid, now, grace)
			if err != nil {
				return err
			}
		} else {
			if action == "rotate" && active != nil && (signingdomain.RequiresPrepublication(signingports.KeyUsage(ctx), signingports.KeyScope(ctx)) || signingdomain.PreparedKey(keys) != nil) {
				return signingdomain.ErrPrepublicationRequired
			}
			if !s.engine.Healthy(ctx) {
				return errors.New("vault key provider unavailable")
			}
			name := s.keyName(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx))
			if err := s.engine.EnsureKey(ctx, name); err != nil {
				return err
			}
			if active != nil {
				if err := s.engine.RotateKey(ctx, name); err != nil {
					return err
				}
			}
			publicPEM, version, err := s.engine.LatestPublicKey(ctx, name)
			if err != nil {
				return err
			}
			pub, err := parseRSAPublicKeyPEM(publicPEM)
			if err != nil {
				return err
			}
			jwk := keysJOSE.PublicJWK(pub)
			newKid, err := keysJOSE.Thumbprint(jwk)
			if err != nil {
				return err
			}
			jwk["kid"], jwk["alg"], jwk["use"] = newKid, string(signingdomain.SigAlgPS256), "sig"
			signer := vaultSigner{engine: s.engine, name: name, version: version, pub: pub}
			certificate, err := keysJOSE.NewFederationCertificate(tenantports.TenantID(ctx), newKid, signer, now)
			if err != nil {
				return err
			}
			key := &signingdomain.SigningKey{TenantID: tenantports.TenantID(ctx), Kid: newKid, Usage: signingports.KeyUsage(ctx), ScopeID: signingports.KeyScope(ctx), Alg: signingdomain.SigAlgPS256, Provider: signingdomain.KeyProviderVaultTransit, Prepared: true, CreatedAt: now, PublicJWK: jwk, PublicKey: pub, PrivateKey: signer, CertificateDER: certificate}
			keys = append(keys, key)
			state.Records = append(state.Records, KeyRecord{Key: *key, Version: version})
			if action == "prepare" {
				result = signingdomain.RotationResult{Next: key, Previous: active, Changed: true}
			} else {
				result, err = signingdomain.ActivatePrepared(keys, newKid, now, grace)
				if err != nil {
					return err
				}
			}
		}
		for i, key := range keys {
			state.Records[i].Key = *key
		}
		return nil
	})
	return result, err
}

func (s *VaultKeyStore) disableXML(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	var disabled *signingdomain.SigningKey
	err := s.states.Update(ctx, func(state *KeySetState) error {
		for i := range state.Records {
			key := &state.Records[i].Key
			if key.Kid != kid {
				continue
			}
			if key.Active {
				return signingdomain.ErrActiveSigningKeyCannotBeDisabled
			}
			now := time.Now().UTC()
			key.Prepared = false
			key.ArchivedAt = &now
			snapshot := *key
			disabled = &snapshot
		}
		return nil
	})
	return disabled, err
}

func (s *VaultKeyStore) archiveXML(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	var archived []*signingdomain.SigningKey
	err := s.states.Update(ctx, func(state *KeySetState) error {
		for i := range state.Records {
			key := &state.Records[i].Key
			if key.ArchivedAt == nil && key.ExpiresAt != nil && !key.ExpiresAt.After(before) {
				at := before
				key.ArchivedAt = &at
				snapshot := *key
				archived = append(archived, &snapshot)
			}
		}
		return nil
	})
	return archived, err
}
