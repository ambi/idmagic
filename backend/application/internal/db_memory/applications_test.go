package db_memory

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/signinpolicy"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
)

func TestApplicationRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()

	app := &appdomain.Application{
		TenantID:    "acme",
		ID:          "app-1",
		Name:        "Zebra",
		Kind:        appdomain.ApplicationFederated,
		Status:      appdomain.ApplicationActive,
		Protocol:    &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "client-1"},
		CategoryIDs: []string{"cat-1", "cat-2"},
	}
	if err := repo.Save(ctx, app); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// 名前順ソートを確認するために 2 件目を追加する。
	if err := repo.Save(ctx, &appdomain.Application{
		TenantID: "acme", ID: "app-2", Name: "Alpha", Kind: appdomain.ApplicationWeblink,
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
	got.Protocol.ClientID = "hijacked"
	got.CategoryIDs[0] = "hijacked"
	again, _ := repo.FindByID(ctx, "acme", "app-1")
	if again.Protocol.ClientID != "client-1" || again.CategoryIDs[0] != "cat-1" {
		t.Fatal("mutation leaked into stored application")
	}

	list, err := repo.ListAll(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "Alpha" || list[1].Name != "Zebra" {
		t.Fatalf("ListAll not sorted by name: %+v", list)
	}
	if other, _ := repo.ListAll(ctx, "unknown"); len(other) != 0 {
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

func TestApplicationRepositoryFindByProtocol(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()

	app := &appdomain.Application{
		TenantID: "acme", ID: "app-1", Name: "SP",
		Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "cid"},
	}
	if err := repo.Save(ctx, app); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		bindingType appdomain.ApplicationProtocolType
		key         string
		wantFound   bool
	}{
		{appdomain.ApplicationProtocolOIDC, "cid", true},
		{appdomain.ApplicationProtocolSAML, "urn:sp", false},
		{appdomain.ApplicationProtocolWsFed, "urn:realm", false},
		{appdomain.ApplicationProtocolOIDC, "other", false},
		{appdomain.ApplicationProtocolOIDC, "", false},
	}
	for _, c := range cases {
		got, err := repo.FindByProtocol(ctx, "acme", c.bindingType, c.key)
		if err != nil {
			t.Fatalf("FindByProtocol(%s,%s): %v", c.bindingType, c.key, err)
		}
		if (got != nil) != c.wantFound {
			t.Fatalf("FindByProtocol(%s,%s) found=%v want=%v", c.bindingType, c.key, got != nil, c.wantFound)
		}
	}
	// 別テナントでは一致しない。
	if got, _ := repo.FindByProtocol(ctx, "other", appdomain.ApplicationProtocolOIDC, "cid"); got != nil {
		t.Fatal("binding lookup ignored tenant boundary")
	}
}

func TestApplicationRepositoryRemoveCategory(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()
	if err := repo.Save(ctx, &appdomain.Application{
		TenantID: "acme", ID: "app-1", Name: "A",
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

	icon := &appdomain.ApplicationIcon{
		TenantID: "acme", ApplicationID: "app-1", ID: "k1",
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
	repo := signinpolicy.NewMemoryModule().ApplicationRepo

	policy := &policydomain.AppSignInPolicy{
		TenantID: "acme", ApplicationID: "app-1",
		Rules: []policydomain.SignInRule{{RuleID: "r1", Name: "rule", Enabled: true}},
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

	list, err := repo.ListAll(ctx, "acme")
	if err != nil || len(list) != 1 {
		t.Fatalf("ListAll: %v len=%d", err, len(list))
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
	repo := signinpolicy.NewMemoryModule().DefaultRepo

	policy := &policydomain.TenantDefaultSignInPolicy{
		TenantID: "acme",
		Rules:    []policydomain.SignInRule{{RuleID: "r1", Name: "floor", Enabled: true}},
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

	assignments := []*appdomain.ApplicationAssignment{
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "u2", Visibility: appdomain.AssignmentVisible},
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "u1", Visibility: appdomain.AssignmentVisible},
		{TenantID: "acme", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectGroup, SubjectID: "g1", Visibility: appdomain.AssignmentHidden},
		{TenantID: "acme", ApplicationID: "app-2", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "u9", Visibility: appdomain.AssignmentVisible},
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
	if byApp[0].SubjectType != appdomain.AssignmentSubjectGroup ||
		byApp[1].SubjectID != "u1" || byApp[2].SubjectID != "u2" {
		t.Fatalf("ListByApplication order: %+v", byApp)
	}

	byTenant, _ := repo.ListAll(ctx, "acme")
	if len(byTenant) != 4 {
		t.Fatalf("ListAll len=%d want 4", len(byTenant))
	}

	bySubjects, err := repo.ListBySubjects(ctx, "acme", []appports.SubjectRef{
		{Type: appdomain.AssignmentSubjectUser, ID: "u1"},
		{Type: appdomain.AssignmentSubjectGroup, ID: "g1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bySubjects) != 2 {
		t.Fatalf("ListBySubjects len=%d want 2", len(bySubjects))
	}

	if err := repo.Delete(ctx, "acme", "app-1", appdomain.AssignmentSubjectUser, "u1"); err != nil {
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
	if got, _ := repo.ListAll(ctx, "acme"); len(got) != 1 {
		t.Fatalf("app-2 assignment should remain, got %d", len(got))
	}
}

func TestApplicationOrderingRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationOrderingRepository()

	ordering := &appdomain.ApplicationOrdering{
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

	categories := []*appdomain.ApplicationCategory{
		{TenantID: "acme", ID: "c1", Name: "B", Position: 2},
		{TenantID: "acme", ID: "c2", Name: "A", Position: 1},
		{TenantID: "acme", ID: "c3", Name: "C", Position: 1},
	}
	for _, c := range categories {
		if err := repo.Save(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	list, err := repo.ListAll(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	// Position 昇順、同 Position は名前順 → c2(A,1), c3(C,1), c1(B,2)。
	if len(list) != 3 || list[0].ID != "c2" || list[1].ID != "c3" || list[2].ID != "c1" {
		t.Fatalf("ListAll order: %+v", list)
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

func TestApplicationAssignmentRepositoryListPage(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationAssignmentRepository()
	for _, sid := range []string{"u3", "u1", "u2", "u4", "u5"} {
		if err := repo.Save(ctx, &appdomain.ApplicationAssignment{
			TenantID: "acme", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: sid,
			Visibility: appdomain.AssignmentVisible,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: "acme", ApplicationID: "app-2", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "other-app",
		Visibility: appdomain.AssignmentVisible,
	}); err != nil {
		t.Fatal(err)
	}

	first, err := repo.ListPageByApplication(ctx, "acme", "app-1", "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].SubjectID != "u1" || first[1].SubjectID != "u2" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	last := first[len(first)-1]
	next, err := repo.ListPageByApplication(ctx, "acme", "app-1", string(last.SubjectType), last.SubjectID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].SubjectID != "u3" || next[1].SubjectID != "u4" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}
	all, err := repo.ListPageByApplication(ctx, "acme", "app-1", "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}

func TestApplicationRepositoryListPage(t *testing.T) {
	ctx := context.Background()
	repo := NewApplicationRepository()
	for _, name := range []string{"Charlie", "Alpha", "Bravo", "Delta", "Echo"} {
		if err := repo.Save(ctx, &appdomain.Application{
			TenantID: "acme", ID: name + "-id", Name: name, Kind: appdomain.ApplicationWeblink,
		}); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}
	if err := repo.Save(ctx, &appdomain.Application{
		TenantID: "other-tenant", ID: "zulu-id", Name: "Zulu", Kind: appdomain.ApplicationWeblink,
	}); err != nil {
		t.Fatal(err)
	}

	first, err := repo.ListPage(ctx, "acme", "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Name != "Alpha" || first[1].Name != "Bravo" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, "acme", last.Name, last.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].Name != "Charlie" || next[1].Name != "Delta" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}
	all, err := repo.ListPage(ctx, "acme", "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
	total, err := repo.Count(ctx, "acme")
	if err != nil || total != 5 {
		t.Fatalf("Count = %d, err=%v", total, err)
	}
	lastPage, err := repo.ListPageBefore(ctx, "acme", "", "", 2)
	if err != nil || len(lastPage) != 2 || lastPage[0].Name != "Delta" || lastPage[1].Name != "Echo" {
		t.Fatalf("last page = %+v, err=%v", lastPage, err)
	}
}
