// Authentication モジュールの境界。OAuth2/OIDC ユースケースはこの context を消費するだけで、
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

// Pending は、認証結果が未取得でも認証待ちとして拒否できるよう nil を扱う。
func (a *AuthenticationContext) Pending() bool { return a == nil || a.AuthenticationPending }

// MFAAuthenticated は ACR の意味を、方式に依存しない認証事実へ射影する。
func (a *AuthenticationContext) MFAAuthenticated() bool { return ACRSatisfies(a.ACR, ACRMFA) }

// SecondFactorAuthenticated は、記憶済み端末の充足と実際の第二要素を区別する。
// 同じセッションに両方があっても、実際の第二要素の成立は失われない。
func (a *AuthenticationContext) SecondFactorAuthenticated() bool {
	for _, method := range a.AMR {
		if method != AMRTrustedDevice && IsMfaAMR(method) {
			return true
		}
	}
	return false
}

func (a *AuthenticationContext) SteppedUpAt() int64 { return a.StepUpAt }

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
