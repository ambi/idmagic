package bootstrap

import (
	"net/http"
	"time"
)

// httpServerHardening は system.yaml の HTTPServerHardening objective を実装する。
// 境界 http.Server に read_header / read / write / idle タイムアウトと、リクエスト
// ボディ上限を与え、slowloris 等の低速接続によるコネクション枯渇や巨大ボディに
// よるメモリ枯渇 (gosec G112 / CWE-400) に対する下限耐性を確保する。既定は本番
// 安全側に倒し、env で上書きできる。volumetric / TLS ハンドシェイク DoS の主防御は
// 前段プロキシが担い、本設定はプロキシ非依存の多層防御としてアプリ自身のホップを守る。
type httpServerHardening struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxBodyBytes      int64
}

// loadHTTPServerHardening は env から上書き可能なハードニング設定を組み立てる。
// 未指定・不正値は本番安全なデフォルトにフォールバックする (envDuration / envInt の規約)。
func loadHTTPServerHardening() httpServerHardening {
	return httpServerHardening{
		ReadHeaderTimeout: envDuration("HTTP_READ_HEADER_TIMEOUT", 10*time.Second),
		ReadTimeout:       envDuration("HTTP_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      envDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:       envDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		MaxBodyBytes:      int64(envInt("HTTP_MAX_BODY_BYTES", 1<<20)),
	}
}

// apply は基盤 http.Server にタイムアウトを設定する。echo.StartConfig.BeforeServeFunc
// から呼び出す。ボディ上限は echo BodyLimit middleware 側で適用する。
func (h httpServerHardening) apply(s *http.Server) {
	s.ReadHeaderTimeout = h.ReadHeaderTimeout
	s.ReadTimeout = h.ReadTimeout
	s.WriteTimeout = h.WriteTimeout
	s.IdleTimeout = h.IdleTimeout
}
