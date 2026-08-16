package spec

import (
	"errors"
	"strings"
	"testing"

	z "github.com/Oudwins/zog"
)

type keySubject struct{ EntityID string }

// KeyString を実際の適用点と同じ形 (struct schema 経由) で通す。
func parseEntityID(value string) error {
	subject := keySubject{EntityID: value}
	return Validate(z.Struct(z.Shape{"EntityID": KeyString(LengthSamlEntityID, BytesSamlEntityID)}), &subject)
}

// indexKey は 1 つの btree の鍵と、その成分が使ってよいバイト数。
// 正本は spec/SPECIFICATION.md の "String length limits"。
type indexKey struct {
	name       string
	components []int
}

// PostgreSQL は複合鍵の索引行を成分ごとではなく 1 件として制限するので、上限は
// 鍵の合計で守らなければならない。UUID は索引上 16 バイトを占める。
const uuidIndexBytes = 16

func indexKeys() []indexKey {
	return []indexKey{
		{"saml_service_providers (tenant_id, entity_id)", []int{uuidIndexBytes, BytesSamlEntityID}},
		{
			"saml_authnrequest_replays (tenant_id, entity_id, request_id)",
			[]int{uuidIndexBytes, BytesSamlEntityID, BytesProtocolMessageID},
		},
		{"wsfed_relying_parties (tenant_id, wtrealm)", []int{uuidIndexBytes, BytesWtrealm}},
		{
			"federated_identities (tenant_id, provider_id, external_subject)",
			[]int{uuidIndexBytes, BytesGeneratedKeyComponent * 2, BytesFederatedSubject},
		},
		{"federated_response_replays (tenant_id, response_id)", []int{uuidIndexBytes, BytesProtocolMessageID}},
		{"oauth2_replay_jtis (tenant_id, kind, jti)", []int{uuidIndexBytes, BytesGeneratedKeyComponent, BytesProtocolMessageID}},
		{"oauth2_access_token_denylist (tenant_id, jti)", []int{uuidIndexBytes, BytesGeneratedKeyComponent}},
		{"webauthn_credentials (credential_id)", []int{BytesWebAuthnCredentialID}},
		{"scim_user_refs (tenant_id, scim_id)", []int{uuidIndexBytes, BytesGeneratedKeyComponent}},
		{"scim_group_refs (tenant_id, scim_id)", []int{uuidIndexBytes, BytesGeneratedKeyComponent}},
		{"signing_keys (kid)", []int{BytesGeneratedKeyComponent}},
	}
}

// 上限を後から広げたときに、btree の索引行上限を割ったことがここで分かる。
// これを散文ではなくテストに置いているのは、鍵の予算が個々の数ではなく合計で
// 決まり、列を 1 つ足すだけで静かに壊れるためである。
func TestIndexKeysFitInBtreeRowLimit(t *testing.T) {
	if KeyByteBudget >= BtreeIndexRowLimitBytes {
		t.Fatalf("key budget %d must stay inside the btree row limit %d", KeyByteBudget, BtreeIndexRowLimitBytes)
	}
	for _, key := range indexKeys() {
		total := 0
		for _, component := range key.components {
			total += component
		}
		if total > KeyByteBudget {
			t.Errorf("%s: key byte budget exceeded: %d > %d", key.name, total, KeyByteBudget)
		}
	}
}

// 契約の上限 (コードポイント) は、資源の上限 (バイト) を UTF-8 の最悪値で
// 上回ることがある。そのときに止めるのは資源の上限なので、両方が課されて
// いなければならない。
func TestKeyStringAppliesBothCeilings(t *testing.T) {
	// 2048 コードポイントちょうどの ASCII は両方の上限の内側。
	value := strings.Repeat("a", LengthSamlEntityID)
	if err := parseEntityID(value); err != nil {
		t.Fatalf("entity id of %d ASCII characters must be accepted: %v", LengthSamlEntityID, err)
	}

	// 1024 コードポイントの日本語は 3072 バイトで、契約の上限の内側だが
	// btree の索引行には収まらない。資源の上限が止める。
	multibyte := strings.Repeat("あ", 1024)
	err := parseEntityID(multibyte)
	if err == nil {
		t.Fatalf("entity id of %d bytes must be rejected before it reaches btree", len(multibyte))
	}
	var lengthErr *LengthError
	if !errors.As(err, &lengthErr) {
		t.Fatalf("byte ceiling violation is not a *LengthError: %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "bytes") {
		t.Errorf("byte ceiling message must name the unit: %q", err.Error())
	}
}

func TestCheckKeyStringReportsTheCeilingThatFailed(t *testing.T) {
	if err := CheckKeyString("entity_id", strings.Repeat("a", BytesSamlEntityID), LengthSamlEntityID, BytesSamlEntityID); err != nil {
		t.Fatalf("value at both ceilings must be accepted: %v", err)
	}

	// コードポイントの上限を超える値は、文字数で報告する。
	err := CheckKeyString("entity_id", strings.Repeat("a", LengthSamlEntityID+1), LengthSamlEntityID, BytesSamlEntityID)
	if err == nil || !strings.Contains(err.Error(), "characters") {
		t.Fatalf("code point violation must be reported in characters: %v", err)
	}

	// コードポイントの上限の内側でバイトの上限を超える値は、バイト数で報告する。
	err = CheckKeyString("entity_id", strings.Repeat("あ", 1024), LengthSamlEntityID, BytesSamlEntityID)
	if err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("byte violation must be reported in bytes: %v", err)
	}
	var lengthErr *LengthError
	if !errors.As(err, &lengthErr) {
		t.Fatalf("byte ceiling violation is not a *LengthError: %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "entity_id") {
		t.Errorf("violation must name the wire field: %q", err.Error())
	}
}

// 標準が許す実例を拒否しないこと。上限は相互運用性を削るために置くのではない。
func TestKeyStringAcceptsConformingIdentifiers(t *testing.T) {
	for _, value := range []string{
		"https://sp.example.com/shibboleth",
		"urn:federation:MicrosoftOnline",
		// saml-schema-metadata-2.0.xsd の entityIDType が許す最大長。
		"https://sp.example.com/" + strings.Repeat("a", 1024-len("https://sp.example.com/")),
	} {
		if err := parseEntityID(value); err != nil {
			t.Errorf("conforming entity id of %d characters rejected: %v", len([]rune(value)), err)
		}
	}
}

func TestTruncateCharsCutsOnCodePointBoundaries(t *testing.T) {
	if got := TruncateChars("あいうえお", 3); got != "あいう" {
		t.Errorf("TruncateChars = %q, want %q", got, "あいう")
	}
	if got := TruncateChars("short", 100); got != "short" {
		t.Errorf("value inside the ceiling must be returned unchanged: %q", got)
	}
}
