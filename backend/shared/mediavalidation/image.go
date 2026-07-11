// Package mediavalidation は画像アップロードの magic byte 検証を bounded context 横断で
// 共有する (wi-89, ADR-096)。Application icon (ADR-073) と Tenant branding asset
// (ADR-096) は同じ受理形式・検証方針を持つため、判定ロジックをここに集約し contract test
// で挙動を固定する。拡張子や Content-Type ヘッダーではなく magic byte で判定する。
package mediavalidation

import "errors"

var (
	ErrImageRequired = errors.New("image data is required")
	ErrImageTooLarge = errors.New("image exceeds the configured size limit")
	ErrImageFormat   = errors.New("image must be PNG, JPEG, WebP, or GIF")
)

// DetectImageContentType は画像バイト列を magic byte で判定し、PNG / JPEG / WebP / GIF の
// いずれかの content-type を返す。SVG を含むそれ以外の形式は ErrImageFormat で拒否する
// (保存型 XSS リスクを避けるため、初期実装では対応しない)。
func DetectImageContentType(data []byte, maxBytes int) (string, error) {
	if len(data) == 0 {
		return "", ErrImageRequired
	}
	if len(data) > maxBytes {
		return "", ErrImageTooLarge
	}
	switch {
	case len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == '\r' && data[5] == '\n' && data[6] == 0x1a && data[7] == '\n':
		return "image/png", nil
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg", nil
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", nil
	case len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a"):
		return "image/gif", nil
	default:
		return "", ErrImageFormat
	}
}
