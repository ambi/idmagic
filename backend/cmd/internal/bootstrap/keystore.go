package bootstrap

import (
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_vault"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

// selectKeyStore は cfg.KeyProvider=vault のとき Vault Transit を本番 KeyProvider として
// 使い、それ以外は永続層が用意した dev/test fallback (local / postgres) を返す。cfg は
// LoadSharedConfig で既に検証済み (KEY_PROVIDER=vault のとき VAULT_ADDR/VAULT_TOKEN は
// 必須) であるため、ここでは組み立てのみ行う。
func selectKeyStore(cfg SharedConfig, fallback signingports.KeyStore) signingports.KeyStore {
	if cfg.KeyProvider != "vault" {
		return fallback
	}
	engine := signingcrypto.NewHTTPTransitEngine(cfg.VaultAddr, cfg.VaultToken.Value(), cfg.VaultTransitMount)
	return signingcrypto.NewVaultKeyStore(engine, cfg.VaultKeyPrefix)
}
