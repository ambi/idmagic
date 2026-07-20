package db_memory

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/scim/ports"
)

func TestScimRepositoryTokens(t *testing.T) {
	ctx := context.Background()
	repo := NewScimRepository()

	token := &ports.ScimToken{
		ID: "t1", TenantID: "acme", TokenHash: "hash-1", Description: "prov",
	}
	if err := repo.SaveToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveToken(ctx, &ports.ScimToken{ID: "t2", TenantID: "acme", TokenHash: "hash-2"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveToken(ctx, &ports.ScimToken{ID: "t3", TenantID: "other", TokenHash: "hash-3"}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindToken(ctx, "hash-1")
	if err != nil || got == nil || got.ID != "t1" {
		t.Fatalf("FindToken: %v got=%v", err, got)
	}
	if missing, _ := repo.FindToken(ctx, "nope"); missing != nil {
		t.Fatalf("expected nil token, got %+v", missing)
	}

	list, err := repo.ListTokens(ctx, "acme")
	if err != nil || len(list) != 2 {
		t.Fatalf("ListTokens: %v len=%d", err, len(list))
	}

	// 別テナントの id では削除されない。
	if err := repo.DeleteToken(ctx, "other", "t1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.FindToken(ctx, "hash-1"); got == nil {
		t.Fatal("token deleted across tenant boundary")
	}
	if err := repo.DeleteToken(ctx, "acme", "t1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.FindToken(ctx, "hash-1"); got != nil {
		t.Fatal("token not deleted")
	}
}

func TestScimRepositoryUserRefs(t *testing.T) {
	ctx := context.Background()
	repo := NewScimRepository()

	ref := &ports.ScimUserRef{TenantID: "acme", ScimID: "s1", UserID: "u1"}
	if err := repo.SaveUserRef(ctx, ref); err != nil {
		t.Fatal(err)
	}

	byScim, err := repo.FindUserRefByScimID(ctx, "acme", "s1")
	if err != nil || byScim == nil || byScim.UserID != "u1" {
		t.Fatalf("FindUserRefByScimID: %v got=%v", err, byScim)
	}
	byUser, err := repo.FindUserRefByUserID(ctx, "acme", "u1")
	if err != nil || byUser == nil || byUser.ScimID != "s1" {
		t.Fatalf("FindUserRefByUserID: %v got=%v", err, byUser)
	}

	// 未知テナント / 未知 id は nil。
	if r, _ := repo.FindUserRefByScimID(ctx, "unknown", "s1"); r != nil {
		t.Fatal("expected nil for unknown tenant")
	}
	if r, _ := repo.FindUserRefByScimID(ctx, "acme", "nope"); r != nil {
		t.Fatal("expected nil for unknown scim id")
	}
	if r, _ := repo.FindUserRefByUserID(ctx, "acme", "nope"); r != nil {
		t.Fatal("expected nil for unknown user id")
	}
	if r, _ := repo.FindUserRefByUserID(ctx, "unknown", "u1"); r != nil {
		t.Fatal("expected nil for unknown tenant")
	}

	if err := repo.DeleteUserRef(ctx, "acme", "s1"); err != nil {
		t.Fatal(err)
	}
	if r, _ := repo.FindUserRefByScimID(ctx, "acme", "s1"); r != nil {
		t.Fatal("user ref not deleted")
	}
	// 未知テナントの削除は no-op。
	if err := repo.DeleteUserRef(ctx, "unknown", "s1"); err != nil {
		t.Fatal(err)
	}
}

func TestScimRepositoryGroupRefs(t *testing.T) {
	ctx := context.Background()
	repo := NewScimRepository()

	ref := &ports.ScimGroupRef{TenantID: "acme", ScimID: "s1", GroupID: "g1"}
	if err := repo.SaveGroupRef(ctx, ref); err != nil {
		t.Fatal(err)
	}

	byScim, err := repo.FindGroupRefByScimID(ctx, "acme", "s1")
	if err != nil || byScim == nil || byScim.GroupID != "g1" {
		t.Fatalf("FindGroupRefByScimID: %v got=%v", err, byScim)
	}
	byGroup, err := repo.FindGroupRefByGroupID(ctx, "acme", "g1")
	if err != nil || byGroup == nil || byGroup.ScimID != "s1" {
		t.Fatalf("FindGroupRefByGroupID: %v got=%v", err, byGroup)
	}

	if r, _ := repo.FindGroupRefByScimID(ctx, "unknown", "s1"); r != nil {
		t.Fatal("expected nil for unknown tenant")
	}
	if r, _ := repo.FindGroupRefByScimID(ctx, "acme", "nope"); r != nil {
		t.Fatal("expected nil for unknown scim id")
	}
	if r, _ := repo.FindGroupRefByGroupID(ctx, "acme", "nope"); r != nil {
		t.Fatal("expected nil for unknown group id")
	}
	if r, _ := repo.FindGroupRefByGroupID(ctx, "unknown", "g1"); r != nil {
		t.Fatal("expected nil for unknown tenant")
	}

	if err := repo.DeleteGroupRef(ctx, "acme", "s1"); err != nil {
		t.Fatal(err)
	}
	if r, _ := repo.FindGroupRefByScimID(ctx, "acme", "s1"); r != nil {
		t.Fatal("group ref not deleted")
	}
	if err := repo.DeleteGroupRef(ctx, "unknown", "s1"); err != nil {
		t.Fatal(err)
	}
}
