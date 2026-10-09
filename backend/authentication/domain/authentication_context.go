// Authentication bounded context の境界。OAuth2/OIDC ユースケースはこの context を消費するだけで、
// password 検証・user lookup・session cookie の詳細には踏み込まない。
package domain

import (
	"context"
	"time"
)

type AuthenticationContext struct {
	UserID                string
	AuthTime              int64
	AMR                   []string
	ACR                   string
	SessionID             string
	AuthenticationPending bool
	PendingPurpose        LoginPendingPurpose
	EnrollmentDeadline    *time.Time
	EnrollmentBypassID    string
	// StepUpAt は直近の step-up 再認証時刻 (Unix 秒、未実施は 0)。高 sensitivity 操作の
	// recency gate が AuthTime と合わせて評価する。
	StepUpAt int64
}

// Subject、AuthenticatedAt、Session、Pending は、共有の HTTP 支援が認証の結果として
// 読む見方である。AuthenticationContext はそのまま認証の結果として渡せる。
func (a *AuthenticationContext) Subject() string        { return a.UserID }
func (a *AuthenticationContext) AuthenticatedAt() int64 { return a.AuthTime }
func (a *AuthenticationContext) Session() string        { return a.SessionID }
func (a *AuthenticationContext) Pending() bool          { return a.AuthenticationPending }

// ResolvedAuthentication は、共有の HTTP 支援が返す認証の結果のうち、文脈を組み立て直すのに
// 読む部分である。
type ResolvedAuthentication interface {
	Subject() string
	AuthenticatedAt() int64
}

// ContextOf は、共有の HTTP 支援が返した認証の結果を認証の文脈として返す。
// セッションで認証した結果はその文脈そのものである。アクセストークンで認証した結果は
// セッションを持たないので、主体と認証時刻だけを持つ文脈にする。
func ContextOf(authn ResolvedAuthentication) *AuthenticationContext {
	if authn == nil {
		return nil
	}
	if resolved, ok := authn.(*AuthenticationContext); ok {
		return resolved
	}
	return &AuthenticationContext{UserID: authn.Subject(), AuthTime: authn.AuthenticatedAt()}
}

type LoginPendingPurpose string

const (
	LoginPendingNone       LoginPendingPurpose = "None"
	LoginPendingChallenge  LoginPendingPurpose = "Challenge"
	LoginPendingEnrollment LoginPendingPurpose = "Enrollment"
)

func (p LoginPendingPurpose) Valid() bool {
	switch p {
	case LoginPendingNone, LoginPendingChallenge, LoginPendingEnrollment:
		return true
	}
	return false
}

type AuthenticationContextResolver interface {
	Resolve(ctx context.Context, headers Headers) (*AuthenticationContext, error)
}

// Headers は HTTP framework 非依存の薄い抽象 (key → first value)。
type Headers interface {
	Get(key string) string
}

// HTTPHeadersAdapter は標準 http.Header から Headers への変換。
type HTTPHeadersAdapter struct {
	H interface{ Get(string) string }
}

func (h HTTPHeadersAdapter) Get(k string) string { return h.H.Get(k) }
