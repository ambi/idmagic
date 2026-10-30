package server_http_test

// Agent の棚卸しを、製品と同じ `Register` の組み立てで確かめる。所見は IdManagement の
// Agent と User、OAuth2 のクライアント、Audit の検索属性を読んで初めて決まるので、
// どれかの配線が欠けると所見の欠落か誤りとして現れる。
//
// 最終のトークン発行は `/token` の client_credentials を実際に通して作る。保存先へ直接
// 書くと、発行イベントが agentId を運び、監査の検索属性へ届く経路を通らない。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type inventoryEntry struct {
	Agent struct {
		ID string `json:"id"`
	} `json:"agent"`
	OwnerStatus       string   `json:"owner_status"`
	LastTokenIssuedAt *string  `json:"last_token_issued_at"`
	MissingClientIDs  []string `json:"missing_client_ids"`
	Findings          []string `json:"findings"`
}

type inventoryResponse struct {
	DormantDays int              `json:"dormant_days"`
	Agents      []inventoryEntry `json:"agents"`
}

func inventoryRequest(t *testing.T, s *stack.Stack, token, query string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, stack.Issuer+"/realms/default/api/admin/v1/agent-inventory"+query, http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func decodeInventory(t *testing.T, recorder *httptest.ResponseRecorder) map[string]inventoryEntry {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("inventory status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body inventoryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	entries := map[string]inventoryEntry{}
	for _, entry := range body.Agents {
		entries[entry.Agent.ID] = entry
	}
	return entries
}

// 所見ごとに、その所見だけを持つ Agent と、どの所見も持たない対照を置く。
//
//   - healthy：所有者が Active、束縛あり、200 日前に登録したが今トークンを得た。休眠でない対照。
//   - dormant：同じ条件で、トークンを一度も得ていない。
//   - orphaned：所有者の User が Disabled。登録は今で、休眠にはならない。
//   - ghost：束縛先のクライアントが存在しない。
//   - bare：束縛がない。
//   - stopped：200 日前に登録して使っていないが Disabled。止めた Agent は休眠にしない。
//
//spec:covers REQ-IDMANAGEMENT-094: 所有者不在、休眠、未束縛の所見、所見による絞り込み、dormant_days の範囲
func TestAgentInventoryReportsOwnerMissingDormantAndUnbound(t *testing.T) {
	s := stack.New(t, stack.WithOAuth2Clients(), stack.WithTokenIssuance(), stack.WithApiTokens(), stack.WithAgentRevocation(), stack.WithAudit())
	f := &agentRevocationFixture{s: s, credentials: map[string]*testing_clientauth.PrivateKeyJWT{}}
	ctx := context.Background()
	now := time.Now().UTC()
	longAgo := now.Add(-200 * 24 * time.Hour)

	if err := s.Users.Save(ctx, &userdomain.User{
		ID: "departed-owner", PreferredUsername: "departed", PasswordHash: "unused", TenantID: tenancydomain.DefaultTenantID,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	seed := func(id, owner string, status idmdomain.AgentStatus, createdAt time.Time, clientID string) {
		t.Helper()
		if err := s.Agents.Save(ctx, &agentdomain.Agent{
			ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
			OwnerUserID: owner, Status: status, CreatedAt: createdAt, UpdatedAt: createdAt,
		}); err != nil {
			t.Fatal(err)
		}
		if clientID == "" {
			return
		}
		if _, err := s.Agents.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: id, ClientID: clientID, CreatedAt: createdAt}); err != nil {
			t.Fatal(err)
		}
	}
	f.seedClient(t, "healthy")
	f.seedClient(t, "dormant")
	f.seedClient(t, "orphaned")
	f.seedClient(t, "stopped")
	seed("healthy", stack.UserID, idmdomain.AgentStatusActive, longAgo, agentClientID("healthy"))
	seed("dormant", stack.UserID, idmdomain.AgentStatusActive, longAgo, agentClientID("dormant"))
	seed("orphaned", "departed-owner", idmdomain.AgentStatusActive, now, agentClientID("orphaned"))
	seed("ghost", stack.UserID, idmdomain.AgentStatusActive, now, "client-that-no-longer-exists")
	seed("bare", stack.UserID, idmdomain.AgentStatusActive, now, "")
	seed("stopped", stack.UserID, idmdomain.AgentStatusDisabled, longAgo, agentClientID("stopped"))
	f.clientCredentials(t, agentClientID("healthy"))

	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsRead)
	entries := decodeInventory(t, inventoryRequest(t, s, adminToken, ""))

	want := map[string][]string{
		"healthy": nil, "dormant": {"dormant"}, "orphaned": {"owner_missing"},
		"ghost": {"unbound"}, "bare": {"unbound"}, "stopped": nil,
	}
	for id, findings := range want {
		entry, ok := entries[id]
		if !ok {
			t.Fatalf("棚卸しに %s が無い: %v", id, entries)
		}
		if !slices.Equal(entry.Findings, findings) {
			t.Errorf("%s の所見=%v、want %v", id, entry.Findings, findings)
		}
	}
	if entries["healthy"].LastTokenIssuedAt == nil {
		t.Error("トークンを得た healthy の last_token_issued_at が無い。発行イベントが agentId を運んでいない")
	}
	if entries["dormant"].LastTokenIssuedAt != nil {
		t.Errorf("発行していない dormant に last_token_issued_at=%v", *entries["dormant"].LastTokenIssuedAt)
	}
	if got := entries["orphaned"].OwnerStatus; got != "disabled" {
		t.Errorf("orphaned の owner_status=%q、want disabled", got)
	}
	if got := entries["ghost"].MissingClientIDs; !slices.Equal(got, []string{"client-that-no-longer-exists"}) {
		t.Errorf("ghost の missing_client_ids=%v", got)
	}

	t.Run("別のテナントの Agent を返さない", func(t *testing.T) {
		if err := s.Agents.Save(ctx, &agentdomain.Agent{
			ID: "foreign", TenantID: stack.OtherRealm, Name: "foreign", Kind: idmdomain.AgentKindAutonomous,
			OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, CreatedAt: longAgo, UpdatedAt: longAgo,
		}); err != nil {
			t.Fatal(err)
		}
		listed := decodeInventory(t, inventoryRequest(t, s, adminToken, ""))
		if _, leaked := listed["foreign"]; leaked {
			t.Fatal("別のテナントの Agent が棚卸しに現れた")
		}
	})
	t.Run("agents:read のないトークンは 403", func(t *testing.T) {
		usersOnly, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeUsersRead)
		if recorder := inventoryRequest(t, s, usersOnly, ""); recorder.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%s、want 403", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("所見で絞り込む", func(t *testing.T) {
		filtered := decodeInventory(t, inventoryRequest(t, s, adminToken, "?finding=unbound"))
		if len(filtered) != 2 || filtered["ghost"].Agent.ID == "" || filtered["bare"].Agent.ID == "" {
			t.Fatalf("unbound の絞り込み=%v、want ghost と bare", filtered)
		}
	})
	t.Run("dormant_days を延ばすと休眠でなくなる", func(t *testing.T) {
		longer := decodeInventory(t, inventoryRequest(t, s, adminToken, "?finding=dormant&dormant_days=365"))
		if len(longer) != 0 {
			t.Fatalf("365 日では休眠でないはずが %v", longer)
		}
	})
	for name, query := range map[string]string{
		"dormant_days が 0":   "?dormant_days=0",
		"dormant_days が 366": "?dormant_days=366",
		"未知の所見":              "?finding=retired",
	} {
		t.Run(name+"は 400", func(t *testing.T) {
			if recorder := inventoryRequest(t, s, adminToken, query); recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s、want 400", recorder.Code, recorder.Body.String())
			}
		})
	}
}
