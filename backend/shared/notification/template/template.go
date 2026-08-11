// Package template は通知メールの組込み既定カタログ、差し込み変数のレンダラ、
// locale 解決、およびそれらを束ねる Notifier を所有する。
//
// 送信は Authentication / IdManagement / IdGovernance の 3 context から起きるため、
// 文面解決は特定の context ではなく shared に置く。テナント上書きの永続化は Tenancy が
// 所有し、本パッケージは ports.TenantNotificationSource だけを知る。
package template

import "errors"

var (
	ErrUnknownTemplateKey = errors.New("unknown notification template key")
	ErrUnsupportedLocale  = errors.New("unsupported notification template locale")
	ErrUnknownPlaceholder = errors.New("notification template uses a placeholder outside the allowed set")
	ErrIncompleteTemplate = errors.New("notification template needs a subject, a text body, and an HTML body")
	ErrMissingVariable    = errors.New("notification template variable has no value")
)

// Definition は 1 テンプレートの文面。件名 / テキスト本文 / HTML 本文は必ず 3 点セットで
// 扱い、片方だけの状態を型で作れないようにする。
type Definition struct {
	Subject         string
	BodyText        string
	BodyHTML        string
	FromDisplayName string
}

// Rendered は差し込み変数を展開した描画結果。
type Rendered struct {
	Subject         string
	Text            string
	HTML            string
	FromDisplayName string
}
