package handlers_http_test

// docs/contexts/system/scenarios.feature.md の REQ-SYSTEM-013 が宣言する具体例を、実際の
// 経路を通して観測する。
//
// 「英語で返る」ことを機械で読む方法は 1 つしかない。人が読む文が非 ASCII を含まない
// ことである。UI が翻訳できるのは stable なエラーコードだけなので (REQ-SYSTEM-011)、
// バックエンドが本文を日本語化すると、en ロケールの利用者へ日本語が出る。それは辞書の
// 側からは直せない。
//
// 観測点は、具体例が名指す 2 つの形である。JSON API の Problem Details と、OAuth /
// OIDC のリダイレクトに載る `error_description` である。3 つめの「未知の内部エラー」は
// 共有の error handler が導出するので `support_http` の側で観測する。

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"unicode"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const errorLanguageVerifier = "error-language-examples-pkce-verifier-0123456789"

// englishText は、人が読む文が空でなく非 ASCII を含まないことを確かめる。
func englishText(t *testing.T, what, value string) {
	t.Helper()
	if value == "" {
		t.Errorf("%s is empty; there is no human-readable text to be in English", what)
		return
	}
	for _, r := range value {
		if r > unicode.MaxASCII {
			t.Errorf("%s = %q, want English text with no non-ASCII rune", what, value)
			return
		}
	}
}

//spec:covers EX-SYSTEM-013-01: 不正な JSON を送った HTTP API が既存のエラーコードとステータスを保ったまま英語の本文を返し、認可リクエストを拒否するリダイレクトが OAuth のエラーコードと英語の error_description を返すこと。
func TestAPIErrorTextStaysEnglish(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())

	// 1. 不正な JSON。トランザクションを開いてから送るのは、復号より前に立つ
	// ブラウザー要求の検証を通し、拒否の理由が JSON の不正であると読めるようにするため。
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	browser.Authorize(t, stack.AuthorizationQuery(errorLanguageVerifier, nil))
	transaction := browser.Transaction(t)
	csrf, _ := transaction["csrf_token"].(string)

	status, body := browser.PostRawJSON(t, "/api/auth/login", csrf, `{"username":`)
	if status != http.StatusBadRequest {
		t.Fatalf("不正な JSON の status=%d body=%v, want %d", status, body, http.StatusBadRequest)
	}
	if got, _ := body["type"].(string); got != "urn:idmagic:error:invalid_request" {
		t.Errorf("type=%q, want urn:idmagic:error:invalid_request", got)
	}
	title, _ := body["title"].(string)
	englishText(t, "title", title)
	detail, _ := body["detail"].(string)
	englishText(t, "detail", detail)

	// 2. 認可リクエストの拒否。prompt=none は同意画面へ回れないので、同意の無い
	// リクエストは登録済みの redirect_uri へ consent_required として戻る。説明文が
	// 載るのはこちらの経路で、セッションの無い login_required には説明が無い。
	signedIn := s.Browser(t, tenancydomain.DefaultRealm)
	signedIn.Authorize(t, stack.AuthorizationQuery(errorLanguageVerifier, nil))
	signedIn.SignIn(t, "user", stack.UserPassword)
	refused := signedIn.Authorize(t, stack.AuthorizationQuery(errorLanguageVerifier,
		map[string]string{"prompt": "none"}))

	location := refused.Header.Get("Location")
	if !strings.HasPrefix(location, stack.BrowserRedirectURI) {
		t.Fatalf("Location=%q が登録済みの redirect_uri を指していない", location)
	}
	redirect, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Location を URL として読めない %q: %v", location, err)
	}
	query := redirect.Query()
	if got := query.Get("error"); got != "consent_required" {
		t.Fatalf("error=%q, want consent_required (Location=%q)", got, location)
	}
	englishText(t, "error_description", query.Get("error_description"))
}
