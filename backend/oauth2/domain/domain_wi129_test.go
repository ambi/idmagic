package domain

import (
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// GenerateAuthorizationCode は既定値補完と生成コードの妥当性を検証する。
func TestGenerateAuthorizationCode_Defaults(t *testing.T) {
	rec, err := GenerateAuthorizationCode(AuthorizationCodeInput{
		AuthorizationRequestID: "00000000-0000-4000-8000-000000000001",
		ClientID:               "client-1",
		Sub:                    "user-1",
		Scopes:                 []string{"openid"},
		RedirectURI:            "https://app.example/cb",
		CodeChallenge:          "abc",
		CodeChallengeMethod:    spec.CodeChallengeMethodS256,
	})
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	if rec.TenantID != tenancydomain.DefaultTenantID {
		t.Errorf("TenantID 既定値が未補完: %q", rec.TenantID)
	}
	if rec.State != spec.AuthCodeRecordIssued {
		t.Errorf("State=%q, want issued", rec.State)
	}
	if rec.Code == "" {
		t.Error("Code が空")
	}
	// 既定 TTL は 60 秒。
	if got := rec.ExpiresAt.Sub(rec.IssuedAt); got != 60*time.Second {
		t.Errorf("TTL=%v, want 60s", got)
	}
}

// GenerateAuthorizationCode は明示 TTL / TenantID / Now を尊重する。
func TestGenerateAuthorizationCode_ExplicitValues(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	rec, err := GenerateAuthorizationCode(AuthorizationCodeInput{
		TenantID:               "tenant-x",
		AuthorizationRequestID: "00000000-0000-4000-8000-000000000002",
		ClientID:               "client-2",
		Sub:                    "user-2",
		Scopes:                 []string{"openid", "profile"},
		RedirectURI:            "https://app.example/cb",
		CodeChallenge:          "chal",
		CodeChallengeMethod:    spec.CodeChallengeMethodS256,
		Nonce:                  new("nonce-1"),
		AuthTime:               now.Unix(),
		TTLSeconds:             120,
		Now:                    now,
	})
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	if rec.TenantID != "tenant-x" {
		t.Errorf("TenantID=%q, want tenant-x", rec.TenantID)
	}
	if !rec.ExpiresAt.Equal(now.Add(120 * time.Second)) {
		t.Errorf("ExpiresAt=%v, want now+120s", rec.ExpiresAt)
	}
	if rec.Nonce == nil || *rec.Nonce != "nonce-1" {
		t.Errorf("Nonce が保持されていない: %v", rec.Nonce)
	}
}

// GenerateAuthorizationCode は Validate 失敗をエラーとして返す (redirect_uri 欠落)。
func TestGenerateAuthorizationCode_ValidationError(t *testing.T) {
	_, err := GenerateAuthorizationCode(AuthorizationCodeInput{
		AuthorizationRequestID: "00000000-0000-4000-8000-000000000003",
		ClientID:               "client-3",
		Sub:                    "user-3",
		Scopes:                 []string{"openid"},
		CodeChallenge:          "chal",
		CodeChallengeMethod:    spec.CodeChallengeMethodS256,
	})
	if err == nil {
		t.Fatal("redirect_uri 欠落で失敗するはず")
	}
}

func TestIsCodeExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := &AuthorizationCodeRecord{ExpiresAt: now.Add(time.Minute)}
	if IsCodeExpired(rec, now) {
		t.Error("期限内なのに期限切れ判定")
	}
	if !IsCodeExpired(rec, now.Add(2*time.Minute)) {
		t.Error("期限後なのに有効判定")
	}
	// now がゼロ値なら現在時刻で評価 (過去の ExpiresAt は期限切れ)。
	past := &AuthorizationCodeRecord{ExpiresAt: time.Now().Add(-time.Hour)}
	if !IsCodeExpired(past, time.Time{}) {
		t.Error("ゼロ now で過去期限が期限切れ判定されない")
	}
}

func TestIsCodeRedeemed(t *testing.T) {
	if IsCodeRedeemed(&AuthorizationCodeRecord{}) {
		t.Error("未使用コードが redeemed 判定")
	}
	ts := time.Now()
	if !IsCodeRedeemed(&AuthorizationCodeRecord{RedeemedAt: &ts}) {
		t.Error("使用済みコードが未使用判定")
	}
}

