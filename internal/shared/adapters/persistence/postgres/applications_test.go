package postgres

import (
	"bytes"
	"context"
	"testing"

	appports "github.com/ambi/idmagic/internal/application/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestApplicationRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	client := seedClient(t, db, tenant.ID)
	repo := &ApplicationRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	categoryID := uniqueID("cat")
	app := &spec.Application{
		TenantID:      tenant.ID,
		ApplicationID: newUUID(t),
		Name:          "Portal App",
		Kind:          spec.ApplicationFederated,
		Status:        spec.ApplicationActive,
		LaunchURL:     "https://app.example/launch",
		Bindings: []spec.ProtocolBinding{
			{Type: spec.ProtocolBindingOIDC, ClientID: client.ClientID},
		},
		CategoryIDs: []string{categoryID},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(ctx, app); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID, app.ApplicationID)
	if err != nil || got == nil {
		t.Fatalf("find by id: %v %+v", err, got)
	}
	if got.Name != "Portal App" || got.Kind != spec.ApplicationFederated {
		t.Fatalf("unexpected application: %+v", got)
	}
	if len(got.Bindings) != 1 || got.Bindings[0].ClientID != client.ClientID {
		t.Fatalf("bindings not round-tripped: %+v", got.Bindings)
	}
	if len(got.CategoryIDs) != 1 || got.CategoryIDs[0] != categoryID {
		t.Fatalf("category ids not round-tripped: %+v", got.CategoryIDs)
	}

	byBinding, err := repo.FindByBinding(ctx, tenant.ID, spec.ProtocolBindingOIDC, client.ClientID)
	if err != nil || byBinding == nil || byBinding.ApplicationID != app.ApplicationID {
		t.Fatalf("find by binding: %v %+v", err, byBinding)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.RemoveCategory(ctx, tenant.ID, categoryID); err != nil {
		t.Fatalf("remove category: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID, app.ApplicationID)
	if err != nil || got == nil || len(got.CategoryIDs) != 0 {
		t.Fatalf("category not removed: %v %+v", err, got)
	}

	if err := repo.Delete(ctx, tenant.ID, app.ApplicationID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID, app.ApplicationID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

func TestSignInPolicyRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	app := seedApplication(t, db, tenant.ID)
	repo := &SignInPolicyRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	policy := &spec.AppSignInPolicy{
		TenantID:      tenant.ID,
		ApplicationID: app.ApplicationID,
		Rules: []spec.SignInRule{
			{
				RuleID:        "rule-1",
				Name:          "require mfa",
				Enabled:       true,
				RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, policy); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, tenant.ID, app.ApplicationID)
	if err != nil || got == nil {
		t.Fatalf("get: %v %+v", err, got)
	}
	if len(got.Rules) != 1 || got.Rules[0].RequiredAuthn.Strength != spec.RequiredAuthnMfa {
		t.Fatalf("rules not round-tripped: %+v", got.Rules)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, app.ApplicationID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.Get(ctx, tenant.ID, app.ApplicationID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

func TestDefaultSignInPolicyRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &DefaultSignInPolicyRepository{Pool: db}
	ctx := context.Background()

	if got, err := repo.Get(ctx, tenant.ID); err != nil || got != nil {
		t.Fatalf("expected no policy initially: %v %+v", err, got)
	}

	now := testClock()
	policy := &spec.TenantDefaultSignInPolicy{
		TenantID: tenant.ID,
		Rules: []spec.SignInRule{
			{
				RuleID:        "floor",
				Name:          "password floor",
				Enabled:       true,
				RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, policy); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, tenant.ID)
	if err != nil || got == nil || len(got.Rules) != 1 {
		t.Fatalf("get: %v %+v", err, got)
	}
	if got.Rules[0].RequiredAuthn.Strength != spec.RequiredAuthnPassword {
		t.Fatalf("rule not round-tripped: %+v", got.Rules[0])
	}
}

func TestApplicationIconStoreRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	app := seedApplication(t, db, tenant.ID)
	store := &ApplicationIconStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	icon := &spec.ApplicationIcon{
		TenantID:      tenant.ID,
		ApplicationID: app.ApplicationID,
		ObjectKey:     "icon-1",
		ContentType:   "image/png",
		SizeBytes:     4,
		Data:          []byte{0x1, 0x2, 0x3, 0x4},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.Save(ctx, icon); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Find(ctx, tenant.ID, app.ApplicationID, "icon-1")
	if err != nil || got == nil {
		t.Fatalf("find: %v %+v", err, got)
	}
	if got.ContentType != "image/png" || !bytes.Equal(got.Data, icon.Data) {
		t.Fatalf("icon not round-tripped: %+v", got)
	}

	if err := store.DeleteByApplication(ctx, tenant.ID, app.ApplicationID); err != nil {
		t.Fatalf("delete by application: %v", err)
	}
	got, err = store.Find(ctx, tenant.ID, app.ApplicationID, "icon-1")
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

func TestApplicationAssignmentRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	group := seedGroup(t, db, tenant.ID)
	app := seedApplication(t, db, tenant.ID)
	repo := &ApplicationAssignmentRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	userAssignment := &spec.ApplicationAssignment{
		TenantID:      tenant.ID,
		ApplicationID: app.ApplicationID,
		SubjectType:   spec.AssignmentSubjectUser,
		SubjectID:     user.ID,
		Visibility:    spec.AssignmentVisible,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	groupAssignment := &spec.ApplicationAssignment{
		TenantID:      tenant.ID,
		ApplicationID: app.ApplicationID,
		SubjectType:   spec.AssignmentSubjectGroup,
		SubjectID:     group.ID,
		Visibility:    spec.AssignmentHidden,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.Save(ctx, userAssignment); err != nil {
		t.Fatalf("save user assignment: %v", err)
	}
	if err := repo.Save(ctx, groupAssignment); err != nil {
		t.Fatalf("save group assignment: %v", err)
	}

	byTenant, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(byTenant) != 2 {
		t.Fatalf("list by tenant: %v len=%d", err, len(byTenant))
	}

	byApp, err := repo.ListByApplication(ctx, tenant.ID, app.ApplicationID)
	if err != nil || len(byApp) != 2 {
		t.Fatalf("list by application: %v len=%d", err, len(byApp))
	}

	bySubjects, err := repo.ListBySubjects(ctx, tenant.ID, []appports.SubjectRef{
		{Type: spec.AssignmentSubjectUser, ID: user.ID},
	})
	if err != nil || len(bySubjects) != 1 || bySubjects[0].SubjectID != user.ID {
		t.Fatalf("list by subjects: %v %+v", err, bySubjects)
	}

	if err := repo.Delete(ctx, tenant.ID, app.ApplicationID, spec.AssignmentSubjectUser, user.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	byApp, err = repo.ListByApplication(ctx, tenant.ID, app.ApplicationID)
	if err != nil || len(byApp) != 1 {
		t.Fatalf("after delete: %v len=%d", err, len(byApp))
	}

	if err := repo.DeleteByApplication(ctx, tenant.ID, app.ApplicationID); err != nil {
		t.Fatalf("delete by application: %v", err)
	}
	byApp, err = repo.ListByApplication(ctx, tenant.ID, app.ApplicationID)
	if err != nil || len(byApp) != 0 {
		t.Fatalf("expected empty: %v len=%d", err, len(byApp))
	}
}

func TestApplicationOrderingRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &ApplicationOrderingRepository{Pool: db}
	ctx := context.Background()

	if got, err := repo.Get(ctx, tenant.ID, user.ID); err != nil || got != nil {
		t.Fatalf("expected no ordering initially: %v %+v", err, got)
	}

	now := testClock()
	ordering := &spec.ApplicationOrdering{
		UserID:         user.ID,
		ApplicationIDs: []string{"app-b", "app-a", "app-c"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.Save(ctx, ordering); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, tenant.ID, user.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v %+v", err, got)
	}
	// text[] は順序を保つため、保存した順序どおりに読み戻せること。
	if len(got.ApplicationIDs) != 3 || got.ApplicationIDs[0] != "app-b" || got.ApplicationIDs[2] != "app-c" {
		t.Fatalf("ordering not round-tripped: %+v", got.ApplicationIDs)
	}

	// upsert: 並び順を差し替えても user 1 行に収束する。
	ordering.ApplicationIDs = []string{"app-a"}
	if err := repo.Save(ctx, ordering); err != nil {
		t.Fatalf("resave: %v", err)
	}
	got, err = repo.Get(ctx, tenant.ID, user.ID)
	if err != nil || got == nil || len(got.ApplicationIDs) != 1 || got.ApplicationIDs[0] != "app-a" {
		t.Fatalf("upsert not applied: %v %+v", err, got)
	}
}

func TestApplicationCategoryRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &ApplicationCategoryRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	category := &spec.ApplicationCategory{
		TenantID:   tenant.ID,
		CategoryID: newUUID(t),
		Name:       "Productivity",
		Position:   2,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.Save(ctx, category); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID, category.CategoryID)
	if err != nil || got == nil || got.Name != "Productivity" || got.Position != 2 {
		t.Fatalf("find by id: %v %+v", err, got)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, category.CategoryID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID, category.CategoryID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}
