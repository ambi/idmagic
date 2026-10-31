package domain

import (
	"net/url"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

//spec:covers REQ-TOKENVAULT-008: Agent を伴わないトークン、権限の詳細の省略と不一致を拒否すること
func TestAuthorizeCallerEvaluatesDelegationAndDetails(t *testing.T) {
	target := mustURL(t, "https://api.example.com/v1/items")
	detail := spec.AuthorizationDetail{
		Type: AuthorizationDetailType, Identifier: "c1", Actions: []string{"GET"}, Locations: []string{"https://api.example.com"},
	}
	delegated := CallerToken{Delegated: true, Subject: "u1", AgentID: "a1", AuthorizationDetails: []spec.AuthorizationDetail{detail}}
	with := func(mutate func(*spec.AuthorizationDetail)) CallerToken {
		changed := detail
		mutate(&changed)
		token := delegated
		token.AuthorizationDetails = []spec.AuthorizationDetail{changed}
		return token
	}
	cases := []struct {
		name  string
		token CallerToken
		want  string
	}{
		{"delegated", delegated, ""},
		{"chain only", CallerToken{Delegated: true, Subject: "u1", AgentChain: []string{"a1"}, AuthorizationDetails: delegated.AuthorizationDetails}, ""},
		{"not exchanged", CallerToken{Subject: "u1", AgentID: "a1", AuthorizationDetails: delegated.AuthorizationDetails}, ReasonNotDelegated},
		{"no agent", CallerToken{Delegated: true, Subject: "u1", AuthorizationDetails: delegated.AuthorizationDetails}, ReasonNotDelegated},
		{"agent principal", CallerToken{Delegated: true, Subject: "a1", PrincipalType: "agent", AgentID: "a1", AuthorizationDetails: delegated.AuthorizationDetails}, ReasonNotDelegated},
		{"no details", CallerToken{Delegated: true, Subject: "u1", AgentID: "a1"}, ReasonAuthorizationDetailMissing},
		{"other type", with(func(d *spec.AuthorizationDetail) { d.Type = "other" }), ReasonAuthorizationDetailMissing},
		{"other connector", with(func(d *spec.AuthorizationDetail) { d.Identifier = "c2" }), ReasonAuthorizationDetailMissing},
		{"empty identifier is not a wildcard", with(func(d *spec.AuthorizationDetail) { d.Identifier = "" }), ReasonAuthorizationDetailMissing},
		{"other method", with(func(d *spec.AuthorizationDetail) { d.Actions = []string{"POST"} }), ReasonAuthorizationDetailMissing},
		{"omitted actions", with(func(d *spec.AuthorizationDetail) { d.Actions = nil }), ReasonAuthorizationDetailMissing},
		{"other origin", with(func(d *spec.AuthorizationDetail) { d.Locations = []string{"https://evil.example"} }), ReasonAuthorizationDetailMissing},
		{"omitted locations", with(func(d *spec.AuthorizationDetail) { d.Locations = nil }), ReasonAuthorizationDetailMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AuthorizeCaller(tc.token, "c1", "GET", target); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

//spec:covers REQ-TOKENVAULT-009: スキーム、利用者情報、フラグメント、ドットセグメント、符号化した区切りを拒否すること
func TestParseExecutionTargetRefusesAmbiguousURLs(t *testing.T) {
	for _, raw := range []string{
		"http://api.example.com/a", "/a", "https://user:pw@api.example.com/a", "https://api.example.com/a#f",
		"https://api.example.com/a/../b", "https://api.example.com/./b", "https://api.example.com/a%2Fb",
		"https://api.example.com/a%2fb", "https://api.example.com/a%5cb", "https://api.example.com/a\\b",
		"https://api.example.com/%2e%2e/b", "https://127.0.0.1/a", "https://[::1]/a", "https://10.0.0.1/a",
		"https://169.254.169.254/latest", "https://localhost/a",
	} {
		if _, err := ParseExecutionTarget(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
	target, err := ParseExecutionTarget("https://API.example.com:443/v1/items?q=a%20b")
	if err != nil {
		t.Fatal(err)
	}
	if got := Origin(target); got != "https://api.example.com" {
		t.Fatalf("origin = %q", got)
	}
}

//spec:covers REQ-TOKENVAULT-009: 指定できないヘッダーを拒否すること
func TestCheckForwardedHeaders(t *testing.T) {
	if err := CheckForwardedHeaders(map[string]string{"accept": "a", "If-Match": "b"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Authorization", "authorization", "Cookie", "Host", "X-Forwarded-For"} {
		if err := CheckForwardedHeaders(map[string]string{name: "x"}); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

//spec:covers REQ-TOKENVAULT-009: パスの接頭辞をセグメントの境界で照合すること
func TestMatchRuleUsesSegmentBoundaries(t *testing.T) {
	connector := Connector{Rules: []ExecutionRule{{ID: "r", Method: "GET", Origin: "https://api.example.com", PathPrefix: "/v1/items"}}}
	cases := map[string]bool{
		"https://api.example.com/v1/items":      true,
		"https://api.example.com/v1/items/1":    true,
		"https://api.example.com/v1/itemsecret": false,
		"https://api.example.com/v1":            false,
		"https://api.example.com:8443/v1/items": false,
		"https://other.example.com/v1/items":    false,
	}
	for raw, want := range cases {
		if _, got := connector.MatchRule("GET", mustURL(t, raw)); got != want {
			t.Errorf("%s: got %v, want %v", raw, got, want)
		}
	}
	if _, got := connector.MatchRule("POST", mustURL(t, "https://api.example.com/v1/items")); got {
		t.Error("another method matched")
	}
}

//spec:covers REQ-TOKENVAULT-001: エンドポイントと規則の検証
func TestConnectorValidate(t *testing.T) {
	valid := func() Connector {
		return Connector{
			Name: "n", Status: ConnectorActive, ClientID: "c", ClientSecret: "s",
			TokenEndpointAuthMethod: AuthMethodClientSecretBasic,
			AuthorizationEndpoint:   "https://idp.example.com/a", TokenEndpoint: "https://idp.example.com/t",
			AllowedScopes: []string{"read"},
			Rules:         []ExecutionRule{{ID: "r", Method: "GET", Origin: "https://api.example.com", PathPrefix: "/v1", RequiredScopes: []string{"read"}}},
		}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Connector){
		"http token endpoint":    func(c *Connector) { c.TokenEndpoint = "http://idp.example.com/t" },
		"internal authorization": func(c *Connector) { c.AuthorizationEndpoint = "https://10.0.0.1/a" },
		"userinfo revocation":    func(c *Connector) { c.RevocationEndpoint = "https://u:p@idp.example.com/r" },
		"bad method":             func(c *Connector) { c.Rules[0].Method = "TRACE" },
		"relative prefix":        func(c *Connector) { c.Rules[0].PathPrefix = "v1" },
		"dotted prefix":          func(c *Connector) { c.Rules[0].PathPrefix = "/v1/../admin" },
		"origin with path":       func(c *Connector) { c.Rules[0].Origin = "https://api.example.com/v1" },
		"unnormalized origin":    func(c *Connector) { c.Rules[0].Origin = "https://API.example.com" },
		"loopback origin":        func(c *Connector) { c.Rules[0].Origin = "https://localhost" },
		"unlisted scope":         func(c *Connector) { c.Rules[0].RequiredScopes = []string{"write"} },
		"duplicate rule":         func(c *Connector) { c.Rules = append(c.Rules, c.Rules[0]) },
		"empty secret":           func(c *Connector) { c.ClientSecret = "" },
		"too many rules": func(c *Connector) {
			c.Rules = make([]ExecutionRule, maxExecutionRules+1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			connector := valid()
			mutate(&connector)
			err := connector.Validate()
			if err == nil || !strings.Contains(err.Error(), "invalid connector") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

//spec:covers REQ-TOKENVAULT-013: invalid_grant だけを再認可必須へ分類すること
func TestClassifyRefreshFailure(t *testing.T) {
	if ClassifyRefreshFailure("invalid_grant") != RefreshRequiresReauthorization {
		t.Fatal("invalid_grant must require reauthorization")
	}
	for _, code := range []string{"", "invalid_client", "unauthorized_client", "temporarily_unavailable", "server_error"} {
		if ClassifyRefreshFailure(code) != RefreshTransient {
			t.Errorf("%q must be transient", code)
		}
	}
}
