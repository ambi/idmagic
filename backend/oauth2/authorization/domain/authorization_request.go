// 認可リクエストのドメインモデル。prompt / max_age / id_token_hint 制御を含む。
package domain

import (
	"fmt"
	"strings"
	"time"
)

// PromptTokens は OIDC prompt の正規化済み値集合である。
type PromptTokens struct {
	Login   bool
	Consent bool
	None    bool
}

// ParsePromptTokens は OIDC Core の空白区切り prompt grammar を fail-closed で解釈する。
func ParsePromptTokens(value string) (PromptTokens, error) {
	var tokens PromptTokens
	for token := range strings.FieldsSeq(value) {
		switch token {
		case "login":
			if tokens.Login {
				return PromptTokens{}, fmt.Errorf("prompt token login is duplicated")
			}
			tokens.Login = true
		case "consent":
			if tokens.Consent {
				return PromptTokens{}, fmt.Errorf("prompt token consent is duplicated")
			}
			tokens.Consent = true
		case "none":
			if tokens.None {
				return PromptTokens{}, fmt.Errorf("prompt token none is duplicated")
			}
			tokens.None = true
		default:
			return PromptTokens{}, fmt.Errorf("unsupported prompt token %q", token)
		}
	}
	if tokens.None && (tokens.Login || tokens.Consent) {
		return PromptTokens{}, fmt.Errorf("prompt none must not be combined with other tokens")
	}
	return tokens, nil
}

func (p PromptTokens) Canonical() string {
	parts := make([]string, 0, 3)
	if p.Login {
		parts = append(parts, "login")
	}
	if p.Consent {
		parts = append(parts, "consent")
	}
	if p.None {
		parts = append(parts, "none")
	}
	return strings.Join(parts, " ")
}

// AuthorizationRequestPolicy は prompt / max_age / id_token_hint 等の OIDC 規定値による
// 再認証必要性判断をまとめる。
type AuthorizationRequestPolicy struct {
	Prompt      *string
	MaxAge      *int
	IDTokenHint *string
}

// NeedsReauthentication は context (現セッション) が要件を満たすか判定する。
//   - prompt=login: 常に true
//   - prompt=none: 認証されていない場合は呼び出し側で access_denied
//   - max_age: auth_time が古ければ true
//   - id_token_hint: 別ユーザー対象なら true (本実装ではプロト簡略化のため未検査)
func NeedsReauthentication(p AuthorizationRequestPolicy, authTime, now time.Time, promptLoginSatisfied bool) bool {
	if p.Prompt != nil {
		switch *p.Prompt {
		case "login":
			return !promptLoginSatisfied
		case "none":
			return false
		}
	}
	if p.MaxAge != nil {
		maxAge := time.Duration(*p.MaxAge) * time.Second
		if now.Sub(authTime) >= maxAge {
			return true
		}
	}
	return false
}

func ParsePrompt(req *AuthorizationRequest) AuthorizationRequestPolicy {
	return AuthorizationRequestPolicy{
		Prompt:      req.Prompt,
		MaxAge:      req.MaxAge,
		IDTokenHint: nil,
	}
}

// ScopeIntersection は scope 文字列を要素分割して共通部分を返す。
func ScopeIntersection(requested, allowed string) []string {
	allow := map[string]bool{}
	for s := range strings.FieldsSeq(allowed) {
		allow[s] = true
	}
	var out []string
	for s := range strings.FieldsSeq(requested) {
		if allow[s] {
			out = append(out, s)
		}
	}
	return out
}
