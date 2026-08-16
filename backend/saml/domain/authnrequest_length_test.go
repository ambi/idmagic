package domain_test

import (
	"strings"
	"testing"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func authnRequestWithID(id string) string {
	return strings.Replace(sampleAuthnRequest, `ID="_req-1"`, `ID="`+id+`"`, 1)
}

// AuthnRequest の ID は replay 表の主キーの成分になる。上限が無いと、相手が
// 決めた長さのまま btree の索引行上限で落ちる。
func TestParseAuthnRequestRejectsOversizedID(t *testing.T) {
	id := "_" + strings.Repeat("a", spec.LengthProtocolMessageID)
	if _, err := samldomain.ParseAuthnRequest([]byte(authnRequestWithID(id))); err == nil {
		t.Fatalf("AuthnRequest ID of %d characters must be rejected", len(id))
	}
}

func TestParseAuthnRequestAcceptsIDAtTheCeiling(t *testing.T) {
	id := "_" + strings.Repeat("a", spec.LengthProtocolMessageID-1)
	if _, err := samldomain.ParseAuthnRequest([]byte(authnRequestWithID(id))); err != nil {
		t.Fatalf("AuthnRequest ID at the ceiling rejected: %v", err)
	}
}

// この接点の応答形は SAML が決めており、Problem Details ではない。長さ違反を
// 型付きの LengthError のまま返すと、大域のエラー写像が 422 の Problem Details を
// 作ってしまう。
func TestParseAuthnRequestLengthErrorIsNotAFieldLengthViolation(t *testing.T) {
	id := "_" + strings.Repeat("a", spec.LengthProtocolMessageID)
	_, err := samldomain.ParseAuthnRequest([]byte(authnRequestWithID(id)))
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := err.(interface{ IsFieldLengthViolation() bool }); ok {
		t.Fatalf("protocol endpoint must not return the admin API's typed length error: %T", err)
	}
}
