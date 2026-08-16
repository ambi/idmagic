package spec

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	z "github.com/Oudwins/zog"
)

// 文字列長の上限の既定区分。単位は Unicode コードポイントで、TypeSpec の
// @maxLength、OpenAPI の maxLength、PostgreSQL の char_length() と同じものを数える。
// 正本は spec/SPECIFICATION.md の "String length limits"。
const (
	// LengthHandle は IdMagic が採番する集約の ID と、関係名や型名のような語彙的な名前。
	LengthHandle = 64
	// LengthName は一行の名前。
	LengthName = 100
	// LengthDisplayName は利用者に見せる表示名とメールの件名。
	LengthDisplayName = 200
	// LengthExternalID は呼び出し側の資源空間から来る識別子。
	LengthExternalID = 256
	// LengthDescription は数文の説明、パターン、表示テンプレート。
	LengthDescription = 500
	// LengthURI は URL と URI。
	LengthURI = 2048
	// LengthExpression は CEL のような式。
	LengthExpression = 4096
	// LengthPlainBody は平文の本文。
	LengthPlainBody = 8000
	// LengthRichBody は HTML の本文。
	LengthRichBody = 20000
)

// 外部の標準または固定の表示面から上限が決まる値。既定の区分の外に置く。
const (
	// LengthEmail は RFC 5321 が定めるメールアドレスの上限 (254 オクテット)。
	// 書式を ASCII に限っているのでコードポイント数と一致する。
	LengthEmail = 254
	// LengthDNSLabel は DNS ラベル 1 つ分の上限。realm がこれに従う。
	LengthDNSLabel = 63
	// LengthDNSName は DNS 名全体の上限。trust domain がこれに従う。
	LengthDNSName = 255
	// LengthClientID は OAuth 2.0 の client_id。UUID を収めたうえで、
	// 他の認可サーバーから移入した値も受けられる幅を取る。
	LengthClientID = 128
	// LengthChromeLabel はサインイン画面とメールの固定枠に収まる短いラベル。
	LengthChromeLabel = 80
	// LengthChromeText は同じ固定枠に置く補足テキスト。
	LengthChromeText = 280
)

// 索引の鍵の成分になる文字列の、契約の上限 (コードポイント)。資源の上限 (バイト) が
// 対になっており、両方を課す。数の根拠は spec/SPECIFICATION.md の
// "String length limits" にある。
const (
	// LengthSamlEntityID は SAML SP の entityID。saml-schema-metadata-2.0.xsd の
	// entityIDType は 1024 文字を定めるが、それを超える非準拠 SP を拒否しないため
	// URI 区分の外側に取る。
	LengthSamlEntityID = LengthURI
	// LengthWtrealm は WS-Federation の relying party 識別子。標準は URI としか
	// 定めないので URI 区分に従う。
	LengthWtrealm = LengthURI
	// LengthProtocolMessageID は相手のプロトコルメッセージが名乗る ID。SAML の
	// AuthnRequest / Response の ID、DPoP proof と client assertion の jti。
	// いずれの標準も長さを定めていない。
	LengthProtocolMessageID = LengthExternalID
	// LengthFederatedSubject は外部 IdP が名乗る subject。OpenID Connect Core 1.0
	// の sub は 255 ASCII 文字以下だが、SAML の NameID には規定がないので広く取る。
	LengthFederatedSubject = 512
	// LengthWebAuthnCredentialID は authenticator が決める credential ID の
	// base64url 表現。WebAuthn は credential ID を 1023 バイト以下と定めるので
	// base64url で 1364 文字になる。
	LengthWebAuthnCredentialID = LengthURI
	// LengthSubjectDN は tls_client_auth の証明書 Subject DN。索引の鍵ではないが
	// 外部が値を決めるので上限を置く。
	LengthSubjectDN = LengthURI
	// LengthQuarantineReason は連携先が返したエラー文。利用者の入力ではないので
	// 超過は拒否せず、書き込み側が TruncateChars で切り詰める。
	LengthQuarantineReason = LengthDescription
)

// 資源の上限 (バイト)。btree v4 の索引行に収まることを保証するためだけの数で、
// 契約の上限とは役割が違う。バイトで数えるのは、btree が制限しているものが
// バイトだからである。
const (
	BytesSamlEntityID          = 2048
	BytesWtrealm               = 2048
	BytesProtocolMessageID     = 256
	BytesFederatedSubject      = 1024
	BytesWebAuthnCredentialID  = 2048
	BytesGeneratedKeyComponent = LengthHandle
)

// BtreeIndexRowLimitBytes は PostgreSQL の btree v4 が索引行 1 件に課す上限。
// これを超えると挿入が SQLSTATE 54000 で落ちる。
const BtreeIndexRowLimitBytes = 2704

// KeyByteBudget は索引の鍵 1 件あたりに使ってよいバイト数。btree の上限との差が、
// 索引タプル自身が使う領域と、将来列を足すための余白になる。
const KeyByteBudget = 2400

