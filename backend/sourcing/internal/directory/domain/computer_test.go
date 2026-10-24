package domain

import (
	"errors"
	"strings"
	"testing"
)

const testSID = "S-1-5-21-1004336348-1177238915-682003330-1101"

func validComputer() ComputerItem {
	return ComputerItem{ExternalID: "g", Enabled: true, SAMAccountName: "WS1$", ObjectSID: testSID, Domain: "corp.example.test"}
}

//spec:covers REQ-SOURCING-020: コンピューターの値は上限の長さまで受け付け、$ で終わらない名前、SID の形でない値、空のドメイン、長すぎる値、重複する外部の不変 ID を拒否する。
func TestValidateComputerBatchChecksEveryValue(t *testing.T) {
	with := func(change func(*ComputerItem)) []ComputerItem {
		item := validComputer()
		change(&item)
		return []ComputerItem{item}
	}
	accepted := map[string][]ComputerItem{
		"minimal":              {validComputer()},
		"longest name":         with(func(c *ComputerItem) { c.SAMAccountName = strings.Repeat("あ", MaxSAMAccountNameLength-1) + "$" }),
		"longest domain":       with(func(c *ComputerItem) { c.Domain = strings.Repeat("d", MaxDomainLength) }),
		"longest host name":    with(func(c *ComputerItem) { c.DNSHostName = strings.Repeat("h", MaxDomainLength) }),
		"longest sid":          with(func(c *ComputerItem) { c.ObjectSID = "S-1-55" + strings.Repeat("-1", (MaxObjectSIDLength-6)/2) }),
		"well-known sid":       with(func(c *ComputerItem) { c.ObjectSID = "S-1-5-18" }),
		"disabled with values": with(func(c *ComputerItem) { c.Enabled = false }),
	}
	for name, batch := range accepted {
		if err := ValidateComputerBatch("b", batch); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	refused := map[string][]ComputerItem{
		"no items":          nil,
		"empty external id": with(func(c *ComputerItem) { c.ExternalID = "" }),
		"duplicate":         {validComputer(), validComputer()},
		"name without $":    with(func(c *ComputerItem) { c.SAMAccountName = "WS1" }),
		"name too long":     with(func(c *ComputerItem) { c.SAMAccountName = strings.Repeat("あ", MaxSAMAccountNameLength) + "$" }),
		"lowercase sid":     with(func(c *ComputerItem) { c.ObjectSID = strings.ToLower(testSID) }),
		"sid without rid":   with(func(c *ComputerItem) { c.ObjectSID = "S-1-5" }),
		"sid with letters":  with(func(c *ComputerItem) { c.ObjectSID = "S-1-5-21-x" }),
		"sid revision 2":    with(func(c *ComputerItem) { c.ObjectSID = "S-2-5-21" }),
		"sid too long":      with(func(c *ComputerItem) { c.ObjectSID = "S-1-555" + strings.Repeat("-1", (MaxObjectSIDLength-6)/2) }),
		"empty domain":      with(func(c *ComputerItem) { c.Domain = "" }),
		"domain too long":   with(func(c *ComputerItem) { c.Domain = strings.Repeat("d", MaxDomainLength+1) }),
		"host too long":     with(func(c *ComputerItem) { c.DNSHostName = strings.Repeat("h", MaxDomainLength+1) }),
	}
	for name, batch := range refused {
		if err := ValidateComputerBatch("b", batch); !errors.Is(err, ErrInvalidBatch) {
			t.Errorf("%s: error = %v, want ErrInvalidBatch", name, err)
		}
	}
	if err := ValidateComputerBatch("", []ComputerItem{validComputer()}); !errors.Is(err, ErrInvalidBatch) {
		t.Errorf("empty batch id: error = %v", err)
	}
}

//spec:covers REQ-SOURCING-017, REQ-SOURCING-020: コンピューターのダイジェストはどの値が違っても異なり、同じ形の User のバッチのダイジェストとも異なる。
func TestComputerBatchDigestDistinguishesEveryFieldAndTheKind(t *testing.T) {
	base := validComputer()
	base.DNSHostName = "ws1.corp.example.test"
	digest := ComputerBatchDigest([]ComputerItem{base})
	if digest != ComputerBatchDigest([]ComputerItem{base}) {
		t.Fatal("the same items gave different digests")
	}
	for name, change := range map[string]func(*ComputerItem){
		"external_id": func(c *ComputerItem) { c.ExternalID = "h" },
		"enabled":     func(c *ComputerItem) { c.Enabled = false },
		"sam":         func(c *ComputerItem) { c.SAMAccountName = "WS2$" },
		"sid":         func(c *ComputerItem) { c.ObjectSID = "S-1-5-18" },
		"domain":      func(c *ComputerItem) { c.Domain = "other.test" },
		"host":        func(c *ComputerItem) { c.DNSHostName = "" },
	} {
		changed := base
		change(&changed)
		if ComputerBatchDigest([]ComputerItem{changed}) == digest {
			t.Errorf("%s does not change the digest", name)
		}
	}
	if ComputerBatchDigest([]ComputerItem{{ExternalID: "g", Enabled: true}}) == BatchDigest([]UserItem{{ExternalID: "g", Enabled: true}}) {
		t.Fatal("a computer batch digest equals a user batch digest")
	}
}

//spec:covers REQ-SOURCING-020: 有効なコンピューターだけが、同じ SID か大文字と小文字を区別しない同じ主体名の項目の持ち主になる。
func TestEnabledComputerHoldsItsSIDAndPrincipalName(t *testing.T) {
	held := Computer{ComputerItem: validComputer()}
	sameName := ComputerItem{SAMAccountName: "ws1$", ObjectSID: "S-1-5-21-1-2-3-9", Domain: "CORP.EXAMPLE.TEST"}
	sameSID := ComputerItem{SAMAccountName: "OTHER$", ObjectSID: testSID, Domain: "other.test"}
	unrelated := ComputerItem{SAMAccountName: "WS1$", ObjectSID: "S-1-5-21-1-2-3-9", Domain: "other.test"}
	if !held.Holds(sameName) || !held.Holds(sameSID) || held.Holds(unrelated) {
		t.Fatalf("holds: name=%v sid=%v unrelated=%v", held.Holds(sameName), held.Holds(sameSID), held.Holds(unrelated))
	}
	held.Enabled = false
	if held.Holds(sameName) || held.Holds(sameSID) {
		t.Fatal("a disabled computer holds a value")
	}
}
