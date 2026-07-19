package memory

import (
	"context"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func TestAgentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewAgentRepository()

	t.Run("Save and FindByID", func(t *testing.T) {
		agent := &agentdomain.Agent{
			ID:          "agent-1",
			TenantID:    "tenant-a",
			Name:        "Agent A",
			OwnerUserID: "user-1",
			Status:      idmdomain.AgentStatusActive,
			Roles:       []string{"role1", "role2"},
			CreatedAt:   time.Now(),
		}

		err := repo.Save(ctx, agent)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-a", "agent-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected to find agent")
		}
		if found.Name != "Agent A" {
			t.Errorf("expected Name to be 'Agent A', got %q", found.Name)
		}
		if len(found.Roles) != 2 || found.Roles[0] != "role1" || found.Roles[1] != "role2" {
			t.Errorf("unexpected roles: %v", found.Roles)
		}

		// 存在しない ID
		notfound, err := repo.FindByID(ctx, "tenant-a", "agent-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected not to find agent-none")
		}
	})

	t.Run("ListByTenant", func(t *testing.T) {
		// すでに agent-1 (tenant-a, Name: Agent A) がいる
		agent2 := &agentdomain.Agent{
			ID:          "agent-2",
			TenantID:    "tenant-a",
			Name:        "Agent C", // 名前順の検証のため C
			OwnerUserID: "user-1",
			Status:      idmdomain.AgentStatusActive,
		}
		agent3 := &agentdomain.Agent{
			ID:          "agent-3",
			TenantID:    "tenant-a",
			Name:        "Agent B", // 名前順の検証のため B
			OwnerUserID: "user-1",
			Status:      idmdomain.AgentStatusActive,
		}
		agentOther := &agentdomain.Agent{
			ID:          "agent-other",
			TenantID:    "tenant-b",
			Name:        "Agent Other",
			OwnerUserID: "user-1",
			Status:      idmdomain.AgentStatusActive,
		}

		_ = repo.Save(ctx, agent2)
		_ = repo.Save(ctx, agent3)
		_ = repo.Save(ctx, agentOther)

		list, err := repo.ListByTenant(ctx, "tenant-a")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-a には agent-1, agent-2, agent-3 の 3 つがあるはず
		if len(list) != 3 {
			t.Fatalf("expected 3 agents, got %d", len(list))
		}
		// Name の順（Agent A, Agent B, Agent C）でソートされていることを検証
		if list[0].Name != "Agent A" || list[1].Name != "Agent B" || list[2].Name != "Agent C" {
			t.Errorf("list is not sorted by Name: %s, %s, %s", list[0].Name, list[1].Name, list[2].Name)
		}
	})

	t.Run("Bindings Management", func(t *testing.T) {
		// agent-1 (tenant-a), agent-other (tenant-b) が存在する前提
		binding1 := &agentdomain.AgentCredentialBinding{
			AgentID:   "agent-1",
			ClientID:  "client-1",
			CreatedAt: time.Now(),
		}
		binding2 := &agentdomain.AgentCredentialBinding{
			AgentID:   "agent-1",
			ClientID:  "client-2",
			CreatedAt: time.Now(),
		}

		// 成功ケース
		ok, err := repo.AddBinding(ctx, binding1)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected AddBinding to succeed")
		}

		// 重複登録 (同じ Agent & Client) -> 失敗
		ok, err = repo.AddBinding(ctx, binding1)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected AddBinding to fail due to duplicate")
		}

		// 異なる Agent、しかし同じ Client (同 Tenant 内の別 Agent) -> 失敗
		bindingDupClientSameTenant := &agentdomain.AgentCredentialBinding{
			AgentID:   "agent-2",  // tenant-a
			ClientID:  "client-1", // client-1 は既に agent-1 (tenant-a) に紐付けられている
			CreatedAt: time.Now(),
		}
		ok, err = repo.AddBinding(ctx, bindingDupClientSameTenant)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected AddBinding to fail due to duplicate ClientID in same tenant")
		}

		// 異なる Tenant の Agent に対する同じ Client -> 成功
		bindingDupClientDiffTenant := &agentdomain.AgentCredentialBinding{
			AgentID:   "agent-other", // tenant-b
			ClientID:  "client-1",
			CreatedAt: time.Now(),
		}
		ok, err = repo.AddBinding(ctx, bindingDupClientDiffTenant)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected AddBinding to succeed for same ClientID in different tenant")
		}

		// 存在しない AgentID に対する Binding -> 失敗 (tenantID が見つからないため)
		bindingNoAgent := &agentdomain.AgentCredentialBinding{
			AgentID:   "agent-not-exist",
			ClientID:  "client-3",
			CreatedAt: time.Now(),
		}
		ok, err = repo.AddBinding(ctx, bindingNoAgent)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected AddBinding to fail for non-existing agent")
		}

		// ２つ目の正常登録
		ok, err = repo.AddBinding(ctx, binding2)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected AddBinding to succeed for second binding")
		}

		// ListBindings の検証 (ソート順含め)
		list, err := repo.ListBindings(ctx, "tenant-a", "agent-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 bindings, got %d", len(list))
		}
		// ClientID 順にソートされていること
		if list[0].ClientID != "client-1" || list[1].ClientID != "client-2" {
			t.Errorf("list is not sorted by ClientID: %s, %s", list[0].ClientID, list[1].ClientID)
		}

		// FindByClientID の検証
		foundAgent, err := repo.FindByClientID(ctx, "tenant-a", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if foundAgent == nil || foundAgent.ID != "agent-1" {
			t.Errorf("expected to find agent-1, got %v", foundAgent)
		}

		// TenantID が不一致の場合は見つからないこと
		foundAgentDiffTenant, err := repo.FindByClientID(ctx, "tenant-diff", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if foundAgentDiffTenant != nil {
			t.Error("expected not to find agent with mismatched tenant ID")
		}

		// RemoveBinding の検証
		removed, err := repo.RemoveBinding(ctx, "tenant-a", "agent-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Error("expected RemoveBinding to succeed")
		}

		// 存在しない binding の削除
		removed, err = repo.RemoveBinding(ctx, "tenant-a", "agent-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Error("expected RemoveBinding to fail for already removed binding")
		}

		// 削除後のリスト確認
		list, _ = repo.ListBindings(ctx, "tenant-a", "agent-1")
		if len(list) != 1 || list[0].ClientID != "client-2" {
			t.Errorf("expected 1 binding remaining (client-2), got: %v", list)
		}
	})

	t.Run("Delete Agent", func(t *testing.T) {
		// agent-1 の削除
		err := repo.Delete(ctx, "tenant-a", "agent-1")
		if err != nil {
			t.Fatal(err)
		}

		// 検索で見つからないこと
		found, err := repo.FindByID(ctx, "tenant-a", "agent-1")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected agent-1 to be deleted")
		}

		// bindings も削除されていること (Delete メソッドが delete(r.bindings, key) している)
		list, err := repo.ListBindings(ctx, "tenant-a", "agent-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Errorf("expected bindings to be deleted, got %d", len(list))
		}
	})
}
