package spec

import (
	"testing"
	"time"
)

// wi-129: 純粋ドメイン (enum Valid / Validate / コンストラクタ / 状態機械) のカバレッジ補強。

// ---------------------------------------------------------------
// enum Valid()
// ---------------------------------------------------------------

// enumValue は Valid() を持つ全 typed string enum を共通に扱うためのインタフェース。
type enumValue interface{ Valid() bool }

func TestEnumValid(t *testing.T) {
	cases := []struct {
		name string
		v    enumValue
		want bool
	}{
		{"client public", ClientPublic, true},
		{"client confidential", ClientConfidential, true},
		{"client bad", ClientType("x"), false},

		{"grant auth code", GrantAuthorizationCode, true},
		{"grant refresh", GrantRefreshToken, true},
		{"grant client credentials", GrantClientCredentials, true},
		{"grant device", GrantDeviceCode, true},
		{"grant token exchange", GrantTokenExchange, true},
		{"grant bad", GrantType("x"), false},

		{"response code", ResponseTypeCode, true},
		{"response bad", ResponseType("token"), false},

		{"authmethod basic", AuthMethodClientSecretBasic, true},
		{"authmethod post", AuthMethodClientSecretPost, true},
		{"authmethod private key jwt", AuthMethodPrivateKeyJwt, true},
		{"authmethod tls", AuthMethodTlsClientAuth, true},
		{"authmethod none", AuthMethodNone, true},
		{"authmethod bad", TokenEndpointAuthMethod("x"), false},

		{"sig ps256", SigAlgPS256, true},
		{"sig es256", SigAlgES256, true},
		{"sig bad", SignatureAlgorithm("RS256"), false},

		{"keyprovider local", KeyProviderLocal, true},
		{"keyprovider postgres", KeyProviderPostgres, true},
		{"keyprovider vault", KeyProviderVaultTransit, true},
		{"keyprovider bad", KeyProvider("x"), false},

		{"keyusage signing", KeyUsageSigning, true},
		{"keyusage bad", KeyUsage("enc"), false},

		{"fapi none", FapiNone, true},
		{"fapi v2", FapiSecurityProfileV2, true},
		{"fapi bad", FapiProfile("x"), false},

		{"cc method s256", CodeChallengeMethodS256, true},
		{"cc method bad", CodeChallengeMethod("plain"), false},

		{"mfa totp", MfaFactorTOTP, true},
		{"mfa webauthn", MfaFactorWebAuthn, true},
		{"mfa hwk", MfaFactorHWK, true},
		{"mfa swk", MfaFactorSWK, true},
		{"mfa bad", MfaFactorType("sms"), false},

		{"authflow received", AuthFlowReceived, true},
		{"authflow exchanged", AuthFlowExchanged, true},
		{"authflow bad", AuthorizationCodeFlowState("x"), false},

		{"authcode issued", AuthCodeRecordIssued, true},
		{"authcode redeemed", AuthCodeRecordRedeemed, true},
		{"authcode expired", AuthCodeRecordExpired, true},
		{"authcode bad", AuthorizationCodeRecordState("x"), false},

		{"consent granted", ConsentGranted, true},
		{"consent revoked", ConsentRevoked, true},
		{"consent expired", ConsentExpired, true},
		{"consent bad", ConsentState("x"), false},

		{"detail field set", DetailFieldSet, true},
		{"detail field at_most", DetailFieldAtMost, true},
		{"detail field enum", DetailFieldEnum, true},
		{"detail field exact", DetailFieldExact, true},
		{"detail field bad", AuthorizationDetailFieldSemantics("x"), false},

		{"detail type enabled", DetailTypeEnabled, true},
		{"detail type disabled", DetailTypeDisabled, true},
		{"detail type bad", AuthorizationDetailTypeState("x"), false},

		{"session logout", SessionEndLogout, true},
		{"session idle", SessionEndIdle, true},
		{"session absolute", SessionEndAbsolute, true},
		{"session self revoke", SessionEndSelfRevoke, true},
		{"session admin revoke", SessionEndAdminRevoke, true},
		{"session password change", SessionEndPasswordChange, true},
		{"session mfa change", SessionEndMfaChange, true},
		{"session other", SessionEndOther, true},
		{"session bad", SessionEndReason("x"), false},

		{"device issued", DeviceFlowIssued, true},
		{"device entered", DeviceFlowUserCodeEntered, true},
		{"device approved", DeviceFlowApproved, true},
		{"device denied", DeviceFlowDenied, true},
		{"device exchanged", DeviceFlowExchanged, true},
		{"device expired", DeviceFlowExpired, true},
		{"device bad", DeviceCodeFlowState("x"), false},

		{"response mode query", ResponseModeQuery, true},
		{"response mode form_post", ResponseModeFormPost, true},
		{"response mode bad", ResponseMode("fragment"), false},

		{"tenant active", TenantStatusActive, true},
		{"tenant disabled", TenantStatusDisabled, true},
		{"tenant bad", TenantStatus("x"), false},

		{"agent active", AgentStatusActive, true},
		{"agent disabled", AgentStatusDisabled, true},
		{"agent killed", AgentStatusKilled, true},
		{"agent bad", AgentStatus("x"), false},

		{"agentkind autonomous", AgentKindAutonomous, true},
		{"agentkind supervised", AgentKindSupervised, true},
		{"agentkind bad", AgentKind("x"), false},

		{"userstatus active", UserStatusActive, true},
		{"userstatus disabled", UserStatusDisabled, true},
		{"userstatus pending", UserStatusPendingDeletion, true},
		{"userstatus deleted", UserStatusDeleted, true},
		{"userstatus locked", UserStatusLocked, true},
		{"userstatus staged", UserStatusStaged, true},
		{"userstatus suspended", UserStatusSuspended, true},
		{"userstatus bad", UserStatus("x"), false},

		{"reqaction update password", RequiredActionUpdatePassword, true},
		{"reqaction verify email", RequiredActionVerifyEmail, true},
		{"reqaction configure totp", RequiredActionConfigureTOTP, true},
		{"reqaction update profile", RequiredActionUpdateProfile, true},
		{"reqaction terms", RequiredActionTermsAndConditions, true},
		{"reqaction bad", RequiredAction("x"), false},

		{"attrtype string", AttributeTypeString, true},
		{"attrtype number", AttributeTypeNumber, true},
		{"attrtype boolean", AttributeTypeBoolean, true},
		{"attrtype date", AttributeTypeDate, true},
		{"attrtype string_array", AttributeTypeStringArray, true},
		{"attrtype bad", AttributeType("x"), false},

		{"attrvis private", AttrVisibilityPrivate, true},
		{"attrvis self", AttrVisibilitySelfReadable, true},
		{"attrvis admin", AttrVisibilityAdminReadable, true},
		{"attrvis claim", AttrVisibilityClaimExposed, true},
		{"attrvis bad", AttrVisibility("x"), false},

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

// ---------------------------------------------------------------
// Validate() happy / failure
// ---------------------------------------------------------------

func mustUUID(t *testing.T) string {
	t.Helper()
	id, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	return id
}

func TestValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validAgent := Agent{
		ID: "agent_1", TenantID: "default", Name: "bot", Kind: AgentKindAutonomous,
		OwnerUserID: "user_1", Status: AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	badAgent := validAgent
	badAgent.Kind = AgentKind("x")

	validBinding := AgentCredentialBinding{AgentID: "agent_1", ClientID: "demo", CreatedAt: now}
	badBinding := AgentCredentialBinding{CreatedAt: now}

	validMfa := MfaFactor{UserID: "user_1", Type: MfaFactorWebAuthn, CreatedAt: now}
	// TOTP は secret 必須なので secret 無しは失敗する。
	badMfa := MfaFactor{UserID: "user_1", Type: MfaFactorTOTP, CreatedAt: now}

	validSession := LoginSession{ID: mustUUID(t), UserID: "user_1", AMR: []string{"pwd"}, ACR: "1", ExpiresAt: now}
	badSession := validSession
	badSession.AMR = nil

	validLoginReq := LoginRequest{RequestID: mustUUID(t), Username: "alice", Password: "pw"}
	badLoginReq := LoginRequest{RequestID: "not-a-uuid", Username: "alice", Password: "pw"}

	validGroup := Group{ID: "group_1", TenantID: "default", Name: "eng", CreatedAt: now, UpdatedAt: now}
	badGroup := validGroup
	badGroup.Name = ""

	validMember := GroupMember{GroupID: "group_1", UserID: "user_1", CreatedAt: now}
	badMember := GroupMember{UserID: "user_1", CreatedAt: now}

	validTenant := Tenant{ID: "acme", DisplayName: "Acme", Status: TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	badTenant := validTenant
	badTenant.ID = "admin" // admin は予約語で拒否される。

	validClient := OAuth2Client{
		ClientID: "demo", ClientType: ClientConfidential,
		RedirectURIs: []string{"https://app.example.com/cb"},
		GrantTypes:   []GrantType{GrantAuthorizationCode}, ResponseTypes: []ResponseType{ResponseTypeCode},
		TokenEndpointAuthMethod: AuthMethodClientSecretBasic, IDTokenSignedResponseAlg: SigAlgPS256,
		FapiProfile: FapiNone, CreatedAt: now, UpdatedAt: now,
	}
	// authorization_code グラントだが redirect_uris が無いので失敗する。
	badClient := validClient
	badClient.RedirectURIs = nil

	validConsent := Consent{UserID: "user_1", ClientID: "demo", Scopes: []string{"openid"}, State: ConsentGranted, GrantedAt: now, ExpiresAt: now}
	badConsent := validConsent
	badConsent.Scopes = nil

	validDetailType := AuthorizationDetailType{
		TenantID: "default", Type: "payment", DisplayTemplate: "{{.amount}}", State: DetailTypeEnabled,
		Schema:    AuthorizationDetailsSchema{Rules: []AuthorizationDetailFieldRule{{Name: "amount", Semantics: DetailFieldExact}}},
		CreatedAt: now, UpdatedAt: now,
	}
	badDetailType := validDetailType
	badDetailType.Schema = AuthorizationDetailsSchema{Rules: nil}

	validAuthReq := AuthorizationRequest{
		ID: mustUUID(t), State: AuthFlowReceived, ClientID: "demo",
		RedirectURI: "https://app.example.com/cb", ResponseType: ResponseTypeCode,
		CodeChallenge: "abc", CodeChallengeMethod: CodeChallengeMethodS256, CreatedAt: now, ExpiresAt: now,
	}
	badAuthReq := validAuthReq
	badAuthReq.RedirectURI = "not-a-url"

	validCodeRec := AuthorizationCodeRecord{
		Code: "c", AuthorizationRequestID: mustUUID(t), ClientID: "demo", UserID: "user_1",
		RedirectURI: "https://app.example.com/cb", CodeChallenge: "abc", CodeChallengeMethod: CodeChallengeMethodS256,
		State: AuthCodeRecordIssued, IssuedAt: now, ExpiresAt: now,
	}
	badCodeRec := validCodeRec
	badCodeRec.State = AuthorizationCodeRecordState("x")

	validRefresh := RefreshTokenRecord{
		ID: mustUUID(t), Hash: "h", FamilyID: mustUUID(t), ClientID: "demo", UserID: "user_1",
		IssuedAt: now, ExpiresAt: now, AbsoluteExpiresAt: now,
	}
	badRefresh := validRefresh
	badRefresh.FamilyID = "not-a-uuid"

	validPAR := PARRecord{RequestURI: "urn:x", ClientID: "demo", IssuedAt: now, ExpiresAt: now}
	badPAR := PARRecord{ClientID: "demo", IssuedAt: now, ExpiresAt: now}

	validDevice := DeviceAuthorization{
		DeviceCodeHash: "h", UserCode: "ABCD", ClientID: "demo", State: DeviceFlowIssued,
		IntervalSeconds: 5, IssuedAt: now, ExpiresAt: now,
	}
	badDevice := validDevice
	badDevice.IntervalSeconds = 0 // GT(0) 違反。

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"agent ok", validAgent, false},
		{"agent bad", badAgent, true},
		{"binding ok", validBinding, false},
		{"binding bad", badBinding, true},
		{"mfa ok", validMfa, false},
		{"mfa bad", badMfa, true},
		{"session ok", validSession, false},
		{"session bad", badSession, true},
		{"login req ok", validLoginReq, false},
		{"login req bad", badLoginReq, true},
		{"group ok", validGroup, false},
		{"group bad", badGroup, true},
		{"member ok", validMember, false},
		{"member bad", badMember, true},
		{"tenant ok", validTenant, false},
		{"tenant bad", badTenant, true},
		{"client ok", validClient, false},
		{"client bad", badClient, true},
		{"consent ok", validConsent, false},
		{"consent bad", badConsent, true},
		{"detail type ok", validDetailType, false},
		{"detail type bad", badDetailType, true},
		{"auth req ok", validAuthReq, false},
		{"auth req bad", badAuthReq, true},
		{"code record ok", validCodeRec, false},
		{"code record bad", badCodeRec, true},
		{"refresh ok", validRefresh, false},
		{"refresh bad", badRefresh, true},
		{"par ok", validPAR, false},
		{"par bad", badPAR, true},
		{"device ok", validDevice, false},
		{"device bad", badDevice, true},
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

// ---------------------------------------------------------------
// コンストラクタ / ID 生成
// ---------------------------------------------------------------

func TestNewUUIDv4Format(t *testing.T) {
	id, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("uuid length = %d, want 36 (%q)", len(id), id)
	}
	// version 4 / RFC 4122 variant の位置を確認する。
	if id[14] != '4' {
		t.Fatalf("version nibble = %q, want 4 (%q)", id[14], id)
	}
	switch id[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Fatalf("variant nibble = %q, want 8/9/a/b (%q)", id[19], id)
	}
	// 一意性: 2 回連続で異なる。
	id2, _ := NewUUIDv4()
	if id == id2 {
		t.Fatal("two UUIDs must differ")
	}
}

func TestNewIDPrefixes(t *testing.T) {
	agentID, err := NewAgentID()
	if err != nil {
		t.Fatalf("NewAgentID: %v", err)
	}
	if len(agentID) <= len("agent_") || agentID[:6] != "agent_" {
		t.Fatalf("NewAgentID = %q, want agent_ prefix", agentID)
	}
	groupID, err := NewGroupID()
	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}
	if len(groupID) <= len("group_") || groupID[:6] != "group_" {
		t.Fatalf("NewGroupID = %q, want group_ prefix", groupID)
	}
}

// ---------------------------------------------------------------
// Agent.IsActive
// ---------------------------------------------------------------

func TestAgentIsActive(t *testing.T) {
	now := time.Now().UTC()
	active := Agent{Status: AgentStatusActive}
	if !active.IsActive() {
		t.Fatal("active agent must be active")
	}
	disabledStatus := Agent{Status: AgentStatusDisabled}
	if disabledStatus.IsActive() {
		t.Fatal("disabled status must not be active")
	}
	withDisabledAt := Agent{Status: AgentStatusActive, DisabledAt: &now}
	if withDisabledAt.IsActive() {
		t.Fatal("disabled_at set must not be active")
	}
	withKilledAt := Agent{Status: AgentStatusActive, KilledAt: &now}
	if withKilledAt.IsActive() {
		t.Fatal("killed_at set must not be active")
	}
}

// ---------------------------------------------------------------
// DeviceCodeFlow 状態機械
// ---------------------------------------------------------------

func TestTransitionDeviceCodeFlow(t *testing.T) {
	steps := []struct {
		from  DeviceCodeFlowState
		event DeviceCodeFlowEvent
		to    DeviceCodeFlowState
	}{
		{DeviceFlowIssued, DeviceEventEnterUserCode, DeviceFlowUserCodeEntered},
		{DeviceFlowUserCodeEntered, DeviceEventApprove, DeviceFlowApproved},
		{DeviceFlowApproved, DeviceEventExchange, DeviceFlowExchanged},
		{DeviceFlowUserCodeEntered, DeviceEventDeny, DeviceFlowDenied},
		{DeviceFlowIssued, DeviceEventExpire, DeviceFlowExpired},
	}
	for _, s := range steps {
		got, err := TransitionDeviceCodeFlow(s.from, s.event)
		if err != nil {
			t.Fatalf("transition %q on %q: %v", s.from, s.event, err)
		}
		if got != s.to {
			t.Fatalf("transition %q on %q: got %q want %q", s.from, s.event, got, s.to)
		}
	}
	if _, err := TransitionDeviceCodeFlow(DeviceFlowIssued, DeviceEventExchange); err == nil {
		t.Fatal("expected error: cannot exchange from issued")
	}
}

func TestIsDeviceCodeFlowTerminal(t *testing.T) {
	terminal := []DeviceCodeFlowState{DeviceFlowDenied, DeviceFlowExchanged, DeviceFlowExpired}
	for _, s := range terminal {
		if !IsDeviceCodeFlowTerminal(s) {
			t.Fatalf("%q must be terminal", s)
		}
	}
	nonTerminal := []DeviceCodeFlowState{DeviceFlowIssued, DeviceFlowUserCodeEntered, DeviceFlowApproved}
	for _, s := range nonTerminal {
		if IsDeviceCodeFlowTerminal(s) {
			t.Fatalf("%q must not be terminal", s)
		}
	}
}

func TestDefaultDeviceCodePolling(t *testing.T) {
	p := DefaultDeviceCodePolling()
	if p.DefaultIntervalSeconds != 5 || p.SlowDownIncrementSeconds != 5 {
		t.Fatalf("unexpected default polling: %+v", p)
	}
}

func TestIsAuthorizationCodeRecordTerminal(t *testing.T) {
	if !IsAuthorizationCodeRecordTerminal(AuthCodeRecordRedeemed) ||
		!IsAuthorizationCodeRecordTerminal(AuthCodeRecordExpired) {
		t.Fatal("redeemed/expired must be terminal")
	}
	if IsAuthorizationCodeRecordTerminal(AuthCodeRecordIssued) {
		t.Fatal("issued must not be terminal")
	}
}

// ---------------------------------------------------------------
// EffectiveRoles (ADR-038)
// ---------------------------------------------------------------

func TestEffectiveRoles(t *testing.T) {
	g1 := &Group{Roles: []string{"editor", "viewer"}}
	g2 := &Group{Roles: []string{"viewer", "admin", ""}}
	got := EffectiveRoles([]string{"viewer", ""}, []*Group{g1, g2, nil})
	want := []string{"admin", "editor", "viewer"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (sorted, deduped, no empties)", got, want)
		}
	}
	// グループ無しなら user.roles に一致する。
	solo := EffectiveRoles([]string{"a"}, nil)
	if len(solo) != 1 || solo[0] != "a" {
		t.Fatalf("solo = %v, want [a]", solo)
	}
}

