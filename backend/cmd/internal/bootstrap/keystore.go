package bootstrap

import (
	"os"
	"strings"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
)

// selectKeyStore は KEY_PROVIDER=vault のとき Vault Transit を本番 KeyProvider として
// 使い、それ以外は永続層が用意した dev/test fallback (local / postgres) を返す。
func selectKeyStore(fallback oauthports.KeyStore) oauthports.KeyStore {
	if !strings.EqualFold(os.Getenv("KEY_PROVIDER"), "vault") {
		return fallback
	}
	engine := crypto.NewHTTPTransitEngine(
		os.Getenv("VAULT_ADDR"),
		os.Getenv("VAULT_TOKEN"),
		os.Getenv("VAULT_TRANSIT_MOUNT"),
	)
	return crypto.NewVaultKeyStore(engine, os.Getenv("VAULT_KEY_PREFIX"))
}