func TestNeedsReauthentication(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		policy     AuthorizationRequestPolicy
		authTime   time.Time
		now        time.Time
		loginSat   bool
		wantReauth bool
	}{
		{"prompt=login 未充足", AuthorizationRequestPolicy{Prompt: new("login")}, base, base, false, true},
		{"prompt=login 充足済み", AuthorizationRequestPolicy{Prompt: new("login")}, base, base, true, false},
		{"prompt=none は再認証不要", AuthorizationRequestPolicy{Prompt: new("none")}, base, base, false, false},
		{"max_age 超過", AuthorizationRequestPolicy{MaxAge: new(60)}, base, base.Add(2 * time.Minute), false, true},
		{"max_age 内", AuthorizationRequestPolicy{MaxAge: new(300)}, base, base.Add(time.Minute), false, false},
		{"指定なし", AuthorizationRequestPolicy{}, base, base, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NeedsReauthentication(tc.policy, tc.authTime, tc.now, tc.loginSat); got != tc.wantReauth {
				t.Errorf("got %v, want %v", got, tc.wantReauth)
			}
		})
	}
}

func TestParsePrompt(t *testing.T) {
	p := ParsePrompt(&AuthorizationRequest{Prompt: new("login"), MaxAge: new(30)})
	if p.Prompt == nil || *p.Prompt != "login" {
		t.Errorf("Prompt が引き継がれていない: %v", p.Prompt)
	}
	if p.MaxAge == nil || *p.MaxAge != 30 {
		t.Errorf("MaxAge が引き継がれていない: %v", p.MaxAge)
	}
	if p.IDTokenHint != nil {
		t.Error("IDTokenHint は nil のはず")
	}
}

// SCL: OIDC-CORE-CODE-FLOW の prompt token grammar。
func TestParsePromptTokens(t *testing.T) {
	for _, tt := range []struct {
		input   string
		valid   bool
		login   bool
		consent bool
		none    bool
	}{
		{"login consent", true, true, true, false},
		{"none", true, false, false, true},
		{"none login", false, false, false, false},
		{"login login", false, false, false, false},
		{"select_account", false, false, false, false},
	} {
		got, err := ParsePromptTokens(tt.input)
		if (err == nil) != tt.valid {
			t.Errorf("ParsePromptTokens(%q) error=%v, valid=%v", tt.input, err, tt.valid)
			continue
		}
		if err == nil && (got.Login != tt.login || got.Consent != tt.consent || got.None != tt.none) {
			t.Errorf("ParsePromptTokens(%q)=%+v", tt.input, got)
		}
	}
}

func TestScopeIntersection(t *testing.T) {
	got := ScopeIntersection("openid profile email admin", "openid email offline_access")
	if len(got) != 2 || got[0] != "openid" || got[1] != "email" {
		t.Fatalf("交差が不正: %v", got)
	}
	if got := ScopeIntersection("", "openid"); got != nil {
		t.Errorf("空リクエストは nil のはず: %v", got)
	}
	if got := ScopeIntersection("a b", "c d"); got != nil {
		t.Errorf("共通なしは nil のはず: %v", got)
	}
}

// --- device authorization ---

func TestGenerateDeviceCode_Unique(t *testing.T) {
	c1, err := GenerateDeviceCode()
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	c2, _ := GenerateDeviceCode()
	if c1 == "" || c1 == c2 {
		t.Errorf("device code が空または重複: %q %q", c1, c2)
	}
}

func TestHashDeviceCode_Stable(t *testing.T) {
	h1 := HashDeviceCode("device-code-abc")
	h2 := HashDeviceCode("device-code-abc")
	if h1 != h2 {
		t.Error("同一入力のハッシュが不一致")
	}
	if len(h1) != 64 {
		t.Errorf("SHA-256 hex 長が不正: %d", len(h1))
	}
	if HashDeviceCode("other") == h1 {
		t.Error("異なる入力のハッシュが一致")
	}
}

func TestGenerateUserCode_Format(t *testing.T) {
	code, err := GenerateUserCode()
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	// XXXX-XXXX 形式。
	if len(code) != 9 || code[4] != '-' {
		t.Fatalf("user code 形式が不正: %q", code)
	}
	for i, r := range code {
		if i == 4 {
			continue
		}
		if !strings.ContainsRune(userCodeCharset, r) {
			t.Errorf("charset 外の文字: %q in %q", r, code)
		}
	}
}

