package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func items(n int) []UserItem {
	result := make([]UserItem, n)
	for i := range result {
		result[i] = UserItem{ExternalID: fmt.Sprintf("g%d", i), PreferredUsername: fmt.Sprintf("u%d", i)}
	}
	return result
}

//spec:covers REQ-SOURCING-016: バッチの ID、項目の数、外部の不変 ID の長さは上限まで受け付け、一つ超えると拒否し、重複と削除でない項目のユーザー名の欠落を拒否する。
func TestValidateUserBatchAcceptsUpToTheLimits(t *testing.T) {
	longest := strings.Repeat("あ", MaxExternalIDLength)
	accepted := map[string]struct {
		batchID string
		items   []UserItem
	}{
		"100 items":                   {"b", items(MaxBatchItems)},
		"128 character batch id":      {strings.Repeat("い", MaxBatchIDLength), items(1)},
		"256 character external id":   {"b", []UserItem{{ExternalID: longest, PreferredUsername: "u"}}},
		"deleted item with no fields": {"b", []UserItem{{ExternalID: "g", Deleted: true}}},
	}
	for name, c := range accepted {
		if err := ValidateUserBatch(c.batchID, c.items); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	refused := map[string]struct {
		batchID string
		items   []UserItem
	}{
		"101 items":                  {"b", items(MaxBatchItems + 1)},
		"no items":                   {"b", nil},
		"empty batch id":             {"", items(1)},
		"129 character batch id":     {strings.Repeat("い", MaxBatchIDLength+1), items(1)},
		"empty external id":          {"b", []UserItem{{PreferredUsername: "u"}}},
		"257 character external id":  {"b", []UserItem{{ExternalID: longest + "a", PreferredUsername: "u"}}},
		"duplicate external id":      {"b", []UserItem{{ExternalID: "g", PreferredUsername: "a"}, {ExternalID: "g", PreferredUsername: "b"}}},
		"missing preferred_username": {"b", []UserItem{{ExternalID: "g"}}},
	}
	for name, c := range refused {
		if err := ValidateUserBatch(c.batchID, c.items); !errors.Is(err, ErrInvalidBatch) {
			t.Errorf("%s: error = %v, want ErrInvalidBatch", name, err)
		}
	}
}

//spec:covers REQ-SOURCING-017: 項目のダイジェストは同じ項目で一致し、どの属性が違っても異なる。
func TestBatchDigestDistinguishesEveryField(t *testing.T) {
	base := UserItem{
		ExternalID: "g", Enabled: true, PreferredUsername: "u", Name: "n", GivenName: "g", FamilyName: "f",
		Email: "e", EmployeeNumber: "1", Department: "d",
	}
	first, again := []UserItem{base}, []UserItem{base}
	if BatchDigest(first) != BatchDigest(again) {
		t.Fatal("the same items produced different digests")
	}
	variants := []func(*UserItem){
		func(i *UserItem) { i.ExternalID = "x" }, func(i *UserItem) { i.Deleted = true }, func(i *UserItem) { i.Enabled = false },
		func(i *UserItem) { i.PreferredUsername = "x" }, func(i *UserItem) { i.Name = "x" }, func(i *UserItem) { i.GivenName = "x" },
		func(i *UserItem) { i.FamilyName = "x" }, func(i *UserItem) { i.Email = "x" }, func(i *UserItem) { i.EmployeeNumber = "x" },
		func(i *UserItem) { i.Department = "x" },
	}
	for n, change := range variants {
		changed := base
		change(&changed)
		if BatchDigest([]UserItem{changed}) == BatchDigest([]UserItem{base}) {
			t.Errorf("variant %d kept the digest", n)
		}
	}
	if BatchDigest([]UserItem{base, {ExternalID: "h"}}) == BatchDigest([]UserItem{{ExternalID: "h"}, base}) {
		t.Error("reordered items kept the digest")
	}
}

//spec:covers REQ-SOURCING-019: 記録したバッチの結果は、バッチの数、項目の数、結果ごとの数へ一件ずつ足される。
func TestRunCountsAddEachOutcome(t *testing.T) {
	counts := RunCounts{Disabled: 7}.Add([]ItemResult{
		{Outcome: OutcomeCreated},
		{Outcome: OutcomeUpdated},
		{Outcome: OutcomeUpdated},
		{Outcome: OutcomeUnchanged},
		{Outcome: OutcomeDeleted},
		{Outcome: OutcomeFailed, Error: ItemErrorQuotaExceeded},
	}).Add([]ItemResult{{Outcome: OutcomeCreated}})
	want := RunCounts{Batches: 2, Items: 7, Created: 2, Updated: 2, Unchanged: 1, Deleted: 1, Failed: 1, Disabled: 7}
	if counts != want {
		t.Fatalf("counts = %+v, want %+v", counts, want)
	}
}

//spec:covers REQ-SOURCING-015: 実行の種類は full と incremental だけを受け付ける。
func TestParseIngestionModeAcceptsOnlyTheTwoModes(t *testing.T) {
	for _, mode := range []IngestionMode{ModeFull, ModeIncremental} {
		if got, err := ParseIngestionMode(string(mode)); err != nil || got != mode {
			t.Errorf("ParseIngestionMode(%q) = %q, %v", mode, got, err)
		}
	}
	for _, value := range []string{"", "Full", "partial"} {
		if _, err := ParseIngestionMode(value); !errors.Is(err, ErrInvalidMode) {
			t.Errorf("ParseIngestionMode(%q) error = %v", value, err)
		}
	}
}
