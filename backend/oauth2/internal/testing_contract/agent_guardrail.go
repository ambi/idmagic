package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// AgentGuardrailFixture は二つのテナントと、最初のテナントの Agent を渡す。
type AgentGuardrailFixture struct {
	Repo     ports.AgentGuardrailPolicyRepository
	Context  context.Context
	TenantID string
	Other    string
	AgentID  string
	Now      time.Time
}

// RunAgentGuardrail はバージョンの比較、nil と空の区別、テナント境界を保存先ごとに確かめる。
func RunAgentGuardrail(t *testing.T, f AgentGuardrailFixture) {
	t.Helper()
	ttl := 60
	first := &domain.AgentGuardrailPolicy{
		TenantID: f.TenantID, AgentID: f.AgentID, UpdatedAt: f.Now,
		AllowedResources: []string{}, MaxTokenTTLSeconds: &ttl,
	}
	if err := f.Repo.Save(f.Context, first, "stale"); !errors.Is(err, ports.ErrAgentGuardrailConflict) {
		t.Fatalf("save without a row and a version = %v", err)
	}
	if err := f.Repo.Save(f.Context, first, ""); err != nil {
		t.Fatal(err)
	}
	if err := f.Repo.Save(f.Context, first, ""); !errors.Is(err, ports.ErrAgentGuardrailConflict) {
		t.Fatalf("second create = %v", err)
	}
	found, err := f.Repo.Find(f.Context, f.TenantID, f.AgentID)
	if err != nil || found == nil {
		t.Fatalf("find = (%+v, %v)", found, err)
	}
	if found.AllowedResources == nil || len(found.AllowedResources) != 0 || found.AllowedAuthorizationDetailTypes != nil {
		t.Fatalf("nil and empty must round-trip: %+v", found.AgentGuardrailLimits)
	}
	if found.Version() != first.Version() {
		t.Fatalf("version changed across storage: %s != %s", found.Version(), first.Version())
	}

	second := &domain.AgentGuardrailPolicy{TenantID: f.TenantID, AgentID: f.AgentID, UpdatedAt: f.Now}
	if err := f.Repo.Save(f.Context, second, second.Version()); !errors.Is(err, ports.ErrAgentGuardrailConflict) {
		t.Fatalf("stale update = %v", err)
	}
	if err := f.Repo.Save(f.Context, second, first.Version()); err != nil {
		t.Fatal(err)
	}

	tenantDefault := &domain.AgentGuardrailPolicy{TenantID: f.TenantID, UpdatedAt: f.Now, RequireApproval: true}
	if err := f.Repo.Save(f.Context, tenantDefault, ""); err != nil {
		t.Fatal(err)
	}
	listed, err := f.Repo.ListEffective(f.Context, f.TenantID, []string{f.AgentID})
	if err != nil || len(listed) != 2 || listed[0].AgentID != "" || listed[1].AgentID != f.AgentID {
		t.Fatalf("effective = (%+v, %v)", listed, err)
	}
	if other, err := f.Repo.ListEffective(f.Context, f.Other, []string{f.AgentID}); err != nil || len(other) != 0 {
		t.Fatalf("tenant isolation = (%+v, %v)", other, err)
	}
	if err := f.Repo.Delete(f.Context, f.TenantID, ""); err != nil {
		t.Fatal(err)
	}
	if listed, err := f.Repo.ListEffective(f.Context, f.TenantID, []string{f.AgentID}); err != nil || len(listed) != 1 {
		t.Fatalf("after delete = (%+v, %v)", listed, err)
	}
}
