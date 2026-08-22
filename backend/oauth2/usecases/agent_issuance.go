package usecases

import (
	"context"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// AgentIssuanceDeps holds what ResolveIssuableAgent needs. Either repository
// being nil skips the corresponding gate, mirroring this codebase's other
// nil-skips-enforcement ports (lightweight wiring that never composes
// IdManagement).
type AgentIssuanceDeps struct {
	AgentRepo agentports.AgentRepository
	UserRepo  userports.UserRepository
	// Emit は区分を根拠とした拒否を監査へ流す。nil の配線では拒否そのものは変わらず、
	// 記録だけが落ちる。
	Emit func(spec.DomainEvent)
}

// ResolveIssuableAgent は clientID に束縛された Agent を、新しいトークンを発行して
// よい場合にだけ返す (REQ-OAUTH2-046)。束縛が無い client では (nil, nil) を返し、
// 所有者の解決も行わない。
//
// 発行を止める条件は 2 つある。Agent 自身が Active でない (kill-switch / 無効化) 場合と、
// 所有者がオフボードされている場合である。後者は Agent の status を書き換える一度きりの
// 状態遷移ではなく、発行のたびの評価として実装する。一度きりの書き込みは、それを行う
// 反応が届かなかった配備でガードごと失われるためで、この形なら所有者が復帰すれば発行も
// 自動的に再開する。所有者を解決できない場合 (ハード削除) も fail-closed で拒否する。
//
// 既に発行済みのトークンはこの評価では止まらない。それは SharedSignals の失効エポックが
// 受け持つ (IntrospectToken)。
func ResolveIssuableAgent(ctx context.Context, deps AgentIssuanceDeps, clientID string) (*agentdomain.Agent, error) {
	if deps.AgentRepo == nil {
		return nil, nil
	}
	tenantID := tenancy.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByClientID(ctx, tenantID, clientID)
	if err != nil || agent == nil {
		return nil, err
	}
	if !agent.IsActive() {
		return nil, NewOAuthError("invalid_client", "agent is disabled or killed")
	}
	active, err := AgentOwnerIsActive(ctx, deps.UserRepo, agent)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, NewOAuthError("invalid_client", "agent owner is offboarded")
	}
	return agent, nil
}

// AgentOwnerIsActive は agent の所有者がオフボードされていないかを返す。所有者を
// 解決できない場合 (ハード削除、別テナント) は false で、fail-closed に倒す。
// userRepo が nil の配線では判定を行わず true を返す。
//
// ResolveIssuableAgent が client_id から Agent を引く経路とは別に、既に Agent を
// 手元に持っている発行経路 (CIBA の auth_req_id 交換など) が、自身の段に合った
// OAuth error code を選べるように分けてある。
func AgentOwnerIsActive(ctx context.Context, userRepo userports.UserRepository, agent *agentdomain.Agent) (bool, error) {
	if userRepo == nil || agent == nil {
		return true, nil
	}
	owner, err := userRepo.FindBySub(ctx, agent.OwnerUserID)
	if err != nil {
		return false, err
	}
	return owner != nil && owner.TenantID == tenancy.TenantID(ctx) && owner.IsActive(), nil
}

// AgentRequiresHumanApproval は agent が人間の承認を記録する経路からしかトークンを
// 得られない区分かを返す (REQ-OAUTH2-050)。承認を記録するのは現在 CIBA だけで、
// client_credentials・トークン交換・ワークロード ID 連携による交換は true の Agent
// に対して拒否する。束縛された Agent がない client (agent == nil) は対象外。
//
// 判定は Supervised との一致ではなく Autonomous の否定で書く。区分に既知でない値が
// 入った場合 (データ破損、将来の区分追加) に、承認不要側へ倒れないためである。
//
// ResolveIssuableAgent には含めない。CIBA の StartApproval も同じ関数を通っており、
// そこは Supervised を通してよい唯一の経路だからである。
func AgentRequiresHumanApproval(agent *agentdomain.Agent) bool {
	return agent != nil && agent.Kind != idmdomain.AgentKindAutonomous
}

// ResolveIssuableAgentWithoutApproval は ResolveIssuableAgent の判定に区分のゲートを
// 重ね、人間の承認を記録しない発行経路 (client_credentials、トークン交換、ワークロード
// ID 連携による交換) が使ってよい Agent だけを返す。
func ResolveIssuableAgentWithoutApproval(
	ctx context.Context, deps AgentIssuanceDeps, clientID, grantType string, now time.Time,
) (*agentdomain.Agent, error) {
	agent, err := ResolveIssuableAgent(ctx, deps, clientID)
	if err != nil {
		return nil, err
	}
	if err := RejectAgentIssuanceWithoutApproval(ctx, deps.Emit, agent, clientID, grantType, now); err != nil {
		return nil, err
	}
	return agent, nil
}

// RejectAgentIssuanceWithoutApproval は既に手元にある Agent へ同じ区分のゲートを適用する。
// client_id から Agent を引かない経路 (ワークロード ID 連携の grant、subject_token が
// 名指す Agent) のためにある。
//
// 拒否は RFC 6749 の unauthorized_client (認証済みクライアントがその grant type を使う
// 権限を持たない) とする。この事実そのものであり、Agent 自身の状態や所有者を理由とする
// invalid_client (REQ-OAUTH2-046) と監査上も区別できる。
func RejectAgentIssuanceWithoutApproval(
	ctx context.Context, emitFn func(spec.DomainEvent), agent *agentdomain.Agent, clientID, grantType string, now time.Time,
) error {
	if !AgentRequiresHumanApproval(agent) {
		return nil
	}
	emit(emitFn, &domain.AgentApprovalRequired{
		At: now, TenantID: tenancy.TenantID(ctx), AgentID: agent.ID,
		ClientID: clientID, Kind: string(agent.Kind), GrantType: grantType,
	})
	return NewOAuthError("unauthorized_client", "supervised agent requires human approval")
}
