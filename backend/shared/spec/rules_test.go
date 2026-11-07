package spec

import (
	"errors"
	"testing"
	"time"
)

func TestAllReturnsNilWithoutIssues(t *testing.T) {
	if err := All(nil, Required("Name", "ok"), Optional("Note", "")); err != nil {
		t.Fatalf("All = %v, want nil", err)
	}
}

func TestRequiredReportsOnlyPresenceForZeroValues(t *testing.T) {
	err := All(
		Required("ID", "", UUID()),
		Required("CreatedAt", time.Time{}),
		Required("IntervalSeconds", 0, GreaterThan(0)),
	)
	want := "created_at: is required; id: is required; interval_seconds: is required"
	if err == nil || err.Error() != want {
		t.Fatalf("All = %v, want %q", err, want)
	}
}

func TestOptionalSkipsRulesForZeroValues(t *testing.T) {
	empty := ""
	if err := All(
		Optional("Email", "", Email()),
		OptionalPtr[string]("Name", nil, Chars(1, 10)),
		OptionalPtr("Description", &empty, Chars(1, 10)),
		Optional("AttemptCount", 0, AtLeast(1)),
		Each("Roles", []string{""}, URL()),
	); err != nil {
		t.Fatalf("All = %v, want nil", err)
	}
}

func TestEveryFailingCheckIsReportedInSortedOrder(t *testing.T) {
	logout := "http://x"
	err := All(
		OptionalPtr("BackChannelLogoutURI", &logout, URL(), Must(func(string) bool { return false }, "must be https")),
		Required("MaxAge", -1, AtLeast(0)),
		Condition(false, "credentials must match"),
	)
	want := "back_channel_logout_uri: must be https; credentials must match; max_age: number must be greater than or equal to 0"
	if err == nil || err.Error() != want {
		t.Fatalf("All = %v, want %q", err, want)
	}
}

func TestNestedAndEachPrefixThePath(t *testing.T) {
	type rule struct{ Name string }
	err := All(
		Nested("Schema", EachWith("Rules", []rule{{Name: "ok"}, {}}, func(r rule) Issues {
			return Required("Name", r.Name)
		})),
		Each("RedirectURIs", []string{"https://ok.example", "nope"}, URL()),
	)
	want := "redirect_ur_is[1]: must be a valid URL; schema._rules[1]._name: is required"
	if err == nil || err.Error() != want {
		t.Fatalf("All = %v, want %q", err, want)
	}
}

func TestNonEmptyRejectsEmptySlices(t *testing.T) {
	if err := All(NonEmpty("Scopes", []string{})); err == nil || err.Error() != "scopes: is required" {
		t.Fatalf("All = %v", err)
	}
	if err := All(NonEmpty("Scopes", []string{""})); err != nil {
		t.Fatalf("All = %v, want nil", err)
	}
}

func TestOneOfNamesTheOptionsUnlessOverridden(t *testing.T) {
	err := All(Required("SubjectType", "Group", OneOf("Agent", "User")))
	if err == nil || err.Error() != "subject_type: string must be one of [Agent User]" {
		t.Fatalf("All = %v", err)
	}
	err = All(Required("ResponseType", "token", OneOf("code").WithMessage("response_type must be code")))
	if err == nil || err.Error() != "response_type: response_type must be code" {
		t.Fatalf("All = %v", err)
	}
}

func TestUUIDAcceptsOnlyTheCanonicalVersion4Form(t *testing.T) {
	generated, err := NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	for value, want := range map[string]bool{
		generated:                                       true,
		"3f2504e0-4f89-41d3-9a0c-0305e82c3301":          true,
		"3F2504E0-4F89-41D3-9A0C-0305E82C3301":          false,
		"f81d4fae-7dec-11d0-a765-00a0c91e6bf6":          false, // バージョン 1
		"3f2504e0-4f89-41d3-ca0c-0305e82c3301":          false, // RFC 9562 以外の variant
		"{3f2504e0-4f89-41d3-9a0c-0305e82c3301}":        false,
		"urn:uuid:3f2504e0-4f89-41d3-9a0c-0305e82c3301": false,
		"3f2504e04f8941d39a0c0305e82c3301":              false,
	} {
		if got := All(Required("ID", value, UUID())) == nil; got != want {
			t.Errorf("UUID accepts %q = %v, want %v", value, got, want)
		}
	}
}

func TestURLRequiresSchemeAndHost(t *testing.T) {
	for value, want := range map[string]bool{
		"https://client.example/callback": true,
		"custom://app":                    true,
		"/callback":                       false,
		"mailto:a@example.com":            false,
		"://missing-scheme":               false,
	} {
		if got := IsURL(value); got != want {
			t.Errorf("IsURL(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestEmailAcceptsOnlyABareAddrSpec(t *testing.T) {
	for value, want := range map[string]bool{
		"alice@example.com":           true,
		"alice+tag@sub.example.co.jp": true,
		"alice@localhost":             true,
		"Straße@Example.com":          true,
		"あいす@例え.jp":                   true,
		"Alice <alice@example.com>":   false,
		"<alice@example.com>":         false,
		"alice":                       false,
		"alice@example.com ":          false,
	} {
		if got := All(Required("Email", value, Email())) == nil; got != want {
			t.Errorf("Email accepts %q = %v, want %v", value, got, want)
		}
	}
}

func TestLengthChecksMakeTheErrorTyped(t *testing.T) {
	var lengthErr *LengthError
	if err := All(Required("Name", "abc", Chars(1, 2))); !errors.As(err, &lengthErr) {
		t.Fatalf("chars violation is not a *LengthError: %T %v", err, err)
	}
	if err := All(Required("Key", "ああ", KeyString(10, 3))); !errors.As(err, &lengthErr) || err.Error() != "key: must be at most 3 bytes" {
		t.Fatalf("byte violation = %T %v", err, err)
	}
}
