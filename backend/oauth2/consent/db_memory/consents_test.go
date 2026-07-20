package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestConsentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewConsentRepository()

	t.Run("Save and Find", func(t *testing.T) {
		consent := &domain.Consent{
			UserID:    "user-1",
			ClientID:  "client-1",
			Scopes:    []string{"read", "write"},
			State:     domain.ConsentGranted,
			GrantedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err := repo.Save(ctx, "tenant-1", consent)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.Find(ctx, "tenant-1", "user-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected consent to be found")
		}
		if found.UserID != "user-1" || found.ClientID != "client-1" {
			t.Errorf("unexpected found user/client: %s / %s", found.UserID, found.ClientID)
		}
		if len(found.Scopes) != 2 || found.Scopes[0] != "read" || found.Scopes[1] != "write" {
			t.Errorf("unexpected scopes: %v", found.Scopes)
		}

		// 存在しない consent
		notfound, err := repo.Find(ctx, "tenant-1", "user-none", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing consent")
		}
	})

	t.Run("FindAll and Sort", func(t *testing.T) {
		// すでに user-1 / client-1 が tenant-1 にある
		c2 := &domain.Consent{UserID: "user-2", ClientID: "client-2", State: domain.ConsentGranted}
		c3 := &domain.Consent{UserID: "user-1", ClientID: "client-2", State: domain.ConsentGranted}
		cOther := &domain.Consent{UserID: "user-1", ClientID: "client-1", State: domain.ConsentGranted}

		_ = repo.Save(ctx, "tenant-1", c2)
		_ = repo.Save(ctx, "tenant-1", c3)
		_ = repo.Save(ctx, "tenant-other", cOther)

		list, err := repo.FindAll(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-1 には 3 件あるはず (user-1/client-1, user-2/client-2, user-1/client-2)
		if len(list) != 3 {
			t.Fatalf("expected 3 consents, got %d", len(list))
		}

		// ソート順の検証: UserID, ClientID の順
		// 期待される順:
		// 1: user-1, client-1
		// 2: user-1, client-2
		// 3: user-2, client-2
		if list[0].UserID != "user-1" || list[0].ClientID != "client-1" {
			t.Errorf("expected index 0 to be user-1/client-1, got %s/%s", list[0].UserID, list[0].ClientID)
		}
		if list[1].UserID != "user-1" || list[1].ClientID != "client-2" {
			t.Errorf("expected index 1 to be user-1/client-2, got %s/%s", list[1].UserID, list[1].ClientID)
		}
		if list[2].UserID != "user-2" || list[2].ClientID != "client-2" {
			t.Errorf("expected index 2 to be user-2/client-2, got %s/%s", list[2].UserID, list[2].ClientID)
		}
	})

	t.Run("Revoke", func(t *testing.T) {
		// 存在しない consent の Revoke
		err := repo.Revoke(ctx, "tenant-1", "user-none", "client-1")
		if err != nil {
			t.Fatal(err)
		}

		// 存在する consent の Revoke
		err = repo.Revoke(ctx, "tenant-1", "user-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.Find(ctx, "tenant-1", "user-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if found.State != domain.ConsentRevoked {
			t.Errorf("expected state to be ConsentRevoked, got %v", found.State)
		}
		if found.RevokedAt == nil {
			t.Error("expected RevokedAt to be set")
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		// user-1 に対する全ての consent を削除
		err := repo.DeleteAllForSub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}

		// user-1 の consent は tenant-1 からも tenant-other からも消えているはず
		list1, _ := repo.FindAll(ctx, "tenant-1")
		for _, c := range list1 {
			if c.UserID == "user-1" {
				t.Error("expected user-1 consents to be deleted in tenant-1")
			}
		}

		listOther, _ := repo.FindAll(ctx, "tenant-other")
		for _, c := range listOther {
			if c.UserID == "user-1" {
				t.Error("expected user-1 consents to be deleted in tenant-other")
			}
		}

		// user-2 の consent は残っているはず
		found, _ := repo.Find(ctx, "tenant-1", "user-2", "client-2")
		if found == nil {
			t.Error("expected user-2 consent to remain")
		}
	})
}
