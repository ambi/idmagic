package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/security/envelope_openbao"
)

// selectMasterKeyProvider は DATA_KEY_PROVIDER=openbao のとき OpenBao Transit を
// 本番の master key custody として使い、それ以外は Tink cleartext keyset
// (dev/local、外部サービス不要) を返す。selectKeyStore (署名鍵) と同じ
// 「env var で本番 provider へ切り替える」パターン。DATA_KEY_PROVIDER=openbao で
// OPENBAO_ADDR/OPENBAO_TOKEN が空のときは起動時に失敗させる (wi-97 T007): 検証
// しないと、最初の暗号化操作まで誤設定に気づけず fail-closed の発火が遅れる。
func selectMasterKeyProvider() (envelope_crypto.MasterKeyProvider, error) {
	if !strings.EqualFold(os.Getenv("DATA_KEY_PROVIDER"), "openbao") {
		return envelope_cleartext.NewCleartextMasterKeyProvider()
	}
	addr := os.Getenv("OPENBAO_ADDR")
	token := os.Getenv("OPENBAO_TOKEN")
	if addr == "" || token == "" {
		return nil, fmt.Errorf("datakeys: DATA_KEY_PROVIDER=openbao requires OPENBAO_ADDR and OPENBAO_TOKEN to be set")
	}
	engine := envelope_openbao.NewHTTPTransitEngine(addr, token, os.Getenv("OPENBAO_TRANSIT_MOUNT"))
	prefix := os.Getenv("OPENBAO_DATA_KEY_PREFIX")
	if prefix == "" {
		prefix = "idmagic/datakeys"
	}
	return envelope_openbao.NewOpenBaoMasterKeyProvider(engine, prefix), nil
}