// 長さ違反だけを他の検証失敗と区別するための issue code。HTTP 層はこの区別を
// 使って 422 を返す。
const (
	issueCodeMaxChars = "max_chars"
	issueCodeMinChars = "min_chars"
	issueCodeMaxBytes = "max_bytes"
)

// Chars は下限と上限をコードポイント数で課す文字列スキーマを返す。zog の Min / Max は
// len(string)、すなわち UTF-8 バイト数を数えるため文字列フィールドには使えない。
// 上限 100 のつもりで Max(100) と書くと、日本語では 33 文字で拒否される。
// minimum が 0 のときは下限を課さない。
func Chars(minimum, maximum int) *z.StringSchema[string] {
	schema := z.String()
	if minimum > 0 {
		schema = schema.TestFunc(
			func(value *string, _ z.Ctx) bool { return utf8.RuneCountInString(*value) >= minimum },
			z.IssueCode(issueCodeMinChars),
			z.Message(fmt.Sprintf("must be at least %d characters", minimum)),
		)
	}
	return schema.TestFunc(
		func(value *string, _ z.Ctx) bool { return utf8.RuneCountInString(*value) <= maximum },
		z.IssueCode(issueCodeMaxChars),
		z.Message(fmt.Sprintf("must be at most %d characters", maximum)),
	)
}

// CharsAtMost は上限だけを課す文字列スキーマを返す。空文字列を「未設定」として
// 扱う省略可能なフィールドで使う。
func CharsAtMost(maximum int) *z.StringSchema[string] { return Chars(0, maximum) }

// KeyString は索引の鍵の成分になる文字列のスキーマを返す。契約の上限 (コードポイント)
// と資源の上限 (バイト) を重ねて課す。コードポイントの上限だけでは、標準が許す
// 長さのまま UTF-8 で btree の索引行上限を超える値を通してしまう。
func KeyString(maxChars, maxBytes int) *z.StringSchema[string] {
	return Chars(0, maxChars).TestFunc(
		func(value *string, _ z.Ctx) bool { return len(*value) <= maxBytes },
		z.IssueCode(issueCodeMaxBytes),
		z.Message(fmt.Sprintf("must be at most %d bytes", maxBytes)),
	)
}

// LengthError は文字列フィールドが長さの制約に違反したことを表す。解析はできた
// 内容が業務規則に違反する場合なので、HTTP 層はこれを 422 に写像する。長さ以外の
// 検証失敗は素の error のままにして、保存済みデータの破損がサーバの不具合として
// 扱われる余地を残す。
type LengthError struct{ message string }

func (e *LengthError) Error() string { return e.message }

// IsFieldLengthViolation は support_http の構造的インターフェースが照合する目印。
// import の向きを spec → http に作らないためのもので、値そのものに意味はない。
func (e *LengthError) IsFieldLengthViolation() bool { return true }

// CheckMaxChars は zog スキーマを経由しない検査から同じ上限を課す。field には
// 公開契約の wire 名を渡す。上限内なら nil を返す。
func CheckMaxChars(field, value string, maximum int) error {
	actual := utf8.RuneCountInString(value)
	if actual <= maximum {
		return nil
	}
	return &LengthError{
		message: fmt.Sprintf("%s: must be at most %d characters, got %d", field, maximum, actual),
	}
}

// CheckMaxBytes は資源の上限を zog スキーマを経由しない検査から課す。field には
// 公開契約の wire 名を渡す。上限内なら nil を返す。
func CheckMaxBytes(field, value string, maximum int) error {
	actual := len(value)
	if actual <= maximum {
		return nil
	}
	return &LengthError{
		message: fmt.Sprintf("%s: must be at most %d bytes, got %d", field, maximum, actual),
	}
}

// CheckKeyString は索引の鍵の成分に契約の上限と資源の上限の両方を課す。
func CheckKeyString(field, value string, maxChars, maxBytes int) error {
	if err := CheckMaxChars(field, value, maxChars); err != nil {
		return err
	}
	return CheckMaxBytes(field, value, maxBytes)
}

// TruncateChars は値をコードポイント単位で切り詰める。利用者の入力ではなく、
// 連携先が返した診断用の文字列にだけ使う。入力を黙って短くしてよい場面は他にない。
func TruncateChars(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum])
}

// snakeCase は zog が返す Go の構造体フィールド名を、公開契約が使う wire 名へ
// 近づける。PreferredUsername → preferred_username、OIDCScope → oidc_scope。
func snakeCase(name string) string {
	runes := []rune(name)
	out := make([]rune, 0, len(runes)+4)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			prevIsLower := i > 0 && !unicode.IsUpper(runes[i-1]) && runes[i-1] != '_'
			// 連続する大文字の並びは 1 語として扱い、次が小文字になる直前で切る。
			endsAcronym := i > 0 && i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && unicode.IsLower(runes[i+1])
			if prevIsLower || endsAcronym {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
