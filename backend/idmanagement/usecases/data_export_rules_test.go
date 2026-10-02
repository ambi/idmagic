package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
)

func exportCount(t *testing.T, deps idmusecases.DataExportDeps, target idmdomain.DataExportTargetKind) int {
	t.Helper()
	views, err := idmusecases.ListDataExports(exportTestCtx(), deps, idmusecases.ExportScope{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	return len(views)
}

// 拒否した開始が残すものは無い。ジョブも監査の記録も作らないことまで読む。
//
//spec:covers EX-IDMANAGEMENT-039-01, EX-IDMANAGEMENT-039-02: 重複した列と許可していない絞り込みの開始を拒否し、ジョブを作らず DataExportRequested を発行しないこと。
func TestStartDataExportRejectsDuplicateColumnsAndUnknownFiltersWithoutAJob(t *testing.T) {
	deps, rec := seededExportDeps(t)
	ctx := exportTestCtx()
	now := time.Now().UTC()
	if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email", "email"}, nil, now); !errors.Is(err, idmdomain.ErrInvalidExportColumns) {
		t.Fatalf("重複した列: err=%v, want ErrInvalidExportColumns", err)
	}
	if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email"}, map[string]string{"email": "a@example.test"}, now); !errors.Is(err, idmusecases.ErrInvalidExportFilter) {
		t.Fatalf("User の未許可の絞り込み: err=%v, want ErrInvalidExportFilter", err)
	}
	if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "group", []string{"name"}, map[string]string{"status": "active"}, now); !errors.Is(err, idmusecases.ErrInvalidExportFilter) {
		t.Fatalf("Group の絞り込み: err=%v, want ErrInvalidExportFilter", err)
	}
	if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email"}, map[string]string{"status": "sleeping"}, now); !errors.Is(err, idmusecases.ErrInvalidExportFilter) {
		t.Fatalf("未知の状態: err=%v, want ErrInvalidExportFilter", err)
	}
	if count := exportCount(t, deps, idmdomain.ExportTargetUser) + exportCount(t, deps, idmdomain.ExportTargetGroup); count != 0 {
		t.Fatalf("拒否した開始がエクスポートを %d 件作った", count)
	}
	if types := rec.types(); len(types) != 0 {
		t.Fatalf("拒否した開始がイベントを発行した: %v", types)
	}
}

//spec:covers EX-IDMANAGEMENT-039-03: 状態の絞り込みを前後の空白と大文字と小文字を区別せずに受け付け、その状態の User だけを書き出すこと。
func TestStartDataExportAcceptsAStatusFilterRegardlessOfCaseAndSpaces(t *testing.T) {
	deps, _ := seededExportDeps(t)
	view, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "user", []string{"preferred_username"}, map[string]string{"status": " Active "}, time.Now().UTC())
	if err != nil {
		t.Fatalf("err=%v, want accepted", err)
	}
	job, err := deps.JobRepo.Get(context.Background(), view.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := idmusecases.DataExportHandler(deps)(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	var result idmusecases.DataExportResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	// 種の User は Active の alice と Disabled の 1 人である。
	if result.TotalRows != 1 {
		t.Fatalf("total_rows=%d, want only the Active user", result.TotalRows)
	}
}

//spec:covers EX-IDMANAGEMENT-040-01: エクスポートの一覧を作成の新しい順に返すこと。
func TestListDataExportsReturnsNewestFirst(t *testing.T) {
	deps, _ := seededExportDeps(t)
	ctx := exportTestCtx()
	base := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)
	first, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email"}, nil, base)
	if err != nil {
		t.Fatal(err)
	}
	second, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email"}, nil, base.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	views, err := idmusecases.ListDataExports(ctx, deps, idmusecases.ExportScope{Target: idmdomain.ExportTargetUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].ID != second.ID || views[1].ID != first.ID {
		t.Fatalf("order=%v, want [%s %s]", exportIDs(views), second.ID, first.ID)
	}
}

// 種類で絞る前に直近 200 件で切るので、別の種類が窓を占めると古い方が見えなくなる。
// 要判断として残した現在の挙動を、そのまま固定する。
//
//spec:covers EX-IDMANAGEMENT-040-02: テナントの直近 200 件を別の種類のエクスポートが占めると、要求した種類の古いエクスポートが一覧に現れないこと。
func TestListDataExportsReadsOnlyTheNewestTwoHundredBeforeFilteringByTarget(t *testing.T) {
	deps, _ := seededExportDeps(t)
	ctx := exportTestCtx()
	base := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)
	if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"email"}, nil, base); err != nil {
		t.Fatal(err)
	}
	for i := range 200 {
		if _, err := idmusecases.StartDataExport(ctx, deps, "admin", "group", []string{"name"}, nil, base.Add(time.Duration(i+1)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if count := exportCount(t, deps, idmdomain.ExportTargetUser); count != 0 {
		t.Fatalf("User のエクスポートが %d 件見えた, want 0", count)
	}
	if count := exportCount(t, deps, idmdomain.ExportTargetGroup); count != 200 {
		t.Fatalf("Group のエクスポートが %d 件, want 200", count)
	}
}

func exportIDs(views []*idmusecases.DataExportView) []string {
	out := make([]string, len(views))
	for i, view := range views {
		out[i] = view.ID
	}
	return out
}
