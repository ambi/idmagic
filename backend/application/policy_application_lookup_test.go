package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
)

type failedPolicyApplicationLookup struct {
	ports.ApplicationRepository
	err error
}

func (r failedPolicyApplicationLookup) FindByID(context.Context, string, string) (*domain.Application, error) {
	return nil, r.err
}

//spec:covers REQ-APPLICATION-009: ポリシー管理へ同じテナントのアプリの実在だけを供給する。
func TestPolicyApplicationLookupKeepsTheTenantAndStoreFailure(t *testing.T) {
	module := NewMemoryModule()
	if err := module.Repo.Save(t.Context(), &domain.Application{
		TenantID: "tenant-a", ID: "app-1", Name: "Payroll",
		Kind: domain.ApplicationWeblink, Status: domain.ApplicationActive,
	}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		tenant string
		id     string
		want   bool
	}{
		{tenant: "tenant-a", id: "app-1", want: true},
		{tenant: "tenant-b", id: "app-1"},
		{tenant: "tenant-a", id: "missing"},
	} {
		got, err := module.ApplicationExists(t.Context(), test.tenant, test.id)
		if err != nil || got != test.want {
			t.Fatalf("ApplicationExists(%q, %q) = %v, %v、期待値 %v, nil", test.tenant, test.id, got, err, test.want)
		}
	}
	if exists, err := (Module{}).ApplicationExists(t.Context(), "tenant-a", "app-1"); exists || err != nil {
		t.Fatalf("保存先のない実在確認 = %v, %v、期待値 false, nil", exists, err)
	}
	lookupFailure := errors.New("lookup failed")
	failed := Module{Repo: failedPolicyApplicationLookup{err: lookupFailure}}
	if exists, err := failed.ApplicationExists(t.Context(), "tenant-a", "app-1"); exists || !errors.Is(err, lookupFailure) {
		t.Fatalf("保存先の障害を伴う実在確認 = %v, %v、期待値 false, %v", exists, err, lookupFailure)
	}
}
