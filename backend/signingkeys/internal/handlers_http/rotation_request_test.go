package handlers_http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//spec:covers REQ-SIGNINGKEYS-006: 共有 XML の既定は準備とし、有効化には kid と登録済みの確認を要すること。
func TestRotationRequest(t *testing.T) {
	for _, tc := range []struct {
		body, action, scope string
		invalid             bool
	}{
		{body: "", action: "rotate", scope: "default"},
		{body: `{}`, action: "rotate", scope: "default"},
		{body: `{"usage":"XmlFederationSigning"}`, action: "prepare", scope: "default"},
		{body: `{"usage":"XmlFederationSigning","scope_id":"profile"}`, action: "rotate", scope: "profile"},
		{body: `{"usage":"XmlFederationSigning","action":"activate","kid":"next","trust_updated":true}`, action: "activate", scope: "default"},
		{body: `{"usage":"XmlFederationSigning","action":"activate","kid":"next"}`, invalid: true},
		{body: `{"usage":"XmlFederationSigning","action":"activate","trust_updated":true}`, invalid: true},
		{body: `{"action":"activate","kid":"next","trust_updated":true}`, invalid: true},
		{body: `{"action":"prepare"}`, invalid: true},
		{body: `{"usage":"XmlFederationSigning","action":"prepare","kid":"next"}`, invalid: true},
		{body: `{"usage":"XmlFederationSigning","action":"prepare","trust_updated":true}`, invalid: true},
		{body: `{"kid":"next"}`, invalid: true},
		{body: `{"trust_updated":true}`, invalid: true},
		{body: `{"usage":"Signing","scope_id":"profile"}`, invalid: true},
		{body: `{"usage":"XmlFederationSigning","scope_id":" default "}`, invalid: true},
		{body: `{"usage":"unknown"}`, invalid: true},
		{body: `{"action":"unknown"}`, invalid: true},
		{body: `{"usage":`, invalid: true},
		{body: `{}` + strings.Repeat(" ", maxRotationBodyBytes), invalid: true},
		{body: `{}` + strings.Repeat(" ", maxRotationBodyBytes-2), action: "rotate", scope: "default"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			got, err := rotationRequest(req)
			if tc.invalid {
				if err == nil {
					t.Fatalf("invalid request accepted: %+v", got)
				}
				return
			}
			if err != nil || got.Action != tc.action || got.ScopeID != tc.scope {
				t.Fatalf("request=%+v err=%v", got, err)
			}
		})
	}
}

//spec:covers REQ-SIGNINGKEYS-006: 正しい XML の有効化は受理し、同じ kid でも確認を外した要求は必ず拒否すること。
func FuzzRotationConfirmation(f *testing.F) {
	f.Add("candidate")
	f.Add("日本語\"\\")
	f.Fuzz(func(t *testing.T, kid string) {
		if kid == "" {
			return
		}
		body := map[string]any{"usage": "XmlFederationSigning", "action": "activate", "kid": kid, "trust_updated": true}
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		if len(payload) > maxRotationBodyBytes {
			return
		}
		parse := func(payload []byte) error {
			_, err := rotationRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(payload))))
			return err
		}
		if err := parse(payload); err != nil {
			t.Fatalf("valid confirmation rejected: %v", err)
		}
		body["trust_updated"] = false
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		if err := parse(payload); err == nil {
			t.Fatal("activation accepted without trust confirmation")
		}
	})
}
