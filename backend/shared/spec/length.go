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

// 長さ違反だけを他の検証失敗と区別するための issue code。HTTP 層はこの区別を
// 使って 422 を返す。
const (
	issueCodeMaxChars = "max_chars"
	issueCodeMinChars = "min_chars"
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
