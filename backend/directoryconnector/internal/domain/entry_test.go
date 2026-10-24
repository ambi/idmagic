package domain

import (
	"errors"
	"slices"
	"testing"
)

func entry(dn string, attributes ...any) Entry {
	e := Entry{DN: dn}
	for i := 0; i < len(attributes); i += 2 {
		attribute := Attribute{Name: attributes[i].(string)}
		switch values := attributes[i+1].(type) {
		case string:
			attribute.Values = [][]byte{[]byte(values)}
		case []byte:
			attribute.Values = [][]byte{values}
		case []string:
			for _, v := range values {
				attribute.Values = append(attribute.Values, []byte(v))
			}
		}
		e.Attributes = append(e.Attributes, attribute)
	}
	return e
}

// objectGUID {6f9619ff-8b86-d011-b42d-00c04fc964ff} の通信上の 16 バイト。
var guid = []byte{0xff, 0x19, 0x96, 0x6f, 0x86, 0x8b, 0x11, 0xd0, 0xb4, 0x2d, 0x00, 0xc0, 0x4f, 0xc9, 0x64, 0xff}

//spec:covers REQ-DIRECTORYCONNECTOR-004: AD のデフォルトの対応で、objectGUID の base64 を不変 ID にし、ACCOUNTDISABLE を無効として写し、多値の最初の値を使い、値のない属性を空にする。
func TestMapEntryFromActiveDirectory(t *testing.T) {
	mapping, err := EffectiveMapping(ActiveDirectory, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := MapEntry(ActiveDirectory, mapping, entry("CN=Alice,OU=Staff",
		"objectGUID", guid, "sAMAccountName", "alice", "displayName", "Alice Liddell",
		"givenName", "Alice", "SN", "Liddell", "mail", []string{"alice@corp.example", "a@corp.example"},
		"employeeNumber", "E-1", "userAccountControl", "514"))
	if err != nil {
		t.Fatal(err)
	}
	want := Item{
		ExternalID: "/xmWb4aLEdC0LQDAT8lk/w==", Enabled: false, PreferredUsername: "alice",
		Name: "Alice Liddell", GivenName: "Alice", FamilyName: "Liddell", Email: "alice@corp.example", EmployeeNumber: "E-1",
	}
	if got != want {
		t.Fatalf("item = %+v, want %+v", got, want)
	}

	enabled, err := MapEntry(ActiveDirectory, mapping, entry("CN=Bob", "objectGUID", guid, "sAMAccountName", "bob", "userAccountControl", "512"))
	if err != nil || !enabled.Enabled {
		t.Fatalf("userAccountControl 512: item=%+v err=%v, want enabled", enabled, err)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-004: OpenLDAP のデフォルトの対応と上書きした対応で、entryUUID を不変 ID にして常に有効として写す。
func TestMapEntryFromOpenLDAP(t *testing.T) {
	mapping, err := EffectiveMapping(OpenLDAP, map[string]string{"email": "mailAlternateAddress"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := MapEntry(OpenLDAP, mapping, entry("uid=carol,ou=people",
		"entryUUID", "a51b5086-63a5-1041-89ef-19b4010f7284", "uid", "carol", "cn", "Carol",
		"mail", "ignored@example.test", "mailAlternateAddress", "carol@example.test", "departmentNumber", "R&D"))
	if err != nil {
		t.Fatal(err)
	}
	want := Item{
		ExternalID: "a51b5086-63a5-1041-89ef-19b4010f7284", Enabled: true, PreferredUsername: "carol",
		Name: "Carol", Email: "carol@example.test", Department: "R&D",
	}
	if got != want {
		t.Fatalf("item = %+v, want %+v", got, want)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-004: 不変 ID かユーザー名の属性のないエントリを、欠けた属性の名前とともに拒否する。
func TestMapEntryRefusesAnEntryWithoutIdentity(t *testing.T) {
	mapping, _ := EffectiveMapping(ActiveDirectory, nil)
	cases := map[string]struct {
		entry   Entry
		missing string
	}{
		"no objectGUID":      {entry("CN=X", "sAMAccountName", "x", "userAccountControl", "512"), "objectGUID"},
		"short objectGUID":   {entry("CN=X", "objectGUID", guid[:15], "sAMAccountName", "x", "userAccountControl", "512"), "objectGUID"},
		"no sAMAccountName":  {entry("CN=X", "objectGUID", guid, "userAccountControl", "512"), "sAMAccountName"},
		"no account control": {entry("CN=X", "objectGUID", guid, "sAMAccountName", "x"), "userAccountControl"},
	}
	for name, c := range cases {
		_, err := MapEntry(ActiveDirectory, mapping, c.entry)
		var incomplete *IncompleteEntryError
		if !errors.As(err, &incomplete) || incomplete.Attribute != c.missing || incomplete.DN != "CN=X" {
			t.Errorf("%s: error = %v, want missing %s", name, err, c.missing)
		}
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-004: 値を持たない属性を、属性がないものとして扱う。
func TestMapEntryTreatsAnAttributeWithoutValuesAsMissing(t *testing.T) {
	mapping, _ := EffectiveMapping(OpenLDAP, nil)
	e := entry("uid=x", "entryUUID", "u-1", "uid", "x")
	e.Attributes = append(e.Attributes, Attribute{Name: "mail"})
	got, err := MapEntry(OpenLDAP, mapping, e)
	if err != nil || got.Email != "" {
		t.Fatalf("item=%+v err=%v, want an empty email", got, err)
	}
	e.Attributes[0].Values = nil
	var incomplete *IncompleteEntryError
	if _, err := MapEntry(OpenLDAP, mapping, e); !errors.As(err, &incomplete) || incomplete.Attribute != "entryUUID" {
		t.Fatalf("error = %v, want a missing entryUUID", err)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-003, REQ-DIRECTORYCONNECTOR-004: 方言のデフォルトのフィルターは仕様の値であり、検索は不変 ID、AD の無効の判定、対応付けの属性だけを要求する。
func TestDefaultsAndRequestedAttributesFollowTheDialect(t *testing.T) {
	if got := DefaultUserFilter(ActiveDirectory); got != "(&(objectCategory=person)(objectClass=user))" {
		t.Fatalf("active directory filter = %s", got)
	}
	if got := DefaultUserFilter(OpenLDAP); got != "(objectClass=inetOrgPerson)" {
		t.Fatalf("openldap filter = %s", got)
	}
	ad, _ := EffectiveMapping(ActiveDirectory, nil)
	want := []string{"objectGUID", "userAccountControl", "department", "displayName", "employeeNumber", "givenName", "mail", "sAMAccountName", "sn"}
	if got := RequestedAttributes(ActiveDirectory, ad); !slices.Equal(got, want) {
		t.Fatalf("active directory attributes = %v, want %v", got, want)
	}
	ldap, _ := EffectiveMapping(OpenLDAP, nil)
	want = []string{"entryUUID", "cn", "departmentNumber", "employeeNumber", "givenName", "mail", "sn", "uid"}
	if got := RequestedAttributes(OpenLDAP, ldap); !slices.Equal(got, want) {
		t.Fatalf("openldap attributes = %v, want %v", got, want)
	}
}

func TestEffectiveMappingRefusesAnUnknownAttribute(t *testing.T) {
	if _, err := EffectiveMapping(OpenLDAP, map[string]string{"phone": "telephoneNumber"}); err == nil {
		t.Fatal("unknown canonical attribute accepted")
	}
}
