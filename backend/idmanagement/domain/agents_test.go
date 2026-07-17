package domain_test

import (
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// wi-129: 純粋ドメイン (enum Valid / Validate / コンストラクタ / 状態機械) のカバレッジ補強。
// internal/shared/spec/wi129_coverage_test.go から移設 (wi-178)。

func TestEnumValid(t *testing.T) {
	cases := []struct {
		name string
		v    interface{ Valid() bool }
		want bool
	}{
		{"agent active", idmdomain.AgentStatusActive, true},
		{"agent disabled", idmdomain.AgentStatusDisabled, true},
		{"agent killed", idmdomain.AgentStatusKilled, true},
		{"agent bad", idmdomain.AgentStatus("x"), false},

		{"agentkind autonomous", idmdomain.AgentKindAutonomous, true},
		{"agentkind supervised", idmdomain.AgentKindSupervised, true},
		{"agentkind bad", idmdomain.AgentKind("x"), false},

		{"userstatus active", idmdomain.UserStatusActive, true},
		{"userstatus disabled", idmdomain.UserStatusDisabled, true},
		{"userstatus pending", idmdomain.UserStatusPendingDeletion, true},
		{"userstatus deleted", idmdomain.UserStatusDeleted, true},
		{"userstatus locked", idmdomain.UserStatusLocked, true},
		{"userstatus staged", idmdomain.UserStatusStaged, true},
		{"userstatus suspended", idmdomain.UserStatusSuspended, true},
		{"userstatus bad", idmdomain.UserStatus("x"), false},

		{"reqaction update password", idmdomain.RequiredActionUpdatePassword, true},
		{"reqaction verify email", idmdomain.RequiredActionVerifyEmail, true},
		{"reqaction configure totp", idmdomain.RequiredActionConfigureTOTP, true},
		{"reqaction update profile", idmdomain.RequiredActionUpdateProfile, true},
		{"reqaction terms", idmdomain.RequiredActionTermsAndConditions, true},
		{"reqaction bad", idmdomain.RequiredAction("x"), false},

		{"attrtype string", idmdomain.AttributeTypeString, true},
		{"attrtype number", idmdomain.AttributeTypeNumber, true},
		{"attrtype boolean", idmdomain.AttributeTypeBoolean, true},
		{"attrtype date", idmdomain.AttributeTypeDate, true},
		{"attrtype string_array", idmdomain.AttributeTypeStringArray, true},
		{"attrtype bad", idmdomain.AttributeType("x"), false},

		{"attrvis private", idmdomain.AttrVisibilityPrivate, true},
		{"attrvis self", idmdomain.AttrVisibilitySelfReadable, true},
		{"attrvis admin", idmdomain.AttrVisibilityAdminReadable, true},
		{"attrvis claim", idmdomain.AttrVisibilityClaimExposed, true},
		{"attrvis bad", idmdomain.AttrVisibility("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

func TestValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validAgent := idmdomain.Agent{
		ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "user_1", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	badAgent := validAgent
	badAgent.Kind = idmdomain.AgentKind("x")

	validBinding := idmdomain.AgentCredentialBinding{AgentID: "agent_1", ClientID: "demo", CreatedAt: now}
	badBinding := idmdomain.AgentCredentialBinding{CreatedAt: now}

	validGroup := idmdomain.Group{ID: "group_1", TenantID: tenancydomain.DefaultTenantID, Name: "eng", CreatedAt: now, UpdatedAt: now}
	badGroup := validGroup
	badGroup.Name = ""

	validMember := idmdomain.GroupMember{GroupID: "group_1", UserID: "user_1", CreatedAt: now}
	badMember := idmdomain.GroupMember{UserID: "user_1", CreatedAt: now}

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"agent ok", validAgent, false},
		{"agent bad", badAgent, true},
		{"binding ok", validBinding, false},
		{"binding bad", badBinding, true},
		{"group ok", validGroup, false},
		{"group bad", badGroup, true},
		{"member ok", validMember, false},
		{"member bad", badMember, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}

func TestNewIDsAreUUIDs(t *testing.T) {
	agentID, err := idmdomain.NewAgentID()
	if err != nil {
		t.Fatalf("NewAgentID: %v", err)
	}
	if len(agentID) != 36 {
		t.Fatalf("NewAgentID = %q, want UUID", agentID)
	}
	groupID, err := idmdomain.NewGroupID()
	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}
	if len(groupID) != 36 {
		t.Fatalf("NewGroupID = %q, want UUID", groupID)
	}
}

func TestAgentIsActive(t *testing.T) {
	now := time.Now().UTC()
	active := idmdomain.Agent{Status: idmdomain.AgentStatusActive}
	if !active.IsActive() {
		t.Fatal("active agent must be active")
	}
	disabledStatus := idmdomain.Agent{Status: idmdomain.AgentStatusDisabled}
	if disabledStatus.IsActive() {
		t.Fatal("disabled status must not be active")
	}
	withDisabledAt := idmdomain.Agent{Status: idmdomain.AgentStatusActive, DisabledAt: &now}
	if withDisabledAt.IsActive() {
		t.Fatal("disabled_at set must not be active")
	}
	withKilledAt := idmdomain.Agent{Status: idmdomain.AgentStatusActive, KilledAt: &now}
	if withKilledAt.IsActive() {
		t.Fatal("killed_at set must not be active")
	}
}
