package domain

import (
	"reflect"
	"testing"
)

// 方式を取り違えた行（DPoP の jkt と同じ値の証明書）と鍵の空の制約を置かないと、
// 値だけを比べる実装や、空同士を一致とみなす実装と区別できない。
//
//spec:covers REQ-OAUTH2-056: 表の各行について、発行する送信者制約か拒否を返し、両方の証明では DPoP を優先する。
func TestDecideExchangeBinding(t *testing.T) {
	dpop := func(jkt string) *SenderConstraint { return &SenderConstraint{Type: SenderConstraintDPoP, JKT: jkt} }
	mtls := func(x5t string) *SenderConstraint { return &SenderConstraint{Type: SenderConstraintMTLS, X5TS256: x5t} }
	for _, tc := range []struct {
		name    string
		subject *SenderConstraint
		proof   ExchangeProof
		want    *SenderConstraint
		wantOK  bool
	}{
		{name: "制約なし、証明なし", wantOK: true},
		{name: "制約なし、DPoP", proof: ExchangeProof{JKT: "k2"}, want: dpop("k2"), wantOK: true},
		{name: "制約なし、mTLS", proof: ExchangeProof{X5TS256: "c2"}, want: mtls("c2"), wantOK: true},
		{name: "制約なし、両方なら DPoP", proof: ExchangeProof{JKT: "k2", X5TS256: "c2"}, want: dpop("k2"), wantOK: true},
		{name: "DPoP、同じ鍵", subject: dpop("k1"), proof: ExchangeProof{JKT: "k1"}, want: dpop("k1"), wantOK: true},
		{name: "DPoP、同じ鍵と別の証明書", subject: dpop("k1"), proof: ExchangeProof{JKT: "k1", X5TS256: "c2"}, want: dpop("k1"), wantOK: true},
		{name: "mTLS、同じ証明書", subject: mtls("c1"), proof: ExchangeProof{X5TS256: "c1"}, want: mtls("c1"), wantOK: true},
		{name: "mTLS、同じ証明書と別の DPoP", subject: mtls("c1"), proof: ExchangeProof{JKT: "k2", X5TS256: "c1"}, want: mtls("c1"), wantOK: true},
		{name: "DPoP、別の鍵", subject: dpop("k1"), proof: ExchangeProof{JKT: "k2"}},
		{name: "DPoP、証明なし", subject: dpop("k1")},
		{name: "DPoP、同じ値の証明書", subject: dpop("k1"), proof: ExchangeProof{X5TS256: "k1"}},
		{name: "mTLS、別の証明書", subject: mtls("c1"), proof: ExchangeProof{X5TS256: "c2"}},
		{name: "mTLS、証明なし", subject: mtls("c1")},
		{name: "mTLS、同じ値の DPoP", subject: mtls("c1"), proof: ExchangeProof{JKT: "c1"}},
		{name: "鍵の空の DPoP 制約", subject: dpop(""), proof: ExchangeProof{X5TS256: "c2"}},
		{name: "鍵の空の mTLS 制約", subject: mtls(""), proof: ExchangeProof{JKT: "k2"}},
		{name: "種別の無い制約", subject: &SenderConstraint{JKT: "k1"}, proof: ExchangeProof{JKT: "k1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DecideExchangeBinding(tc.subject, tc.proof)
			if ok != tc.wantOK || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DecideExchangeBinding=(%+v, %v), want (%+v, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
