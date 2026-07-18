package spec

import (
	"testing"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

// wi-129: 純粋ドメイン (enum Valid / Validate / コンストラクタ / 状態機械) のカバレッジ補強。

// ---------------------------------------------------------------
// enum Valid()
// ---------------------------------------------------------------

// enumValue は Valid() を持つ全 typed string enum を共通に扱うためのインタフェース。
type enumValue interface{ Valid() bool }

func TestEnumValid(t *testing.T) {
	cases := []struct {
		name string
		v    enumValue
		want bool
	}{
		{"client public", ClientPublic, true},
		{"client confidential", ClientConfidential, true},
		{"client bad", ClientType("x"), false},

		{"grant auth code", GrantAuthorizationCode, true},
		{"grant refresh", GrantRefreshToken, true},
		{"grant client credentials", GrantClientCredentials, true},
		{"grant device", GrantDeviceCode, true},
		{"grant token exchange", GrantTokenExchange, true},
		{"grant bad", GrantType("x"), false},

		{"response code", ResponseTypeCode, true},
		{"response bad", ResponseType("token"), false},

		{"sig ps256", signingdomain.SigAlgPS256, true},
		{"sig es256", signingdomain.SigAlgES256, true},
		{"sig bad", signingdomain.SignatureAlgorithm("RS256"), false},

		{"keyprovider local", signingdomain.KeyProviderLocal, true},
		{"keyprovider database", signingdomain.KeyProviderDatabase, true},
		{"keyprovider vault", signingdomain.KeyProviderVaultTransit, true},
		{"keyprovider bad", signingdomain.KeyProvider("x"), false},

		{"keyusage signing", signingdomain.KeyUsageSigning, true},
		{"keyusage bad", signingdomain.KeyUsage("enc"), false},

		{"cc method s256", CodeChallengeMethodS256, true},
		{"cc method bad", CodeChallengeMethod("plain"), false},

		{"mfa totp", MfaFactorTOTP, true},
		{"mfa webauthn", MfaFactorWebAuthn, true},
		{"mfa hwk", MfaFactorHWK, true},
		{"mfa swk", MfaFactorSWK, true},
		{"mfa bad", MfaFactorType("sms"), false},

		{"authflow received", AuthFlowReceived, true},
		{"authflow exchanged", AuthFlowExchanged, true},
		{"authflow bad", AuthorizationCodeFlowState("x"), false},

		{"authcode issued", AuthCodeRecordIssued, true},
		{"authcode redeemed", AuthCodeRecordRedeemed, true},
		{"authcode expired", AuthCodeRecordExpired, true},
		{"authcode bad", AuthorizationCodeRecordState("x"), false},

		{"session logout", SessionEndLogout, true},
		{"session idle", SessionEndIdle, true},
		{"session absolute", SessionEndAbsolute, true},
		{"session self revoke", SessionEndSelfRevoke, true},
		{"session admin revoke", SessionEndAdminRevoke, true},
		{"session password change", SessionEndPasswordChange, true},
		{"session mfa change", SessionEndMfaChange, true},
		{"session other", SessionEndOther, true},
		{"session bad", SessionEndReason("x"), false},

		{"device issued", DeviceFlowIssued, true},
		{"device entered", DeviceFlowUserCodeEntered, true},
		{"device approved", DeviceFlowApproved, true},
		{"device denied", DeviceFlowDenied, true},
		{"device exchanged", DeviceFlowExchanged, true},
		{"device expired", DeviceFlowExpired, true},
		{"device bad", DeviceCodeFlowState("x"), false},

		{"response mode query", ResponseModeQuery, true},
		{"response mode form_post", ResponseModeFormPost, true},
		{"response mode bad", ResponseMode("fragment"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

// Validate() happy/failure カバレッジは tenancy/domain へ移設した (wi-179, ADR-089/ADR-093)。

// ---------------------------------------------------------------
// コンストラクタ / ID 生成
// ---------------------------------------------------------------

func TestNewUUIDv4Format(t *testing.T) {
	id, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("uuid length = %d, want 36 (%q)", len(id), id)
	}
	// version 4 / RFC 4122 variant の位置を確認する。
	if id[14] != '4' {
		t.Fatalf("version nibble = %q, want 4 (%q)", id[14], id)
	}
	switch id[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Fatalf("variant nibble = %q, want 8/9/a/b (%q)", id[19], id)
	}
	// 一意性: 2 回連続で異なる。
	id2, _ := NewUUIDv4()
	if id == id2 {
		t.Fatal("two UUIDs must differ")
	}
}

// ---------------------------------------------------------------
// DeviceCodeFlow 状態機械
// ---------------------------------------------------------------

func TestTransitionDeviceCodeFlow(t *testing.T) {
	steps := []struct {
		from  DeviceCodeFlowState
		event DeviceCodeFlowEvent
		to    DeviceCodeFlowState
	}{
		{DeviceFlowIssued, DeviceEventEnterUserCode, DeviceFlowUserCodeEntered},
		{DeviceFlowUserCodeEntered, DeviceEventApprove, DeviceFlowApproved},
		{DeviceFlowApproved, DeviceEventExchange, DeviceFlowExchanged},
		{DeviceFlowUserCodeEntered, DeviceEventDeny, DeviceFlowDenied},
		{DeviceFlowIssued, DeviceEventExpire, DeviceFlowExpired},
	}
	for _, s := range steps {
		got, err := TransitionDeviceCodeFlow(s.from, s.event)
		if err != nil {
			t.Fatalf("transition %q on %q: %v", s.from, s.event, err)
		}
		if got != s.to {
			t.Fatalf("transition %q on %q: got %q want %q", s.from, s.event, got, s.to)
		}
	}
	if _, err := TransitionDeviceCodeFlow(DeviceFlowIssued, DeviceEventExchange); err == nil {
		t.Fatal("expected error: cannot exchange from issued")
	}
}

func TestIsDeviceCodeFlowTerminal(t *testing.T) {
	terminal := []DeviceCodeFlowState{DeviceFlowDenied, DeviceFlowExchanged, DeviceFlowExpired}
	for _, s := range terminal {
		if !IsDeviceCodeFlowTerminal(s) {
			t.Fatalf("%q must be terminal", s)
		}
	}
	nonTerminal := []DeviceCodeFlowState{DeviceFlowIssued, DeviceFlowUserCodeEntered, DeviceFlowApproved}
	for _, s := range nonTerminal {
		if IsDeviceCodeFlowTerminal(s) {
			t.Fatalf("%q must not be terminal", s)
		}
	}
}

func TestDefaultDeviceCodePolling(t *testing.T) {
	p := DefaultDeviceCodePolling()
	if p.DefaultIntervalSeconds != 5 || p.SlowDownIncrementSeconds != 5 {
		t.Fatalf("unexpected default polling: %+v", p)
	}
}

func TestIsAuthorizationCodeRecordTerminal(t *testing.T) {
	if !IsAuthorizationCodeRecordTerminal(AuthCodeRecordRedeemed) ||
		!IsAuthorizationCodeRecordTerminal(AuthCodeRecordExpired) {
		t.Fatal("redeemed/expired must be terminal")
	}
	if IsAuthorizationCodeRecordTerminal(AuthCodeRecordIssued) {
		t.Fatal("issued must not be terminal")
	}
}

// ---------------------------------------------------------------
// Grant matrix helpers
// ---------------------------------------------------------------

func TestGrantMatrix(t *testing.T) {
	if _, ok := GetGrantSpec(GrantAuthorizationCode); !ok {
		t.Fatal("authorization_code must be in grant spec")
	}
	if _, ok := GetGrantSpec(GrantTokenExchange); ok {
		t.Fatal("token-exchange is not in the grant matrix")
	}
	if !GrantAllowsClientType(GrantClientCredentials, ClientConfidential) {
		t.Fatal("client_credentials must allow confidential clients")
	}
	if GrantAllowsClientType(GrantClientCredentials, ClientPublic) {
		t.Fatal("client_credentials must not allow public clients")
	}
	if !GrantRequiresPKCE(GrantAuthorizationCode) {
		t.Fatal("authorization_code must require PKCE")
	}
	if GrantRequiresPKCE(GrantRefreshToken) {
		t.Fatal("refresh_token must not require PKCE")
	}
	if !GrantIssues(GrantAuthorizationCode, "id_token") {
		t.Fatal("authorization_code must issue id_token")
	}
	if GrantIssues(GrantClientCredentials, "id_token") {
		t.Fatal("client_credentials must not issue id_token")
	}
	if GrantIssues(GrantType("bogus"), "access_token") {
		t.Fatal("unknown grant issues nothing")
	}
}
