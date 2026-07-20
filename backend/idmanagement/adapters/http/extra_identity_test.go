package http_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/adapters/persistence/memory"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/adapters/persistence/memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/labstack/echo/v5"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type identityTestHandler struct {
	echo     *echo.Echo
	users    *usermemory.UserRepository
	tokens   *authnmemory.EmailChangeTokenStore
	groups   *groupmemory.GroupRepository
	clients  *oauth2memory.OAuth2ClientRepository
	consents *oauth2memory.ConsentRepository
}

func newIdentityTestHandler(t *testing.T) identityTestHandler {
	t.Helper()
	repo := usermemory.NewUserRepository()
	tokenStore := authnmemory.NewEmailChangeTokenStore()
	groupRepo := groupmemory.NewGroupRepository()
	clientRepo := oauth2memory.NewClientRepository()
	consentRepo := oauth2memory.NewConsentRepository()

	history := passwordmemory.NewPasswordHistoryRepository()
	hasher := crypto.NewArgon2idPasswordHasher()
	now := time.Now().UTC()
	for _, user := range []*userdomain.User{
		{
			ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
			Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
			TenantID: tenancydomain.DefaultTenantID,
		},
		{
			ID: "regular", PreferredUsername: "regular", PasswordHash: "unused",
			CreatedAt: now, UpdatedAt: now,
			TenantID: tenancydomain.DefaultTenantID,
		},
	} {
		repo.Seed(user)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{Issuer: "http://idp.test"}, UserRepo: repo, PasswordHasher: hasher,
		PasswordHistoryRepo: history, AuthnResolver: authusecases.DemoHeaderResolver{},
		AgentRepo: agentmemory.NewAgentRepository(),
		GroupRepo: groupRepo,
		OAuth2: oauth2.Module{
			ClientRepo:  clientRepo,
			ConsentRepo: consentRepo,
		},
		EmailChangeTokenStore: tokenStore,
		EmailSender:           mockEmailSender{},
	})
	return identityTestHandler{
		echo: e, users: repo, tokens: tokenStore, groups: groupRepo, clients: clientRepo, consents: consentRepo,
	}
}

