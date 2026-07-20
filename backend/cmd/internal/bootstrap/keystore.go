package bootstrap

import (
	"os"
	"strings"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_vault"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

// selectKeyStore は KEY_PROVIDER=vault のとき Vault Transit を本番 KeyProvider として
// 使い、それ以外は永続層が用意した dev/test fallback (local / postgres) を返す。
func selectKeyStore(fallback signingports.KeyStore) signingports.KeyStore {
	if !strings.EqualFold(os.Getenv("KEY_PROVIDER"), "vault") {
		return fallback
	}
	engine := signingcrypto.NewHTTPTransitEngine(
		os.Getenv("VAULT_ADDR"),
		os.Getenv("VAULT_TOKEN"),
		os.Getenv("VAULT_TRANSIT_MOUNT"),
	)
	return signingcrypto.NewVaultKeyStore(engine, os.Getenv("VAULT_KEY_PREFIX"))
}
