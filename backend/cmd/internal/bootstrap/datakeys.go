package bootstrap

import (
	"os"
	"strings"

	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/security/envelope_openbao"
)

// selectMasterKeyProvider は DATA_KEY_PROVIDER=openbao のとき OpenBao Transit を
// 本番の master key custody として使い、それ以外は Tink cleartext keyset
// (dev/local、外部サービス不要) を返す (ADR-148)。selectKeyStore (署名鍵) と同じ
// 「env var で本番 provider へ切り替える」パターン。
func selectMasterKeyProvider() (envelope_crypto.MasterKeyProvider, error) {
	if !strings.EqualFold(os.Getenv("DATA_KEY_PROVIDER"), "openbao") {
		return envelope_cleartext.NewCleartextMasterKeyProvider()
	}
	engine := envelope_openbao.NewHTTPTransitEngine(
		os.Getenv("OPENBAO_ADDR"),
		os.Getenv("OPENBAO_TOKEN"),
		os.Getenv("OPENBAO_TRANSIT_MOUNT"),
	)
	prefix := os.Getenv("OPENBAO_DATA_KEY_PREFIX")
	if prefix == "" {
		prefix = "idmagic/datakeys"
	}
	return envelope_openbao.NewOpenBaoMasterKeyProvider(engine, prefix), nil
}