func TestNormalizeUserCode(t *testing.T) {
	cases := map[string]string{
		"bcdf-ghjk": "BCDFGHJK",
		"BCDF GHJK": "BCDFGHJK",
		"b_c!d#f":   "BCDF",
		"12ab":      "12AB",
	}
	for in, want := range cases {
		if got := NormalizeUserCode(in); got != want {
			t.Errorf("NormalizeUserCode(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestIsDeviceExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// State が expired なら常に true。
	if !IsDeviceExpired(&DeviceAuthorization{State: spec.DeviceFlowExpired, ExpiresAt: now.Add(time.Hour)}, now) {
		t.Error("expired state は常に期限切れ")
	}
	active := &DeviceAuthorization{State: spec.DeviceFlowIssued, ExpiresAt: now.Add(time.Minute)}
	if IsDeviceExpired(active, now) {
		t.Error("期限内なのに期限切れ判定")
	}
	if !IsDeviceExpired(active, now.Add(2*time.Minute)) {
		t.Error("期限後なのに有効判定")
	}
	// now ゼロ値なら現在時刻評価。
	past := &DeviceAuthorization{State: spec.DeviceFlowIssued, ExpiresAt: time.Now().Add(-time.Hour)}
	if !IsDeviceExpired(past, time.Time{}) {
		t.Error("ゼロ now で過去期限が期限切れ判定されない")
	}
}

// --- refresh token ---

func TestHashRefreshToken_Stable(t *testing.T) {
	h := HashRefreshToken("token-xyz")
	if h != HashRefreshToken("token-xyz") {
		t.Error("同一入力のハッシュ不一致")
	}
	if len(h) != 64 {
		t.Errorf("hex 長が不正: %d", len(h))
	}
}

func TestGenerateInitialRefreshToken(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	sc := &SenderConstraint{Type: SenderConstraintDPoP, JKT: "jkt-1"}
	sid := "session-1"
	gen, err := GenerateInitialRefreshToken("client-1", "user-1", []string{"openid"}, sc, &sid, now)
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	rec := gen.Record
	if gen.Token == "" {
		t.Error("トークンが空")
	}
	if rec.Hash != HashRefreshToken(gen.Token) {
		t.Error("Hash がトークンと一致しない")
	}
	if rec.ParentID != nil {
		t.Error("初回発行に ParentID があってはならない")
	}
	if rec.FamilyID == "" || rec.ID == "" {
		t.Error("ID/FamilyID が空")
	}
	if !rec.ExpiresAt.Equal(now.Add(refreshTokenTTL)) {
		t.Errorf("ExpiresAt=%v", rec.ExpiresAt)
	}
	if !rec.AbsoluteExpiresAt.Equal(now.Add(refreshTokenAbsoluteTTL)) {
		t.Errorf("AbsoluteExpiresAt=%v", rec.AbsoluteExpiresAt)
	}
	if rec.SenderConstraint == nil || rec.SenderConstraint.JKT != "jkt-1" {
		t.Error("SenderConstraint が保持されていない")
	}
	// ADR-127: sid は AuthorizationCodeRecord.sid からそのまま引き継ぐ。
	if rec.Sid == nil || *rec.Sid != sid {
		t.Errorf("Sid が伝播していない: %v", rec.Sid)
	}
}

func TestGenerateInitialRefreshToken_NilSid(t *testing.T) {
	// client_credentials 等 browser session を持たない発行では sid は nil のまま (ADR-127)。
	gen, err := GenerateInitialRefreshToken("client-1", "user-1", []string{"openid"}, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	if gen.Record.Sid != nil {
		t.Errorf("Sid は nil のはず: %v", gen.Record.Sid)
	}
}

func TestGenerateInitialRefreshToken_ZeroNow(t *testing.T) {
	gen, err := GenerateInitialRefreshToken("client-1", "user-1", []string{"openid"}, nil, nil, time.Time{})
	if err != nil {
		t.Fatalf("生成に失敗: %v", err)
	}
	if gen.Record.IssuedAt.IsZero() {
		t.Error("ゼロ now で IssuedAt が補完されていない")
	}
}

func TestRotateRefreshToken(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	sid := "session-1"
	parent, err := GenerateInitialRefreshToken("client-1", "user-1", []string{"openid"}, nil, &sid, now)
	if err != nil {
		t.Fatalf("親生成に失敗: %v", err)
	}
	rotated, err := RotateRefreshToken(parent.Record, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("回転に失敗: %v", err)
	}
	rec := rotated.Record
	if rec.ParentID == nil || *rec.ParentID != parent.Record.ID {
		t.Errorf("ParentID が親を指していない: %v", rec.ParentID)
	}
	if rec.FamilyID != parent.Record.FamilyID {
		t.Error("FamilyID が継承されていない")
	}
	if rec.ID == parent.Record.ID {
		t.Error("回転後の ID が親と同一")
	}
	// ADR-127: rotate 後も親の sid をそのまま引き継ぐ。
	if rec.Sid == nil || *rec.Sid != sid {
		t.Errorf("Sid が rotate で引き継がれていない: %v", rec.Sid)
	}
}

// RotateRefreshToken は absolute_expires_at を超えないよう ExpiresAt を打ち切る。
func TestRotateRefreshToken_CappedByAbsolute(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	parent := &RefreshTokenRecord{
		ID:                "00000000-0000-4000-8000-0000000000a1",
		Hash:              HashRefreshToken("x"),
		FamilyID:          "00000000-0000-4000-8000-0000000000f1",
		ClientID:          "client-1",
		UserID:            "user-1",
		Scopes:            []string{"openid"},
		IssuedAt:          now.Add(-refreshTokenAbsoluteTTL + time.Hour),
		ExpiresAt:         now.Add(time.Hour),
		AbsoluteExpiresAt: now.Add(30 * time.Minute), // TTL より短い
	}
	rotated, err := RotateRefreshToken(parent, now)
	if err != nil {
		t.Fatalf("回転に失敗: %v", err)
	}
	if !rotated.Record.ExpiresAt.Equal(parent.AbsoluteExpiresAt) {
		t.Errorf("ExpiresAt が absolute で打ち切られていない: %v", rotated.Record.ExpiresAt)
	}
}

func TestIsRefreshTokenReplay(t *testing.T) {
	if IsRefreshTokenReplay(&RefreshTokenRecord{}) {
		t.Error("未使用は replay ではない")
	}
	if !IsRefreshTokenReplay(&RefreshTokenRecord{Rotated: true}) {
		t.Error("Rotated は replay")
	}
	if !IsRefreshTokenReplay(&RefreshTokenRecord{Revoked: true}) {
		t.Error("Revoked は replay")
	}
}

func TestIsRefreshTokenAbsoluteExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := &RefreshTokenRecord{AbsoluteExpiresAt: now.Add(time.Hour)}
	if IsRefreshTokenAbsoluteExpired(rec, now) {
		t.Error("期限内なのに期限切れ判定")
	}
	if !IsRefreshTokenAbsoluteExpired(rec, now.Add(2*time.Hour)) {
		t.Error("期限後なのに有効判定")
	}
	past := &RefreshTokenRecord{AbsoluteExpiresAt: time.Now().Add(-time.Hour)}
	if !IsRefreshTokenAbsoluteExpired(past, time.Time{}) {
		t.Error("ゼロ now で過去期限が期限切れ判定されない")
	}
}

// --- client secret ---

func TestHashClientSecret_Stable(t *testing.T) {
	h := HashClientSecret("s3cr3t")
	if h != HashClientSecret("s3cr3t") {
		t.Error("同一入力のハッシュ不一致")
	}
	if len(h) != 64 {
		t.Errorf("hex 長が不正: %d", len(h))
	}
}

func TestVerifyClientSecret(t *testing.T) {
	hash := HashClientSecret("correct-secret")
	if !VerifyClientSecret("correct-secret", hash) {
		t.Error("正しいシークレットが検証失敗")
	}
	if VerifyClientSecret("wrong-secret", hash) {
		t.Error("誤ったシークレットが検証成功")
	}
	// 長さ不一致 (ハッシュでない生の値) は即 false。
	if VerifyClientSecret("correct-secret", "short") {
		t.Error("長さ不一致で true を返した")
	}
}
