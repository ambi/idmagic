package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
)

type SigningKeyLifecycleDeps struct {
	KeyStore ports.KeyStore
	Emit     func(spec.DomainEvent)
}

// RunSigningKeyLifecycle は ctx のテナントに定期実行のバッチを 1 回適用する。Signing の鍵は周期で
// ローテーションし、XML 鍵は鍵を持つスコープごとに次の鍵を準備するだけで有効にしない。連携先が
// 新しい証明書を登録したかを、バッチは確かめられないからである。
func RunSigningKeyLifecycle(ctx context.Context, deps SigningKeyLifecycleDeps, now time.Time, cadence, grace time.Duration) error {
	signing := ports.WithKeyUsage(ctx, signingdomain.KeyUsageSigning)
	if _, err := RotateSigningKeyIfDue(signing, RotateSigningKeyDeps{KeyStore: deps.KeyStore, Emit: deps.Emit, Grace: grace}, now, cadence); err != nil {
		return err
	}
	archive := ArchiveExpiredSigningKeysDeps(deps)
	if _, err := ArchiveExpiredSigningKeys(signing, archive, now); err != nil {
		return err
	}
	xml := ports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
	scopes, err := deps.KeyStore.KeyScopes(xml)
	if err != nil {
		return err
	}
	for _, scopeID := range scopes {
		scoped := ports.WithKeyScope(xml, scopeID)
		if err := prepareFederationKeyIfDue(scoped, deps, now, cadence); err != nil {
			return err
		}
		if _, err := ArchiveExpiredSigningKeys(scoped, archive, now); err != nil {
			return err
		}
	}
	return nil
}

func prepareFederationKeyIfDue(ctx context.Context, deps SigningKeyLifecycleDeps, now time.Time, cadence time.Duration) error {
	keys, err := deps.KeyStore.GetAllKeys(ctx)
	if err != nil || !signingdomain.PreparationDue(keys, now, cadence) {
		return err
	}
	staged, ok := deps.KeyStore.(ports.FederationKeyStore)
	if !ok {
		return errors.New("the key store cannot publish XML keys")
	}
	result, err := staged.Prepare(ctx, now)
	if err != nil {
		return err
	}
	if result.Changed && deps.Emit != nil {
		deps.Emit(&signingdomain.SigningKeyPrepared{At: now, TenantID: result.Next.TenantID, Kid: result.Next.Kid})
	}
	return nil
}