func TestAdminAgentLifecycle(t *testing.T) {
	h := newIdentityTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)
	clientName := "Agent Client"
	_ = h.clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-1", ClientName: &clientName,
		ClientType: spec.ClientConfidential, GrantTypes: []spec.GrantType{spec.GrantClientCredentials},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodClientSecretBasic,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
	})

	// --- 1. Agent 1 (Create, Update, Bind, Unbind, Delete) ---
	desc := "Agent 1 Description"
	kind := "daemon"
	create1 := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": "Agent 1", "description": &desc, "kind": &kind, "roles": []string{"support"},
	})
	if create1.Code != http.StatusCreated {
		t.Fatalf("register agent 1 status=%d body=%s", create1.Code, create1.Body.String())
	}
	var created1 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(create1.Body.Bytes(), &created1)
	agentID1 := created1.ID

	// Get Agent
	getAgent := adminJSONRequest(t, e, http.MethodGet, "/api/admin/agents/"+agentID1, csrf, cookie, nil)
	if getAgent.Code != http.StatusOK {
		t.Fatalf("get agent status=%d body=%s", getAgent.Code, getAgent.Body.String())
	}

	// List Agents
	listAgents := adminJSONRequest(t, e, http.MethodGet, "/api/admin/agents", csrf, cookie, nil)
	if listAgents.Code != http.StatusOK {
		t.Fatalf("list agents status=%d body=%s", listAgents.Code, listAgents.Body.String())
	}

	// Update Agent
	newName := "Agent A Updated"
	update := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/agents/"+agentID1, csrf, cookie, map[string]any{
		"name": &newName,
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update agent status=%d body=%s", update.Code, update.Body.String())
	}

	// Disable Agent
	disable := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents/"+agentID1+"/disable", csrf, cookie, nil)
	if disable.Code != http.StatusNoContent {
		t.Fatalf("disable agent status=%d body=%s", disable.Code, disable.Body.String())
	}

	// Enable Agent
	enable := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents/"+agentID1+"/enable", csrf, cookie, nil)
	if enable.Code != http.StatusNoContent {
		t.Fatalf("enable agent status=%d body=%s", enable.Code, enable.Body.String())
	}

	// Bind Agent Credential
	bind := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents/"+agentID1+"/credentials", csrf, cookie, map[string]any{
		"client_id": "client-1",
	})
	if bind.Code != http.StatusNoContent {
		t.Fatalf("bind credential status=%d body=%s", bind.Code, bind.Body.String())
	}

	// Unbind Agent Credential
	unbind := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/agents/"+agentID1+"/credentials/client-1", csrf, cookie, nil)
	if unbind.Code != http.StatusNoContent {
		t.Fatalf("unbind credential status=%d body=%s", unbind.Code, unbind.Body.String())
	}

	// Delete Agent
	deleteAgent := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/agents/"+agentID1, csrf, cookie, nil)
	if deleteAgent.Code != http.StatusNoContent {
		t.Fatalf("delete agent status=%d body=%s", deleteAgent.Code, deleteAgent.Body.String())
	}

	// --- 2. Agent 2 (Create, Kill) ---
	create2 := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": "Agent 2", "description": &desc, "kind": &kind, "roles": []string{"support"},
	})
	if create2.Code != http.StatusCreated {
		t.Fatalf("register agent 2 status=%d body=%s", create2.Code, create2.Body.String())
	}
	var created2 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(create2.Body.Bytes(), &created2)
	agentID2 := created2.ID

	// Kill Agent
	kill := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents/"+agentID2+"/kill", csrf, cookie, nil)
	if kill.Code != http.StatusNoContent {
		t.Fatalf("kill agent status=%d body=%s", kill.Code, kill.Body.String())
	}
}

func TestAdminGroupLifecycleExtra(t *testing.T) {
	e := newIdentityTestHandler(t).echo
	csrf, cookie := adminCSRF(t, e)

	// Create Group
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{
		"name": "group-1", "roles": []string{"role-1"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create group status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	groupID := created.ID

	// Get Group
	get := adminJSONRequest(t, e, http.MethodGet, "/api/admin/groups/"+groupID, csrf, cookie, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get group status=%d body=%s", get.Code, get.Body.String())
	}

	// Update Group
	newName := "group-1-updated"
	update := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/groups/"+groupID, csrf, cookie, map[string]any{
		"name": &newName, "roles": []string{"role-2"},
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update group status=%d body=%s", update.Code, update.Body.String())
	}

	// Add Group Member
	add := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+groupID+"/members/regular", csrf, cookie, nil)
	if add.Code != http.StatusNoContent {
		t.Fatalf("add member status=%d body=%s", add.Code, add.Body.String())
	}

	// Get Group with Members
	getWithMember := adminJSONRequest(t, e, http.MethodGet, "/api/admin/groups/"+groupID, csrf, cookie, nil)
	if getWithMember.Code != http.StatusOK {
		t.Fatalf("get group with member status=%d body=%s", getWithMember.Code, getWithMember.Body.String())
	}

	// List User Groups
	userGroups := adminJSONRequest(t, e, http.MethodGet, "/api/admin/users/regular/groups", csrf, cookie, nil)
	if userGroups.Code != http.StatusOK {
		t.Fatalf("list user groups status=%d body=%s", userGroups.Code, userGroups.Body.String())
	}

	// Remove Group Member
	remove := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/groups/"+groupID+"/members/regular", csrf, cookie, nil)
	if remove.Code != http.StatusNoContent {
		t.Fatalf("remove member status=%d body=%s", remove.Code, remove.Body.String())
	}

	// Delete Group
	del := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/groups/"+groupID, csrf, cookie, nil)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete group status=%d body=%s", del.Code, del.Body.String())
	}
}

