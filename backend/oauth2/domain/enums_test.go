package domain

import "testing"

// wi-129 の enum Valid() カバレッジのうち client/consent/認可詳細タイプ系を移設 (wi-173)。

type enumValue interface{ Valid() bool }

func TestEnumValid(t *testing.T) {
	cases := []struct {
		name string
		v    enumValue
		want bool
	}{
		{"authmethod basic", AuthMethodClientSecretBasic, true},
		{"authmethod post", AuthMethodClientSecretPost, true},
		{"authmethod private key jwt", AuthMethodPrivateKeyJwt, true},
		{"authmethod tls", AuthMethodTlsClientAuth, true},
		{"authmethod none", AuthMethodNone, true},
		{"authmethod bad", TokenEndpointAuthMethod("x"), false},

		{"fapi none", FapiNone, true},
		{"fapi v2", FapiSecurityProfileV2, true},
		{"fapi bad", FapiProfile("x"), false},

		{"consent granted", ConsentGranted, true},
		{"consent revoked", ConsentRevoked, true},
		{"consent expired", ConsentExpired, true},
		{"consent bad", ConsentState("x"), false},

		{"detail field set", DetailFieldSet, true},
		{"detail field at_most", DetailFieldAtMost, true},
		{"detail field enum", DetailFieldEnum, true},
		{"detail field exact", DetailFieldExact, true},
		{"detail field bad", AuthorizationDetailFieldSemantics("x"), false},

		{"detail type enabled", DetailTypeEnabled, true},
		{"detail type disabled", DetailTypeDisabled, true},
		{"detail type bad", AuthorizationDetailTypeState("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Errorf("%s: Valid()=%v, want %v", c.name, got, c.want)
			}
		})
	}
}
