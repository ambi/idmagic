package memory

import (
	"context"
	"testing"

	appports "github.com/ambi/idmagic/internal/application/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestApplicationRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()

	app := &spec.Application{
		TenantID:      "acme",
		ApplicationID: "app-1",
		Name:          "Zebra",
		Kind:          spec.ApplicationFederated,
		Status:        spec.ApplicationActive,
		Bindings: []spec.ProtocolBinding{
			{Type: spec.ProtocolBindingOIDC, ClientID: "client-1"},
		},
		CategoryIDs: []string{"cat-1", "cat-2"},
	}
	if err := repo.Save(ctx, app); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// 名前順ソートを確認するために 2 件目を追加する。
	if err := repo.Save(ctx, &spec.Application{
		TenantID: "acme", ApplicationID: "app-2", Name: "Alpha", Kind: spec.ApplicationWeblink,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(ctx, "acme", "app-1")
	if err != nil || got == nil {
		t.Fatalf("FindByID: %v got=%v", err, got)
	}
	if got.Name != "Zebra" {
		t.Fatalf("unexpected name: %q", got.Name)
	}

	// 返り値の変更が内部状態へ漏れないこと (clone 分離)。
	got.Bindings[0].ClientID = "hijacked"
	got.CategoryIDs[0] = "hijacked"
	again, _ := repo.FindByID(ctx, "acme", "app-1")
	if again.Bindings[0].ClientID != "client-1" || again.CategoryIDs[0] != "cat-1" {
		t.Fatal("mutation leaked into stored application")
	}

	list, err := repo.ListByTenant(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "Alpha" || list[1].Name != "Zebra" {
		t.Fatalf("ListByTenant not sorted by name: %+v", list)
	}
	if other, _ := repo.ListByTenant(ctx, "unknown"); len(other) != 0 {
		t.Fatalf("expected empty list for unknown tenant, got %d", len(other))
	}

	if missing, _ := repo.FindByID(ctx, "acme", "nope"); missing != nil {
		t.Fatalf("expected nil for unknown app, got %+v", missing)
	}

	if err := repo.Delete(ctx, "acme", "app-1"); err != nil {
		t.Fatal(err)
	}
	if gone, _ := repo.FindByID(ctx, "acme", "app-1"); gone != nil {
		t.Fatal("app not deleted")
	}
}

func TestApplicationRepositoryFindByBinding(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()

	app := &spec.Application{
		TenantID: "acme", ApplicationID: "app-1", Name: "SP",
		Bindings: []spec.ProtocolBinding{
			{Type: spec.ProtocolBindingOIDC, ClientID: "cid"},
			{Type: spec.ProtocolBindingSAML, EntityID: "urn:sp"},
			{Type: spec.ProtocolBindingWsFed, Wtrealm: "urn:realm"},
		},
	}
	if err := repo.Save(ctx, app); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		bindingType spec.ProtocolBindingType
		key         string
		wantFound   bool
	}{
		{spec.ProtocolBindingOIDC, "cid", true},
		{spec.ProtocolBindingSAML, "urn:sp", true},
		{spec.ProtocolBindingWsFed, "urn:realm", true},
		{spec.ProtocolBindingOIDC, "other", false},
		{spec.ProtocolBindingOIDC, "", false},
	}
	for _, c := range cases {
		got, err := repo.FindByBinding(ctx, "acme", c.bindingType, c.key)
		if err != nil {
			t.Fatalf("FindByBinding(%s,%s): %v", c.bindingType, c.key, err)
		}
		if (got != nil) != c.wantFound {
			t.Fatalf("FindByBinding(%s,%s) found=%v want=%v", c.bindingType, c.key, got != nil, c.wantFound)
		}
	}
	// 別テナントでは一致しない。
	if got, _ := repo.FindByBinding(ctx, "other", spec.ProtocolBindingOIDC, "cid"); got != nil {
		t.Fatal("binding lookup ignored tenant boundary")
	}
}

func TestApplicationRepositoryRemoveCategory(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()
	if err := repo.Save(ctx, &spec.Application{
		TenantID: "acme", ApplicationID: "app-1", Name: "A",
		CategoryIDs: []string{"cat-1", "cat-2"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveCategory(ctx, "acme", "cat-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.FindByID(ctx, "acme", "app-1")
	if len(got.CategoryIDs) != 1 || got.CategoryIDs[0] != "cat-2" {
		t.Fatalf("category not removed: %+v", got.CategoryIDs)
	}
}

func TestApplicationIconStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewApplicationIconStore()

	icon := &spec.ApplicationIcon{
		TenantID: "acme", ApplicationID: "app-1", ObjectKey: "k1",
		ContentType: "image/png", SizeBytes: 3, Data: []byte{1, 2, 3},
	}
	if err := store.Save(ctx, icon); err != nil {
		t.Fatal(err)
	}

	got, err := store.Find(ctx, "acme", "app-1", "k1")
	if err != nil || got == nil {
		t.Fatalf("Find: %v got=%v", err, got)
	}
	// clone 分離。
	got.Data[0] = 9
	again, _ := store.Find(ctx, "acme", "app-1", "k1")
	if again.Data[0] != 1 {
		t.Fatal("icon data mutation leaked")
	}

	if missing, _ := store.Find(ctx, "acme", "app-1", "nope"); missing != nil {
		t.Fatalf("expected nil for unknown key, got %+v", missing)
	}

	if err := store.DeleteByApplication(ctx, "acme", "app-1"); err != nil {
		t.Fatal(err)
	}
	if gone, _ := store.Find(ctx, "acme", "app-1", "k1"); gone != nil {
		t.Fatal("icon not deleted")
	}
}

func TestSignInPolicyRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewSignInPolicyRepository()

	policy := &spec.AppSignInPolicy{
		TenantID: "acme", ApplicationID: "app-1",
		Rules: []spec.SignInRule{{RuleID: "r1", Name: "rule", Enabled: true}},
	}
	if err := repo.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "acme", "app-1")
	if err != nil || got == nil {
		t.Fatalf("Get: %v got=%v", err, got)
	}
	got.Rules[0].Name = "hijacked"
	again, _ := repo.Get(ctx, "acme", "app-1")
	if again.Rules[0].Name != "rule" {
		t.Fatal("rule mutation leaked")
	}

	list, err := repo.ListByTenant(ctx, "acme")
	if err != nil || len(list) != 1 {
		t.Fatalf("ListByTenant: %v len=%d", err, len(list))
	}

	if missing, _ := repo.Get(ctx, "acme", "nope"); missing != nil {
		t.Fatalf("expected nil, got %+v", missing)
	}

	if err := repo.Delete(ctx, "acme", "app-1"); err != nil {
		t.Fatal(err)
	}
	if gone, _ := repo.Get(ctx, "acme", "app-1"); gone != nil {
		t.Fatal("policy not deleted")
	}
}

func TestDefaultSignInPolicyRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewDefaultSignInPolicyRepository()

	policy := &spec.TenantDefaultSignInPolicy{
		TenantID: "acme",
		Rules:    []spec.SignInRule{{RuleID: "r1", Name: "floor", Enabled: true}},
	}
	if err := repo.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "acme")
	if err != nil || got == nil {
		t.Fatalf("Get: %v got=%v", err, got)
	}
	got.Rules[0].Name = "hijacked"
	again, _ := repo.Get(ctx, "acme")
	if again.Rules[0].Name != "floor" {
		t.Fatal("rule mutation leaked")
	}

	if missing, _ := repo.Get(ctx, "unknown"); missing != nil {
		t.Fatalf("expected nil, got %+v", missing)
	}
}

func TestApplicationAssignmentRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationAssignmentRepository()

	assignments := []*spec.ApplicationAssignment{
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectUser, SubjectID: "u2", Visibility: spec.AssignmentVisible},
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectUser, SubjectID: "u1", Visibility: spec.AssignmentVisible},
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectGroup, SubjectID: "g1", Visibility: spec.AssignmentHidden},
		{TenantID: "acme", ApplicationID: "app-2", SubjectType: spec.AssignmentSubjectUser, SubjectID: "u9", Visibility: spec.AssignmentVisible},
	}
	for _, a := range assignments {
		if err := repo.Save(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	byApp, err := repo.ListByApplication(ctx, "acme", "app-1")
	if err != nil {
		t.Fatal(err)
	}
	// group < user (アルファベット順), user 内では id 順。
	if len(byApp) != 3 {
		t.Fatalf("ListByApplication len=%d want 3", len(byApp))
	}
	if byApp[0].SubjectType != spec.AssignmentSubjectGroup ||
		byApp[1].SubjectID != "u1" || byApp[2].SubjectID != "u2" {
		t.Fatalf("ListByApplication order: %+v", byApp)
	}

	byTenant, _ := repo.ListByTenant(ctx, "acme")
	if len(byTenant) != 4 {
		t.Fatalf("ListByTenant len=%d want 4", len(byTenant))
	}

	bySubjects, err := repo.ListBySubjects(ctx, "acme", []appports.SubjectRef{
		{Type: spec.AssignmentSubjectUser, ID: "u1"},
		{Type: spec.AssignmentSubjectGroup, ID: "g1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bySubjects) != 2 {
		t.Fatalf("ListBySubjects len=%d want 2", len(bySubjects))
	}

	if err := repo.Delete(ctx, "acme", "app-1", spec.AssignmentSubjectUser, "u1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.ListByApplication(ctx, "acme", "app-1"); len(got) != 2 {
		t.Fatalf("after Delete len=%d want 2", len(got))
	}

	if err := repo.DeleteByApplication(ctx, "acme", "app-1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.ListByApplication(ctx, "acme", "app-1"); len(got) != 0 {
		t.Fatalf("after DeleteByApplication len=%d want 0", len(got))
	}
	// app-2 は残る。
	if got, _ := repo.ListByTenant(ctx, "acme"); len(got) != 1 {
		t.Fatalf("app-2 assignment should remain, got %d", len(got))
	}
}

func TestApplicationOrderingRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationOrderingRepository()

	ordering := &spec.ApplicationOrdering{
		UserID: "u1", ApplicationIDs: []string{"app-1", "app-2"},
	}
	if err := repo.Save(ctx, ordering); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "acme", "u1")
	if err != nil || got == nil {
		t.Fatalf("Get: %v got=%v", err, got)
	}
	got.ApplicationIDs[0] = "hijacked"
	again, _ := repo.Get(ctx, "acme", "u1")
	if again.ApplicationIDs[0] != "app-1" {
		t.Fatal("ordering mutation leaked")
	}

	if missing, _ := repo.Get(ctx, "acme", "unknown"); missing != nil {
		t.Fatalf("expected nil, got %+v", missing)
	}
}

func TestApplicationCategoryRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationCategoryRepository()

	categories := []*spec.ApplicationCategory{
		{TenantID: "acme", CategoryID: "c1", Name: "B", Position: 2},
		{TenantID: "acme", CategoryID: "c2", Name: "A", Position: 1},
		{TenantID: "acme", CategoryID: "c3", Name: "C", Position: 1},
	}
	for _, c := range categories {
		if err := repo.Save(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	list, err := repo.ListByTenant(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	// Position 昇順、同 Position は名前順 → c2(A,1), c3(C,1), c1(B,2)。
	if len(list) != 3 || list[0].CategoryID != "c2" || list[1].CategoryID != "c3" || list[2].CategoryID != "c1" {
		t.Fatalf("ListByTenant order: %+v", list)
	}

	got, err := repo.FindByID(ctx, "acme", "c1")
	if err != nil || got == nil || got.Name != "B" {
		t.Fatalf("FindByID: %v got=%v", err, got)
	}
	if missing, _ := repo.FindByID(ctx, "acme", "nope"); missing != nil {
		t.Fatalf("expected nil, got %+v", missing)
	}

	if err := repo.Delete(ctx, "acme", "c1"); err != nil {
		t.Fatal(err)
	}
	if gone, _ := repo.FindByID(ctx, "acme", "c1"); gone != nil {
		t.Fatal("category not deleted")
	}
}