func TestAdminUserLifecycleExtra(t *testing.T) {
	e := newIdentityTestHandler(t).echo
	csrf, cookie := adminCSRF(t, e)

	// Create User
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/users", csrf, cookie, map[string]any{
		"preferred_username": "user1",
		"password":           "password-123",
		"email":              "user1@example.com",
		"roles":              []string{"role-1"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	userID := created.ID

	// Get User
	get := adminJSONRequest(t, e, http.MethodGet, "/api/admin/users/"+userID, csrf, cookie, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get user status=%d body=%s", get.Code, get.Body.String())
	}

	// Update User
	newUsername := "user1-updated"
	newEmail := "user1-updated@example.com"
	update := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/users/"+userID, csrf, cookie, map[string]any{
		"preferred_username": &newUsername,
		"email":              &newEmail,
		"roles":              []string{"role-2"},
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update user status=%d body=%s", update.Code, update.Body.String())
	}

	// Disable User
	disable := adminJSONRequest(t, e, http.MethodPost, "/api/admin/users/"+userID+"/disable", csrf, cookie, nil)
	if disable.Code != http.StatusNoContent {
		t.Fatalf("disable user status=%d body=%s", disable.Code, disable.Body.String())
	}

	// Enable User
	enable := adminJSONRequest(t, e, http.MethodPost, "/api/admin/users/"+userID+"/enable", csrf, cookie, nil)
	if enable.Code != http.StatusNoContent {
		t.Fatalf("enable user status=%d body=%s", enable.Code, enable.Body.String())
	}
}

