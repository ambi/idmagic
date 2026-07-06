package usecases_test

import (
	"errors"
	"testing"
	"time"

	appusecases "github.com/ambi/idmagic/internal/application/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func newCategoryDeps() (appusecases.CategoryDeps, appusecases.ApplicationDeps) {
	apps := memory.NewApplicationRepository()
	assignments := memory.NewApplicationAssignmentRepository()
	categories := memory.NewApplicationCategoryRepository()
	return appusecases.CategoryDeps{Repo: categories, AppRepo: apps},
		appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: assignments}
}

func TestCreateCategoryAssignsTrailingPosition(t *testing.T) {
	ctx := tenantContext()
	deps, _ := newCategoryDeps()

	first, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Work"})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.Position != 0 {
		t.Fatalf("first position want 0, got %d", first.Position)
	}
	second, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Personal"})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.Position != 1 {
		t.Fatalf("second position want 1, got %d", second.Position)
	}

	// 指定ポジション
	pos := 10
	third, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Others", Position: &pos})
	if err != nil {
		t.Fatalf("create third: %v", err)
	}
	if third.Position != 10 {
		t.Fatalf("third position want 10, got %d", third.Position)
	}
}

func TestCreateCategoryRejectsBlankName(t *testing.T) {
	ctx := tenantContext()
	deps, _ := newCategoryDeps()
	if _, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "  "}); !errors.Is(err, appusecases.ErrCategoryNameRequired) {
		t.Fatalf("expected ErrCategoryNameRequired, got %v", err)
	}
}

func TestListCategories(t *testing.T) {
	ctx := tenantContext()
	deps, _ := newCategoryDeps()

	_, _ = appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Work"})
	_, _ = appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Personal"})

	list, err := appusecases.ListCategories(ctx, deps)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len want 2, got %d", len(list))
	}
}

func TestUpdateCategory(t *testing.T) {
	ctx := tenantContext()
	deps, _ := newCategoryDeps()

	cat, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Work"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 正常系更新
	newName := "Work Updated"
	pos := 5
	updated, err := appusecases.UpdateCategory(ctx, deps, appusecases.UpdateCategoryInput{
		ActorUserID: "admin", CategoryID: cat.CategoryID, Name: &newName, Position: &pos,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Work Updated" || updated.Position != 5 {
		t.Fatalf("update failed: %+v", updated)
	}

	// 名前の変更なし
	updated2, err := appusecases.UpdateCategory(ctx, deps, appusecases.UpdateCategoryInput{
		ActorUserID: "admin", CategoryID: cat.CategoryID, Name: nil, Position: nil,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated2.Name != "Work Updated" {
		t.Fatalf("name should not change: %+v", updated2)
	}

	// 存在しないカテゴリID
	if _, err := appusecases.UpdateCategory(ctx, deps, appusecases.UpdateCategoryInput{
		ActorUserID: "admin", CategoryID: "ghost", Name: &newName,
	}); !errors.Is(err, appusecases.ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound, got %v", err)
	}

	// カテゴリ名を空にする
	emptyName := "  "
	if _, err := appusecases.UpdateCategory(ctx, deps, appusecases.UpdateCategoryInput{
		ActorUserID: "admin", CategoryID: cat.CategoryID, Name: &emptyName,
	}); !errors.Is(err, appusecases.ErrCategoryNameRequired) {
		t.Fatalf("expected ErrCategoryNameRequired, got %v", err)
	}
}

func TestSetApplicationCategoriesValidatesAndDedups(t *testing.T) {
	ctx := tenantContext()
	deps, appDeps := newCategoryDeps()

	work, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Work"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	app, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Payroll", Kind: spec.ApplicationFederated,
	})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// 重複を含めても 1 件に正規化される。
	updated, err := appusecases.SetApplicationCategories(ctx, deps, appusecases.SetApplicationCategoriesInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, CategoryIDs: []string{work.CategoryID, work.CategoryID},
	})
	if err != nil {
		t.Fatalf("set categories: %v", err)
	}
	if len(updated.CategoryIDs) != 1 || updated.CategoryIDs[0] != work.CategoryID {
		t.Fatalf("category_ids should dedup to one: %v", updated.CategoryIDs)
	}

	// 未知のカテゴリは拒否する。
	if _, err := appusecases.SetApplicationCategories(ctx, deps, appusecases.SetApplicationCategoriesInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, CategoryIDs: []string{"nope"},
	}); !errors.Is(err, appusecases.ErrUnknownCategory) {
		t.Fatalf("expected ErrUnknownCategory, got %v", err)
	}

	// 存在しないアプリ
	if _, err := appusecases.SetApplicationCategories(ctx, deps, appusecases.SetApplicationCategoriesInput{
		ActorUserID: "admin", ApplicationID: "ghost", CategoryIDs: []string{work.CategoryID},
	}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

func TestDeleteCategoryScrubsFromApplications(t *testing.T) {
	ctx := tenantContext()
	deps, appDeps := newCategoryDeps()

	work, err := appusecases.CreateCategory(ctx, deps, appusecases.CreateCategoryInput{ActorUserID: "admin", Name: "Work"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	app, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Payroll", Kind: spec.ApplicationFederated,
	})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := appusecases.SetApplicationCategories(ctx, deps, appusecases.SetApplicationCategoriesInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, CategoryIDs: []string{work.CategoryID},
	}); err != nil {
		t.Fatalf("set categories: %v", err)
	}

	if err := appusecases.DeleteCategory(ctx, deps, "admin", work.CategoryID, time.Time{}); err != nil {
		t.Fatalf("delete category: %v", err)
	}
	got, err := appDeps.Repo.FindByID(ctx, "acme", app.ApplicationID)
	if err != nil {
		t.Fatalf("find app: %v", err)
	}
	if len(got.CategoryIDs) != 0 {
		t.Fatalf("deleted category must be scrubbed from app, got %v", got.CategoryIDs)
	}

	// 存在しないカテゴリ
	if err := appusecases.DeleteCategory(ctx, deps, "admin", "ghost", time.Time{}); !errors.Is(err, appusecases.ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound, got %v", err)
	}
}
