package domain

import (
	"slices"
	"testing"
	"time"
)

// REQ-AUTHENTICATION-033: 必須の種別はカタログ側で固定し、設定からは外せない。
func TestMandatoryCategoriesCannotBeDisabled(t *testing.T) {
	t.Parallel()

	for _, category := range []Category{
		CategoryCredentialChange, CategoryMfaChange, CategoryContactChange, CategoryImpersonation,
	} {
		if !category.Mandatory() {
			t.Errorf("%s.Mandatory() = false, want true", category)
		}
		if _, err := NewPreferences("alice", []Category{category}, time.Now()); err == nil {
			t.Errorf("NewPreferences with the mandatory %s returned no error", category)
		}
	}
	for _, category := range []Category{CategoryNewDeviceSignIn, CategorySessionRevoked} {
		if category.Mandatory() {
			t.Errorf("%s.Mandatory() = true, want false", category)
		}
	}
}

// REQ-AUTHENTICATION-034: 停止した種別だけが止まり、他の種別は届き続ける。
func TestPreferencesAllowEveryCategoryExceptTheDisabledOnes(t *testing.T) {
	t.Parallel()

	empty := Preferences{}
	for _, category := range Categories() {
		if !empty.Allows(category) {
			t.Errorf("the zero Preferences must allow %s", category)
		}
	}

	prefs, err := NewPreferences("alice", []Category{CategoryNewDeviceSignIn}, time.Now())
	if err != nil {
		t.Fatalf("NewPreferences: %v", err)
	}
	if prefs.Allows(CategoryNewDeviceSignIn) {
		t.Error("a disabled category must not be allowed")
	}
	if !prefs.Allows(CategoryCredentialChange) {
		t.Error("disabling one category must not affect another")
	}
}

func TestNewPreferencesRejectsUnknownCategoriesAndDeduplicates(t *testing.T) {
	t.Parallel()

	if _, err := NewPreferences("alice", []Category{"does_not_exist"}, time.Now()); err == nil {
		t.Error("NewPreferences accepted an unknown category")
	}
	if _, err := NewPreferences("", []Category{}, time.Now()); err == nil {
		t.Error("NewPreferences accepted an empty user id")
	}
	prefs, err := NewPreferences(
		"alice", []Category{CategorySessionRevoked, CategoryNewDeviceSignIn, CategorySessionRevoked}, time.Now(),
	)
	if err != nil {
		t.Fatalf("NewPreferences: %v", err)
	}
	want := []Category{CategoryNewDeviceSignIn, CategorySessionRevoked}
	if !slices.Equal(prefs.Disabled, want) {
		t.Errorf("Disabled = %v, want the deduplicated catalog order %v", prefs.Disabled, want)
	}
}

// REQ-AUTHENTICATION-030 / 031 / 032: カタログに載る全イベントが、種別・宛先の項目・
// ja / en の説明をそろって持つ。
func TestTriggerCatalogIsComplete(t *testing.T) {
	t.Parallel()

	eventTypes := []string{
		"UserAuthenticated", "PasswordChanged",
		"MfaFactorEnrolled", "MfaFactorRemoved",
		"WebAuthnCredentialRegistered", "WebAuthnCredentialRemoved",
		"RecoveryCodesGenerated", "RecoveryCodesRevoked",
		"AuthenticatorResetCompleted", "TrustedDeviceRegistered",
		"EmailChangeRequested", "EmailChanged",
		"SessionEnded", "SessionImpersonationStarted",
	}
	for _, eventType := range eventTypes {
		trigger, ok := TriggerFor(eventType)
		if !ok {
			t.Errorf("TriggerFor(%q) is not in the catalog", eventType)
			continue
		}
		if !trigger.Category.Valid() {
			t.Errorf("%s carries the unknown category %q", eventType, trigger.Category)
		}
		if trigger.RecipientField == "" {
			t.Errorf("%s declares no recipient field", eventType)
		}
		for _, locale := range []string{"ja", "en"} {
			if trigger.Descriptions[locale] == "" {
				t.Errorf("%s has no %s description", eventType, locale)
			}
		}
	}
	if _, ok := TriggerFor("AccountSecurityNotificationSent"); ok {
		t.Error("the dispatcher's own event must not be in the catalog, or notifications would chain")
	}
	if _, ok := TriggerFor("AuthenticationFailed"); ok {
		t.Error("failed authentication must not notify: it is the flood an attacker controls")
	}
}

// なりすましの通知は、操作した管理者ではなく、なりすまされた本人へ送る。
func TestImpersonationNotifiesTheTargetRatherThanTheActor(t *testing.T) {
	t.Parallel()

	trigger, ok := TriggerFor("SessionImpersonationStarted")
	if !ok {
		t.Fatal("SessionImpersonationStarted is not in the catalog")
	}
	if trigger.RecipientField != "targetUserId" {
		t.Errorf("RecipientField = %q, want targetUserId", trigger.RecipientField)
	}
}

// セッションの失効は、本人または管理者が明示的に失効させたときだけ通知する。
// 期限切れやログアウトまで通知すると、通知そのものが無視されるようになる。
func TestSessionEndNotifiesOnlyExplicitRevocations(t *testing.T) {
	t.Parallel()

	trigger, ok := TriggerFor("SessionEnded")
	if !ok {
		t.Fatal("SessionEnded is not in the catalog")
	}
	for reason, want := range map[string]bool{
		"self_revoke": true, "admin_revoke": true,
		"logout": false, "idle": false, "absolute": false,
		"password_change": false, "mfa_change": false, "other": false,
	} {
		if got := trigger.Applies(map[string]any{"reason": reason}); got != want {
			t.Errorf("SessionEnded with reason %q: Applies = %v, want %v", reason, got, want)
		}
	}
}
