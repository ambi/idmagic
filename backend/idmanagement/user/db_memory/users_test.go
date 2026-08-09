package db_memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestUserRepositoryListPage(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepository()
	for i, name := range []string{"charlie", "alice", "bob", "delta", "echo"} {
		repo.Seed(&userdomain.User{
			ID: fmt.Sprintf("user-%d", i), TenantID: "tenant-1", PreferredUsername: name,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now(),
		})
	}
	repo.Seed(&userdomain.User{
		ID: "user-deleted", TenantID: "tenant-1", PreferredUsername: "banana",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDeleted}, CreatedAt: time.Now(),
	})
	repo.Seed(&userdomain.User{
		ID: "user-other-tenant", TenantID: "tenant-2", PreferredUsername: "zzz",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now(),
	})

	t.Run("first page respects limit and (preferred_username, id) order", func(t *testing.T) {
		page, err := repo.ListPage(ctx, "tenant-1", "", "", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 2 || page[0].PreferredUsername != "alice" || page[1].PreferredUsername != "bob" {
			t.Fatalf("unexpected first page: %+v", page)
		}
	})

	t.Run("continuation page resumes strictly after the keyset", func(t *testing.T) {
		first, err := repo.ListPage(ctx, "tenant-1", "", "", 2)
		if err != nil {
			t.Fatal(err)
		}
		last := first[len(first)-1]
		next, err := repo.ListPage(ctx, "tenant-1", last.PreferredUsername, last.ID, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(next) != 2 || next[0].PreferredUsername != "charlie" || next[1].PreferredUsername != "delta" {
			t.Fatalf("unexpected continuation page: %+v", next)
		}
	})

	t.Run("excludes deleted users and other tenants", func(t *testing.T) {
		page, err := repo.ListPage(ctx, "tenant-1", "", "", 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 5 {
			t.Fatalf("expected 5 active tenant-1 users, got %d", len(page))
		}
		for _, u := range page {
			if u.TenantID != "tenant-1" {
				t.Fatalf("leaked cross-tenant user %s", u.ID)
			}
		}
	})
}

func TestUserRepositoryListPageFilteredSearchesTenantWideAndSupportsPrevious(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepository()
	name := "Alice Example"
	email := "Alice@Example.COM"
	for _, user := range []*userdomain.User{
		{ID: "00000000-0000-4000-8000-000000000001", TenantID: "tenant-1", PreferredUsername: "alpha", Name: &name, Email: &email, Roles: []string{"billing-admin"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}},
		{ID: "00000000-0000-4000-8000-000000000002", TenantID: "tenant-1", PreferredUsername: "bravo", Roles: []string{"reader"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}},
		{ID: "00000000-0000-4000-8000-000000000003", TenantID: "tenant-2", PreferredUsername: "alice-other", Roles: []string{"billing-admin"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}},
	} {
		repo.Seed(user)
	}
	active := idmdomain.UserStatusActive
	page, err := repo.ListPageFiltered(ctx, "tenant-1", "EXAMPLE.com", &active, "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].PreferredUsername != "alpha" {
		t.Fatalf("unexpected tenant-wide filtered result: %+v", page)
	}
	rolePage, err := repo.ListPageFiltered(ctx, "tenant-1", "BILLING", nil, "", "", 10)
	if err != nil || len(rolePage) != 1 || rolePage[0].PreferredUsername != "alpha" {
		t.Fatalf("role search result=%+v err=%v", rolePage, err)
	}
	previous, err := repo.ListPageBeforeFiltered(ctx, "tenant-1", "", nil, "bravo", "00000000-0000-4000-8000-000000000002", 10)
	if err != nil || len(previous) != 1 || previous[0].PreferredUsername != "alpha" {
		t.Fatalf("previous filtered page=%+v err=%v", previous, err)
	}
}

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepository()

	email1 := "user1@example.com"
	email2 := "bob@EXAMPLE.COM" // 大文字混じり

	t.Run("Save and FindBySub", func(t *testing.T) {
		user := &userdomain.User{
			ID:                "user-1",
			TenantID:          "tenant-1",
			PreferredUsername: "alice",
			Email:             &email1,
			CreatedAt:         time.Now(),
		}

		err := repo.Save(ctx, user)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected user to be found")
		}
		if found.PreferredUsername != "alice" {
			t.Errorf("expected PreferredUsername to be 'alice', got %q", found.PreferredUsername)
		}

		// 存在しない Sub
		notfound, err := repo.FindBySub(ctx, "user-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing sub")
		}
	})

	t.Run("Seed", func(t *testing.T) {
		user := &userdomain.User{
			ID:                "user-seeded",
			TenantID:          "tenant-1",
			PreferredUsername: "seeded",
		}
		//nolint:contextcheck // memory repo Seed doesn't take context
		repo.Seed(user)

		found, _ := repo.FindBySub(ctx, "user-seeded")
		if found == nil {
			t.Fatal("expected seeded user to be found")
		}
	})

	t.Run("Username Change indexing", func(t *testing.T) {
		user, _ := repo.FindBySub(ctx, "user-1")
		// コピーを作成して PreferredUsername を変更
		newUser := *user
		newUser.PreferredUsername = "alice-new"

		err := repo.Save(ctx, &newUser)
		if err != nil {
			t.Fatal(err)
		}

		// 新しいユーザー名で検索できること
		foundNew, _ := repo.FindByUsername(ctx, "tenant-1", "alice-new")
		if foundNew == nil {
			t.Error("expected user to be found by new username")
		}

		// 旧ユーザー名では検索できないこと
		foundOld, _ := repo.FindByUsername(ctx, "tenant-1", "alice")
		if foundOld != nil {
			t.Error("expected user not to be found by old username")
		}
	})

	t.Run("FindByEmail with Case Insensitivity and Tenant isolation", func(t *testing.T) {
		user2 := &userdomain.User{
			ID:                "user-2",
			TenantID:          "tenant-1",
			PreferredUsername: "bob",
			Email:             &email2,
		}
		userOtherTenant := &userdomain.User{
			ID:                "user-other-tenant",
			TenantID:          "tenant-other",
			PreferredUsername: "alice-new",
			Email:             &email1, // aliceと同じEmail
		}

		_ = repo.Save(ctx, user2)
		_ = repo.Save(ctx, userOtherTenant)

		// 大文字小文字無視 (bob@EXAMPLE.COM に対して bob@example.com で検索)
		foundBob, err := repo.FindByEmail(ctx, "tenant-1", "bob@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if foundBob == nil || foundBob.ID != "user-2" {
			t.Errorf("expected to find user-2 by lowercase email, got %v", foundBob)
		}

		// テナント分離
		foundAliceOther, _ := repo.FindByEmail(ctx, "tenant-other", "user1@example.com")
		if foundAliceOther == nil || foundAliceOther.ID != "user-other-tenant" {
			t.Errorf("expected to find user-other-tenant, got %v", foundAliceOther)
		}

		// Email が nil のユーザー（user-seeded）がいる状態で検索し、エラーにならないこと
		// 存在しないメールアドレスの検索
		foundNone, err := repo.FindByEmail(ctx, "tenant-1", "none@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if foundNone != nil {
			t.Error("expected nil for non-existing email")
		}
	})

	t.Run("Deleted User behaviors", func(t *testing.T) {
		userDel := &userdomain.User{
			ID:                "user-deleted",
			TenantID:          "tenant-1",
			PreferredUsername: "charlie",
			Email:             &email1,
			Lifecycle: userdomain.UserLifecycle{
				Status: idmdomain.UserStatusDeleted,
			},
		}
		_ = repo.Save(ctx, userDel)

		// FindBySub では見つからないこと
		found, err := repo.FindBySub(ctx, "user-deleted")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected FindBySub to return nil for deleted user")
		}

		// FindBySubIncludingDeleted では見つかること
		foundInc, err := repo.FindBySubIncludingDeleted(ctx, "user-deleted")
		if err != nil {
			t.Fatal(err)
		}
		if foundInc == nil || foundInc.ID != "user-deleted" {
			t.Error("expected FindBySubIncludingDeleted to find deleted user")
		}

		// FindByUsername では見つからないこと
		foundName, _ := repo.FindByUsername(ctx, "tenant-1", "charlie")
		if foundName != nil {
			t.Error("expected FindByUsername to return nil for deleted user")
		}

		// FindByEmail では見つからないこと
		foundEmail, _ := repo.FindByEmail(ctx, "tenant-1", "user1@example.com")
		// (alice@example.com もいるが、charlie にヒットしないこと。かつ、charlie が先にループで評価されて無視されることを検証)
		if foundEmail != nil && foundEmail.ID == "user-deleted" {
			t.Error("expected FindByEmail to ignore deleted user")
		}
	})

	t.Run("FindAll and Sort", func(t *testing.T) {
		// tenant-1 には現在：
		// - user-1 (alice-new)
		// - user-seeded (seeded)
		// - user-2 (bob)
		// - user-deleted (charlie, status deleted なので除外されるべき)

		list, err := repo.FindAll(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// 3 件あるはず
		if len(list) != 3 {
			t.Fatalf("expected 3 active users, got %d", len(list))
		}
		// PreferredUsername 順（alice-new, bob, seeded）でソートされていることを検証
		if list[0].PreferredUsername != "alice-new" || list[1].PreferredUsername != "bob" || list[2].PreferredUsername != "seeded" {
			t.Errorf("list is not sorted by PreferredUsername: %s, %s, %s", list[0].PreferredUsername, list[1].PreferredUsername, list[2].PreferredUsername)
		}
	})
}