// ---------------------------------------------------------------
// Grant matrix helpers
// ---------------------------------------------------------------

func TestGrantMatrix(t *testing.T) {
	if _, ok := GetGrantSpec(GrantAuthorizationCode); !ok {
		t.Fatal("authorization_code must be in grant spec")
	}
	if _, ok := GetGrantSpec(GrantTokenExchange); ok {
		t.Fatal("token-exchange is not in the grant matrix")
	}
	if !GrantAllowsClientType(GrantClientCredentials, ClientConfidential) {
		t.Fatal("client_credentials must allow confidential clients")
	}
	if GrantAllowsClientType(GrantClientCredentials, ClientPublic) {
		t.Fatal("client_credentials must not allow public clients")
	}
	if !GrantRequiresPKCE(GrantAuthorizationCode) {
		t.Fatal("authorization_code must require PKCE")
	}
	if GrantRequiresPKCE(GrantRefreshToken) {
		t.Fatal("refresh_token must not require PKCE")
	}
	if !GrantIssues(GrantAuthorizationCode, "id_token") {
		t.Fatal("authorization_code must issue id_token")
	}
	if GrantIssues(GrantClientCredentials, "id_token") {
		t.Fatal("client_credentials must not issue id_token")
	}
	if GrantIssues(GrantType("bogus"), "access_token") {
		t.Fatal("unknown grant issues nothing")
	}
}

// ---------------------------------------------------------------
// Application イベント (EventType / OccurredAt)
// ---------------------------------------------------------------

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
