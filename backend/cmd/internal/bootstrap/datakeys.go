package bootstrap

import (
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/security/envelope_openbao"
)

// selectMasterKeyProvider は cfg.DataKeyProvider=openbao のとき OpenBao Transit を
// 本番の master key custody として使い、それ以外は Tink cleartext keyset
// (dev/local、外部依存なし) を返す。selectKeyStore (署名鍵) と同じ
// 「env var で本番 provider へ切り替える」パターン。cfg は LoadSharedConfig で
// 既に検証済み (DATA_KEY_PROVIDER=openbao のとき OPENBAO_ADDR/OPENBAO_TOKEN は
// 必須) であるため、ここでは組み立てのみ行う (wi-97 T007 の起動時検証を維持)。
func selectMasterKeyProvider(cfg SharedConfig) (envelope_crypto.MasterKeyProvider, error) {
	if cfg.DataKeyProvider != "openbao" {
		return envelope_cleartext.NewCleartextMasterKeyProvider()
	}
	engine := envelope_openbao.NewHTTPTransitEngine(cfg.OpenBaoAddr, cfg.OpenBaoToken.Value(), cfg.OpenBaoTransitMount)
	return envelope_openbao.NewOpenBaoMasterKeyProvider(engine, cfg.OpenBaoDataKeyPrefix), nil
}
