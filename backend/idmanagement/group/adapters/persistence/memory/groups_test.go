package memory

import (
	"context"
	"testing"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

func TestGroupRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewGroupRepository()

	t.Run("Save and FindByID", func(t *testing.T) {
		group := &groupdomain.Group{
			ID:          "group-1",
			TenantID:    "tenant-1",
			Name:        "Administrators",
			Description: nil,
			Roles:       []string{"admin", "operator"},
			CreatedAt:   time.Now(),
		}

		err := repo.Save(ctx, group)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1", "group-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected group to be found")
		}
		if found.Name != "Administrators" {
			t.Errorf("expected Name to be 'Administrators', got %q", found.Name)
		}
		if len(found.Roles) != 2 || found.Roles[0] != "admin" || found.Roles[1] != "operator" {
			t.Errorf("unexpected roles: %v", found.Roles)
		}

		// 存在しないグループ
		notfound, err := repo.FindByID(ctx, "tenant-1", "group-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing group")
		}
	})

	t.Run("ListByTenant", func(t *testing.T) {
		// すでに group-1 が tenant-1 に存在する
		g2 := &groupdomain.Group{ID: "group-2", TenantID: "tenant-1", Name: "Developers"}
		g3 := &groupdomain.Group{ID: "group-3", TenantID: "tenant-1", Name: "Auditors"}
		gOther := &groupdomain.Group{ID: "group-other", TenantID: "tenant-other", Name: "Other"}

		_ = repo.Save(ctx, g2)
		_ = repo.Save(ctx, g3)
		_ = repo.Save(ctx, gOther)

		list, err := repo.ListByTenant(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-1 には 3 件あるはず
		if len(list) != 3 {
			t.Fatalf("expected 3 groups, got %d", len(list))
		}
		// 名前順でソートされていること (Administrators, Auditors, Developers)
		if list[0].Name != "Administrators" || list[1].Name != "Auditors" || list[2].Name != "Developers" {
			t.Errorf("list is not sorted by Name: %s, %s, %s", list[0].Name, list[1].Name, list[2].Name)
		}
	})

	t.Run("Members Management", func(t *testing.T) {
		member1 := &groupdomain.GroupMember{
			GroupID:   "group-1",
			UserID:    "user-1",
			CreatedAt: time.Now(),
		}
		member2 := &groupdomain.GroupMember{
			GroupID:   "group-1",
			UserID:    "user-2",
			CreatedAt: time.Now(),
		}

		// AddMember 成功
		ok, err := repo.AddMember(ctx, member1)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected AddMember to succeed")
		}

		// 同一メンバーの重複登録 -> 失敗
		ok, err = repo.AddMember(ctx, member1)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected AddMember to fail on duplicate")
		}

		// 2人目のメンバー登録
		_, _ = repo.AddMember(ctx, member2)

		// CountMembers の検証
		count, err := repo.CountMembers(ctx, "tenant-1", "group-1")
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("expected 2 members, got %d", count)
		}

		// ListMembersByGroup の検証 (UserID 順ソート確認)
		members, err := repo.ListMembersByGroup(ctx, "tenant-1", "group-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(members) != 2 {
			t.Fatalf("expected 2 members, got %d", len(members))
		}
		if members[0].UserID != "user-1" || members[1].UserID != "user-2" {
			t.Errorf("expected members sorted by UserID, got: %s, %s", members[0].UserID, members[1].UserID)
		}

		// ListGroupsByUser の検証
		userGroups, err := repo.ListGroupsByUser(ctx, "tenant-1", "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(userGroups) != 1 || userGroups[0].ID != "group-1" {
			t.Errorf("expected user-1 to be in group-1, got %v", userGroups)
		}

		// RemoveMember 成功
		removed, err := repo.RemoveMember(ctx, "tenant-1", "group-1", "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Error("expected RemoveMember to succeed")
		}

		// 存在しないメンバーの削除 -> 失敗
		removed, err = repo.RemoveMember(ctx, "tenant-1", "group-1", "user-none")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Error("expected RemoveMember to fail for non-existing member")
		}
	})

	t.Run("Delete Group and members key fallback", func(t *testing.T) {
		// Group 削除
		err := repo.Delete(ctx, "tenant-1", "group-1")
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1", "group-1")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected group-1 to be deleted")
		}

		// members も削除されていること
		count, _ := repo.CountMembers(ctx, "tenant-1", "group-1")
		if count != 0 {
			t.Errorf("expected member count to be 0 after group delete, got %d", count)
		}

		// memberKey が存在しない groupID に遭遇した場合、その id 自体を返すフォールバック確認
		keyFallback := repo.memberKey("non-existent-group-id")
		if keyFallback != "non-existent-group-id" {
			t.Errorf("expected fallback to return original groupID, got %q", keyFallback)
		}
	})
}