func TestAccountDataExport(t *testing.T) {
	h := newIdentityTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)

	// Seed client and consent
	clientName := "Client One"
	client := &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-1", ClientName: &clientName,
	}
	h.clients.Seed(client)

	now := time.Now().UTC()
	consent := &oauthdomain.Consent{
		UserID: "regular", ClientID: "client-1",
		Scopes: []string{"openid", "profile"}, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
		State: oauthdomain.ConsentGranted,
	}
	_ = h.consents.Save(context.Background(), tenancydomain.DefaultTenantID, consent)

	// Test Export for regular user
	request := httptest.NewRequest(http.MethodGet, "/api/account/data_export", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("export data status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAccountProfileHTTPExtra(t *testing.T) {
	e := newIdentityTestHandler(t).echo
	csrf, cookie := adminCSRF(t, e)

	summary := adminJSONRequest(t, e, http.MethodGet, "/api/account/summary", csrf, cookie, nil)
	if summary.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", summary.Code, summary.Body.String())
	}

	profile := adminJSONRequest(t, e, http.MethodGet, "/api/account/profile", csrf, cookie, nil)
	if profile.Code != http.StatusOK {
		t.Fatalf("profile status=%d body=%s", profile.Code, profile.Body.String())
	}

	givenName := "Admin"
	update := adminJSONRequest(t, e, http.MethodPatch, "/api/account/profile", csrf, cookie, map[string]any{
		"given_name": &givenName,
	})
	if update.Code != http.StatusOK {
		t.Fatalf("profile update status=%d body=%s", update.Code, update.Body.String())
	}

	attrs := map[string]userdomain.AttributeValue{
		"not_a_real_attribute": {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	invalidAttr := adminJSONRequest(t, e, http.MethodPatch, "/api/account/profile", csrf, cookie, map[string]any{
		"attributes": attrs,
	})
	if invalidAttr.Code != http.StatusBadRequest {
		t.Fatalf("invalid attr status=%d body=%s", invalidAttr.Code, invalidAttr.Body.String())
	}

	invalidJSON := adminJSONRequest(t, e, http.MethodPatch, "/api/account/profile", csrf, cookie, "invalid-json")
	if invalidJSON.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status=%d body=%s", invalidJSON.Code, invalidJSON.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/api/account/profile", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated profile status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEmailChangeLifecycle(t *testing.T) {
	h := newIdentityTestHandler(t)
	e := h.echo
	tokenStore := h.tokens
	csrf, cookie := adminCSRF(t, e)

	// GET verify context
	ctxReq := httptest.NewRequest(http.MethodGet, "/api/account/email/verify_context", http.NoBody)
	ctxRes := httptest.NewRecorder()
	e.ServeHTTP(ctxRes, ctxReq)
	if ctxRes.Code != http.StatusOK {
		t.Fatalf("verify context status=%d body=%s", ctxRes.Code, ctxRes.Body.String())
	}

	// POST email change request
	changeReq := adminJSONRequest(t, e, http.MethodPost, "/api/account/email/change_request", csrf, cookie, map[string]any{
		"new_email": "admin-new@example.com",
	})
	if changeReq.Code != http.StatusNoContent {
		t.Fatalf("request email change status=%d body=%s", changeReq.Code, changeReq.Body.String())
	}

	// Seed a valid token for verify
	rawToken := "my-raw-email-change-token-456"
	tokenHash := sha256Hex(rawToken)
	_ = tokenStore.Save(context.Background(), authnports.EmailChangeTokenRecord{
		TokenHash: tokenHash,
		Sub:       "admin",
		NewEmail:  "admin-new@example.com",
		ExpiresAt: time.Now().Add(time.Hour).UTC(),
	})

	// POST confirm email change
	verifyReq := adminJSONRequest(t, e, http.MethodPost, "/api/account/email/verify", csrf, cookie, map[string]any{
		"token": rawToken,
	})
	if verifyReq.Code != http.StatusOK {
		t.Fatalf("confirm email change status=%d body=%s", verifyReq.Code, verifyReq.Body.String())
	}
}

func TestIdentityAPIErrors(t *testing.T) {
	h := newIdentityTestHandler(t)
	e := h.echo
	repo := h.users
	groupRepo := h.groups
	csrf, cookie := adminCSRF(t, e)

	// Set regular user email for email change testing
	email := "regular@example.com"
	regularUser, _ := repo.FindBySub(context.Background(), "regular")
	regularUser.Email = &email
	regularUser.EmailVerified = true
	_ = repo.Save(context.Background(), regularUser)

	// Seed a group
	group := &groupdomain.Group{
		ID: "group-1", Name: "Group One", TenantID: tenancydomain.DefaultTenantID,
	}
	_ = groupRepo.Save(context.Background(), group)

	// --- 1. User Errors ---
	// 404 User Not Found
	getUsr := adminJSONRequest(t, e, http.MethodGet, "/api/admin/users/ghost-user", csrf, cookie, nil)
	if getUsr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for ghost user, got %d", getUsr.Code)
	}

	// Username conflict (409)
	createUsrDupUsername := adminJSONRequest(t, e, http.MethodPost, "/api/admin/users", csrf, cookie, map[string]any{
		"preferred_username": "regular",
		"password":           "password-123",
		"email":              "unique-email@example.com",
	})
	if createUsrDupUsername.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d", createUsrDupUsername.Code)
	}

	// --- 2. Group Errors ---
	// 404 Group Not Found
	getGrp := adminJSONRequest(t, e, http.MethodGet, "/api/admin/groups/ghost-group", csrf, cookie, nil)
	if getGrp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for ghost group, got %d", getGrp.Code)
	}

	// Add member user not found (404)
	addUsrGhost := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/group-1/members/ghost-user", csrf, cookie, nil)
	if addUsrGhost.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for adding ghost user to group, got %d", addUsrGhost.Code)
	}

	// Group name conflict (409)
	createGrpDup := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{
		"name": "Group One",
	})
	if createGrpDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate group name, got %d", createGrpDup.Code)
	}
	updateGrpBlank := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/groups/group-1", csrf, cookie, map[string]any{
		"name": " ",
	})
	if updateGrpBlank.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank group name, got %d", updateGrpBlank.Code)
	}
	updateGrpBadRole := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/groups/group-1", csrf, cookie, map[string]any{
		"roles": []string{" "},
	})
	if updateGrpBadRole.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid group role, got %d", updateGrpBadRole.Code)
	}

	// --- 3. Agent Errors ---
	// 404 Agent Not Found
	getAgt := adminJSONRequest(t, e, http.MethodGet, "/api/admin/agents/ghost-agent", csrf, cookie, nil)
	if getAgt.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for ghost agent, got %d", getAgt.Code)
	}

	// Agent name required (400)
	createAgtBlank := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": " ",
	})
	if createAgtBlank.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank agent name, got %d", createAgtBlank.Code)
	}

	// Agent owner not found (400)
	missingOwner := "ghost-user"
	createAgtOwnerMissing := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": "OwnerMissingAgent", "owner_user_id": &missingOwner,
	})
	if createAgtOwnerMissing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing owner, got %d", createAgtOwnerMissing.Code)
	}

	// Agent name conflict (409)
	descAg := "agent description"
	kindAg := "daemon"
	createAgtSeed := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": "DuplicateAgent", "description": &descAg, "kind": &kindAg,
	})
	if createAgtSeed.Code != http.StatusCreated {
		t.Fatalf("seed agent status=%d body=%s", createAgtSeed.Code, createAgtSeed.Body.String())
	}
	var seedAgent struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createAgtSeed.Body.Bytes(), &seedAgent); err != nil {
		t.Fatal(err)
	}
	createAgtDup := adminJSONRequest(t, e, http.MethodPost, "/api/admin/agents", csrf, cookie, map[string]any{
		"name": "DuplicateAgent", "description": &descAg, "kind": &kindAg,
	})
	if createAgtDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate agent name, got %d", createAgtDup.Code)
	}

	updateAgtBadRole := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/agents/"+seedAgent.ID, csrf, cookie, map[string]any{
		"roles": []string{" "},
	})
	if updateAgtBadRole.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid agent role, got %d", updateAgtBadRole.Code)
	}

	// --- 4. Email Change Errors ---
	// Invalid email format (400)
	badEmailReq := adminJSONRequest(t, e, http.MethodPost, "/api/account/email/change_request", csrf, cookie, map[string]any{
		"new_email": "invalid-email-format",
	})
	if badEmailReq.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad email, got %d", badEmailReq.Code)
	}

	// Unchanged email (400)
	unchangedReq := httptest.NewRequest(
		http.MethodPost,
		"/api/account/email/change_request",
		bytes.NewBufferString(`{"new_email":"regular@example.com"}`),
	)
	unchangedReq.Header.Set("Content-Type", "application/json")
	unchangedReq.Header.Set("Origin", "http://idp.test")
	unchangedReq.Header.Set("X-Csrf-Token", csrf)
	unchangedReq.Header.Set("X-Demo-Sub", "regular")
	unchangedReq.AddCookie(cookie)
	unchangedRes := httptest.NewRecorder()
	e.ServeHTTP(unchangedRes, unchangedReq)
	if unchangedRes.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unchanged email, got %d body=%s", unchangedRes.Code, unchangedRes.Body.String())
	}

	// Email taken (409)
	takenEmailReq := adminJSONRequest(t, e, http.MethodPost, "/api/account/email/change_request", csrf, cookie, map[string]any{
		"new_email": "regular@example.com",
	})
	if takenEmailReq.Code != http.StatusConflict {
		t.Fatalf("expected 409 for email taken, got %d", takenEmailReq.Code)
	}

	// Invalid verify token (410)
	badVerifyReq := adminJSONRequest(t, e, http.MethodPost, "/api/account/email/verify", csrf, cookie, map[string]any{
		"token": "invalid-token",
	})
	if badVerifyReq.Code != http.StatusGone {
		t.Fatalf("expected 410 for invalid token, got %d", badVerifyReq.Code)
	}

	// --- 5. CSRF / Browser Verification Errors ---
	// 403 Forbidden on missing CSRF header
	noCsrfReq := adminJSONRequest(t, e, http.MethodPost, "/api/admin/users", "", cookie, map[string]any{})
	if noCsrfReq.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", noCsrfReq.Code)
	}
}
