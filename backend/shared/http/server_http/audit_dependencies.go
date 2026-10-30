package server_http

import (
	"context"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// tokenIssuanceEventTypes は、Agent の棚卸しが最終のトークン発行として数える監査の種別である
// (REQ-IDMANAGEMENT-094)。
var tokenIssuanceEventTypes = []string{"AccessTokenIssued", "TokenExchanged"}

// auditAgentActivity は IdManagement の AgentActivityReader を、Audit の検索属性 agent.id の
// 読み取りで満たす。未配線の Audit は nil を返し、棚卸しは材料を読めないものとして失敗する。
type auditAgentActivity struct {
	events auditports.AuditEventRepository
}

func newAuditAgentActivity(events auditports.AuditEventRepository) agentports.AgentActivityReader {
	if events == nil {
		return nil
	}
	return auditAgentActivity{events: events}
}

func (a auditAgentActivity) LastTokenIssuedAt(ctx context.Context, tenantID string, agentIDs []string) (map[string]time.Time, error) {
	return a.events.LatestByAgent(ctx, tenantID, agentIDs, tokenIssuanceEventTypes)
}

// auditUsernameLookup は Audit の要求する ID と存在の有無へ User を変換する。
// 未配線の照会は、従来と同じく username に一致する記録を返さない。
func auditUsernameLookup(users userports.UserRepository) auditports.UsernameLookup {
	if users == nil {
		return nil
	}
	return func(ctx context.Context, tenantID, username string) (string, bool, error) {
		user, err := users.FindByUsername(ctx, tenantID, username)
		if err != nil || user == nil {
			return "", false, err
		}
		return user.ID, true, nil
	}
}
