package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
)

// activeAgents は、どの識別子にも有効な Agent を返す。
type activeAgents struct{ agentports.AgentRepository }

func (activeAgents) FindByID(_ context.Context, tenantID, id string) (*agentdomain.Agent, error) {
	return &agentdomain.Agent{ID: id, TenantID: tenantID, Status: idmdomain.AgentStatusActive}, nil
}

// failFirstDelegationSave は一度目の代行の保存だけを失敗させる。
type failFirstDelegationSave struct {
	consentports.ConsentRepository
	failed bool
}

func (r *failFirstDelegationSave) SaveAgentDelegation(ctx context.Context, tenantID string, c *domain.Consent) error {
	if !r.failed {
		r.failed = true
		return errConsentSave
	}
	return r.ConsentRepository.SaveAgentDelegation(ctx, tenantID, c)
}

//spec:covers REQ-TENANCY-050: 保存に失敗した Agent への代行の付与は consents の使用量を残さず、保存できた付与は一つ残す。
func TestGrantAgentDelegationReleasesQuotaWhenSaveFails(t *testing.T) {
	const tenantID = quotaTenantID
	ctx := tenantContext(tenantID)
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	deps := AgentDelegationDeps{
		ConsentRepo: &failFirstDelegationSave{ConsentRepository: oauth2memory.NewConsentRepository()},
		QuotaRepo:   quotaRepo,
		Agents:      activeAgents{},
	}
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	in := AgentDelegationGrant{Sub: "alice", AgentID: "agent-1", Scopes: []string{"documents:read"}}

	if _, err := GrantAgentDelegation(ctx, deps, in, now); !errors.Is(err, errConsentSave) {
		t.Fatalf("GrantAgentDelegation with a failing save = %v, want %v", err, errConsentSave)
	}
	if got := consentsUsage(t, quotaRepo); got != 0 {
		t.Fatalf("consents usage after a failed save = %d, want 0", got)
	}
	if _, err := GrantAgentDelegation(ctx, deps, in, now); err != nil {
		t.Fatalf("GrantAgentDelegation after the failed save: %v", err)
	}
	if got := consentsUsage(t, quotaRepo); got != 1 {
		t.Fatalf("consents usage after a saved delegation = %d, want 1", got)
	}
}
