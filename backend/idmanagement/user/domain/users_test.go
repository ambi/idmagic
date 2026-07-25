package domain_test

import (
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func validUser() userdomain.User {
	now := time.Now().UTC()
	return userdomain.User{
		ID:                "user_alice",
		PreferredUsername: "alice",
		PasswordHash:      "$argon2id$v=19$m=19456,t=2,p=1$...",
		EmailVerified:     true,
		MfaEnrolled:       false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func TestUserValidateAcceptsMinimumValidShape(t *testing.T) {
	if err := validUser().Validate(); err != nil {
		t.Fatalf("expected valid user, got %v", err)
	}
}

func TestUserValidateRejectsEmptySub(t *testing.T) {
	u := validUser()
	u.ID = ""
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for empty sub")
	}
}

func TestUserValidateRejectsOversizedUsername(t *testing.T) {
	u := validUser()
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'x'
	}
	u.PreferredUsername = string(long)
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for >100-char preferred_username")
	}
}

func TestUserValidateRejectsMalformedEmail(t *testing.T) {
	u := validUser()
	bad := "not-an-email"
	u.Email = &bad
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for malformed email")
	}
}

// 通知の locale 解決の第 1 段は受信者 User の locale 属性 (ADR-142 決定 7)。属性は
// sum type なので、型が String でない、値が無い、そもそも属性が無いケースは
// 「未設定」として空文字列を返す。
func TestUserLocaleAttribute(t *testing.T) {
	stringValue := func(value string) userdomain.AttributeValue {
		return userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: &value}
	}

	cases := []struct {
		name       string
		attributes map[string]userdomain.AttributeValue
		want       string
	}{
		{"no attributes at all", nil, ""},
		{"locale is unset", map[string]userdomain.AttributeValue{"nickname": stringValue("al")}, ""},
		{"locale is a language tag", map[string]userdomain.AttributeValue{"locale": stringValue("ja")}, "ja"},
		{"locale is a bcp47 tag", map[string]userdomain.AttributeValue{"locale": stringValue("ja-JP")}, "ja-JP"},
		{"locale has the wrong type", map[string]userdomain.AttributeValue{
			"locale": {Type: idmdomain.AttributeTypeNumber, Number: func() *float64 { v := 1.0; return &v }()},
		}, ""},
		{"locale has no value", map[string]userdomain.AttributeValue{
			"locale": {Type: idmdomain.AttributeTypeString},
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := validUser()
			user.Attributes = tc.attributes
			if got := user.LocaleAttribute(); got != tc.want {
				t.Fatalf("LocaleAttribute() = %q, want %q", got, tc.want)
			}
		})
	}
}

// 通知の宛名に使う表示名。name → given_name + family_name → preferred_username の順に
// 「人が読める名前」へ落とす。空文字列は返さない (宛名が消えたメールを送らない)。
func TestUserNotificationDisplayName(t *testing.T) {
	user := validUser()
	if got := user.DisplayName(); got != "alice" {
		t.Errorf("DisplayName() with no name = %q, want the preferred username", got)
	}

	given, family := "Hanako", "Yamada"
	user.GivenName, user.FamilyName = &given, &family
	if got := user.DisplayName(); got != "Hanako Yamada" {
		t.Errorf("DisplayName() with given/family = %q", got)
	}

	name := "山田 花子"
	user.Name = &name
	if got := user.DisplayName(); got != "山田 花子" {
		t.Errorf("DisplayName() with name = %q, want the full name", got)
	}

	blank := "   "
	user.Name = &blank
	if got := user.DisplayName(); got != "Hanako Yamada" {
		t.Errorf("DisplayName() with a blank name = %q, want the given/family fallback", got)
	}
}
