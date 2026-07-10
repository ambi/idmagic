package domain

import (
	"testing"
	"time"
)

// wi-172: shared/spec から移設した Application enum / event のカバレッジ (旧 wi-129 分)。

type enumValue interface{ Valid() bool }

func TestApplicationEnumValid(t *testing.T) {
	cases := []struct {
		name string
		v    enumValue
		want bool
	}{
		{"appkind federated", ApplicationFederated, true},
		{"appkind weblink", ApplicationWeblink, true},
		{"appkind service", ApplicationService, true},
		{"appkind bad", ApplicationKind("x"), false},

		{"appstatus active", ApplicationActive, true},
		{"appstatus disabled", ApplicationDisabled, true},
		{"appstatus bad", ApplicationStatus("x"), false},

		{"binding oidc", ProtocolBindingOIDC, true},
		{"binding saml", ProtocolBindingSAML, true},
		{"binding wsfed", ProtocolBindingWsFed, true},
		{"binding bad", ProtocolBindingType("x"), false},

		{"subject user", AssignmentSubjectUser, true},
		{"subject group", AssignmentSubjectGroup, true},
		{"subject bad", AssignmentSubjectType("x"), false},

		{"visibility visible", AssignmentVisible, true},
		{"visibility hidden", AssignmentHidden, true},
		{"visibility bad", AssignmentVisibility("x"), false},

		{"authn password", RequiredAuthnPassword, true},
		{"authn mfa", RequiredAuthnMfa, true},
		{"authn bad", RequiredAuthnStrength("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

func TestApplicationEventsTypeAndOccurredAt(t *testing.T) {
	at := time.Now().UTC()
	type ev interface {
		EventType() string
		OccurredAt() time.Time
	}
	cases := []struct {
		e    ev
		want string
	}{
		{&ApplicationCreated{At: at}, "ApplicationCreated"},
		{&ApplicationUpdated{At: at}, "ApplicationUpdated"},
		{&ApplicationIconUpdated{At: at}, "ApplicationIconUpdated"},
		{&ApplicationDeleted{At: at}, "ApplicationDeleted"},
		{&ProtocolBindingAttached{At: at}, "ProtocolBindingAttached"},
		{&ProtocolBindingDetached{At: at}, "ProtocolBindingDetached"},
		{&ApplicationAssigned{At: at}, "ApplicationAssigned"},
		{&ApplicationUnassigned{At: at}, "ApplicationUnassigned"},
		{&AppSignInPolicyUpdated{At: at}, "AppSignInPolicyUpdated"},
		{&TenantDefaultSignInPolicyUpdated{At: at}, "TenantDefaultSignInPolicyUpdated"},
		{&AppAccessDeniedByPolicy{At: at}, "AppAccessDeniedByPolicy"},
		{&AppStepUpRequired{At: at}, "AppStepUpRequired"},
		{&ApplicationCategoryCreated{At: at}, "ApplicationCategoryCreated"},
		{&ApplicationCategoryUpdated{At: at}, "ApplicationCategoryUpdated"},
		{&ApplicationCategoryDeleted{At: at}, "ApplicationCategoryDeleted"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.e.EventType(); got != c.want {
				t.Fatalf("EventType = %q, want %q", got, c.want)
			}
			if !c.e.OccurredAt().Equal(at) {
				t.Fatalf("OccurredAt = %v, want %v", c.e.OccurredAt(), at)
			}
		})
	}
}
