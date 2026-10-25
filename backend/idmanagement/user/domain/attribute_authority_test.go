package domain

import (
	"slices"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

//spec:covers REQ-IDMANAGEMENT-091: User と値がなくてもローカル専用だけを区別し、省略、未知のキーと未知の分類を上流から更新可能として扱う。
func TestAttributeWriteAuthorityWithoutUserValues(t *testing.T) {
	defs := []UserAttributeDef{
		{Key: "approval", WriteAuthority: AttributeAuthorityLocal},
		{Key: "legacy"},
		{Key: "unrecognized", WriteAuthority: "invalid"},
	}
	keys := []string{"approval", "legacy", "unrecognized", "missing", "name"}
	want := []string{"legacy", "unrecognized", "missing", "name"}
	if got := UpstreamWritableAttributes(keys, defs); !slices.Equal(got, want) {
		t.Fatalf("upstream=%v, want %v", got, want)
	}
	def := UserAttributeDef{Key: "approval", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable, WriteAuthority: "invalid"}
	if err := def.Validate(); err == nil {
		t.Fatal("unknown write authority was accepted")
	}
	for _, authority := range []AttributeWriteAuthority{"", AttributeAuthorityUpstream, AttributeAuthorityLocal} {
		def.WriteAuthority = authority
		if err := def.Validate(); err != nil {
			t.Fatalf("authority=%q: %v", authority, err)
		}
	}
}

//spec:covers REQ-IDMANAGEMENT-091: ローカル専用の値は追加、削除と置換を拒否し、同じ値の保持は許す。
func TestUpstreamAttributeChangeBothSides(t *testing.T) {
	defs := []UserAttributeDef{{Key: "approval", WriteAuthority: AttributeAuthorityLocal}}
	value := func(s string) map[string]AttributeValue {
		return map[string]AttributeValue{"approval": {Type: idmdomain.AttributeTypeString, String: new(s)}}
	}
	for _, tc := range []struct {
		name          string
		before, after map[string]AttributeValue
		refused       bool
	}{
		{"add", nil, value("yes"), true},
		{"remove", value("yes"), nil, true},
		{"replace", value("yes"), value("no"), true},
		{"preserve", value("yes"), value("yes"), false},
		{"absent", nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateUpstreamAttributeChanges(tc.before, tc.after, defs); (err != nil) != tc.refused {
				t.Fatalf("err=%v, want refused=%v", err, tc.refused)
			}
		})
	}
}
