package usecases

// 管理者向け Agent ライフサイクル操作と OAuth2Client 資格情報束縛。
// SCL IdManagement モジュールが所有する admin インターフェース群:
// ListAgents / GetAgent / RegisterAgent / UpdateAgent / DisableAgent /
// EnableAgent / KillAgent / DeleteAgent / BindAgentCredential /
// UnbindAgentCredential。
//
// すべての操作は tenantports.TenantID(ctx) のテナント境界に閉じ、cross-tenant な参照・
// 束縛は reject する。Agent は自身の資格情報を持たず、AgentCredentialBinding で既存
// OAuth2Client に束縛してトークンを得る。Status は Active / Disabled / Killed の三状態で、
// Killed は一方向終端 (緊急停止) であり復帰できない。

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	ErrAgentNotFound       = errors.New("agent not found")
	ErrAgentNameConflict   = errors.New("agent name already exists")
	ErrAgentNameEmpty      = errors.New("agent name is required")
	ErrAgentOwnerRequired  = errors.New("agent owner is required")
	ErrAgentOwnerNotFound  = errors.New("agent owner not found")
	ErrAgentKindRequired   = errors.New("agent kind is required")
	ErrAgentKindInvalid    = errors.New("agent kind is not a known value")
	ErrAgentKilled         = errors.New("agent is killed and cannot be modified")
	ErrAgentClientNotFound = errors.New("client not found")
	ErrAgentClientBound    = errors.New("client is already bound to another agent")
	// ErrAgentClientAuthMethodNotAllowed は束縛先のクライアントが共有シークレットか none で
	// 認証する。Agent の資格情報は鍵の所有を証明する方式に限る (REQ-IDMANAGEMENT-074)。
	ErrAgentClientAuthMethodNotAllowed = errors.New("client authentication method cannot be bound to an agent")
)

type AdminAgentDeps struct {
	AgentRepo  agentports.AgentRepository
	ClientRepo agentports.ClientLookup
	UserRepo   userports.UserRepository
	// Activity は棚卸しが最終のトークン発行を読む先である。nil の配線では棚卸しを失敗させる。
	Activity agentports.AgentActivityReader
	Emit     func(spec.DomainEvent) error
	// QuotaRepo enforces the tenant's Hard Quota on agents (wi-160).
	// nil skips enforcement (wiring gaps in tests/tools); production
	// bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

// AgentView は一覧・詳細で Agent と束縛済み client id をまとめて返す。
type AgentView struct {
	Agent     *agentdomain.Agent
	ClientIDs []string
}

// ListAgents returns up to limit+1 agents after the given keyset (wi-159)
// — callers pass limit+1 to detect whether a next page exists, then
// trim to limit before responding.
func ListAgents(ctx context.Context, deps AdminAgentDeps, afterName, afterID string, limit int) ([]AgentView, error) {
	tenantID := tenantports.TenantID(ctx)
	agents, err := deps.AgentRepo.ListPage(ctx, tenantID, afterName, afterID, limit)
	return agentViews(ctx, deps, tenantID, agents, err)
}

func ListAgentsBefore(ctx context.Context, deps AdminAgentDeps, beforeName, beforeID string, limit int) ([]AgentView, error) {
	tenantID := tenantports.TenantID(ctx)
	agents, err := deps.AgentRepo.ListPageBefore(ctx, tenantID, beforeName, beforeID, limit)
	return agentViews(ctx, deps, tenantID, agents, err)
}

func agentViews(ctx context.Context, deps AdminAgentDeps, tenantID string, agents []*agentdomain.Agent, err error) ([]AgentView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]AgentView, 0, len(agents))
	for _, agent := range agents {
		clientIDs, err := agentClientIDs(ctx, deps, tenantID, agent.ID)
		if err != nil {
			return nil, err
		}
		views = append(views, AgentView{Agent: agent, ClientIDs: clientIDs})
	}
	return views, nil
}

// GetAgent は Agent 本体と束縛済み client id を返す。別テナントの Agent は未存在
// として扱う。
func GetAgent(ctx context.Context, deps AdminAgentDeps, id string) (*AgentView, error) {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	clientIDs, err := agentClientIDs(ctx, deps, tenantID, agent.ID)
	if err != nil {
		return nil, err
	}
	return &AgentView{Agent: agent, ClientIDs: clientIDs}, nil
}

// RegisterAgentInput の Kind は必須である。区分は実行時にトークンを承認なしで発行して
// よいかを決めるため (REQ-OAUTH2-050)、既定値で補うとどちらへ倒しても誤る。空文字は
// 未指定、既知の列挙に無い値は不正として、それぞれ別の error で拒否する。
type RegisterAgentInput struct {
	ActorUserID string
	Name        string
	Description *string
	Kind        idmdomain.AgentKind
	OwnerUserID string
	Roles       []string
	Now         time.Time
}

func RegisterAgent(ctx context.Context, deps AdminAgentDeps, in RegisterAgentInput) (*agentdomain.Agent, error) {
	tenantID := tenantports.TenantID(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrAgentNameEmpty
	}
	if err := ensureAgentNameAvailable(ctx, deps, tenantID, name, ""); err != nil {
		return nil, err
	}
	if in.Kind == "" {
		return nil, ErrAgentKindRequired
	}
	if !in.Kind.Valid() {
		return nil, ErrAgentKindInvalid
	}
	owner := strings.TrimSpace(in.OwnerUserID)
	if owner == "" {
		owner = strings.TrimSpace(in.ActorUserID)
	}
	if owner == "" {
		return nil, ErrAgentOwnerRequired
	}
	if err := ensureAgentOwner(ctx, deps, tenantID, owner); err != nil {
		return nil, err
	}
	roles, err := idmusecases.NormalizeRoles(in.Roles)
	if err != nil {
		return nil, err
	}
	if err := idmusecases.ValidateRoleAssignment(idmusecases.RoleTargetAgent, tenantID, nil, roles); err != nil {
		return nil, err
	}
	now := idmusecases.NormalizedNow(in.Now)
	if err := idmusecases.CheckQuotaAndAudit(ctx, deps.QuotaRepo, deps.Emit, tenantID, tenancydomain.ResourceAgents, now); err != nil {
		return nil, err
	}
	id, err := agentdomain.NewAgentID()
	if err != nil {
		return nil, err
	}
	agent := &agentdomain.Agent{
		ID: id, TenantID: tenantID, Name: name, Description: idmusecases.NormalizeDescription(in.Description),
		Kind: in.Kind, OwnerUserID: owner, Status: idmdomain.AgentStatusActive,
		Roles: roles, CreatedAt: now, UpdatedAt: now,
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	if err := deps.AgentRepo.Save(ctx, agent); err != nil {
		return nil, err
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentRegistered{At: now, TenantID: agent.TenantID, ActorUserID: in.ActorUserID, AgentID: agent.ID}); err != nil {
		return nil, err
	}
	return agent, nil
}

type UpdateAgentInput struct {
	ActorUserID string
	ID          string
	Name        *string
	Description *string
	Kind        *idmdomain.AgentKind
	OwnerUserID *string
	Roles       *[]string
	Now         time.Time
}

func UpdateAgent(ctx context.Context, deps AdminAgentDeps, in UpdateAgentInput) (*agentdomain.Agent, error) {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, in.ID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	if agent.Status == idmdomain.AgentStatusKilled {
		return nil, ErrAgentKilled
	}
	// 入力の検証と、ほかの Agent と User を読む確認は、行を固定する前に行う。
	var name, owner string
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrAgentNameEmpty
		}
		if name != agent.Name {
			if err := ensureAgentNameAvailable(ctx, deps, tenantID, name, agent.ID); err != nil {
				return nil, err
			}
		}
	}
	if in.Kind != nil && !in.Kind.Valid() {
		return nil, ErrAgentKindInvalid
	}
	if in.OwnerUserID != nil {
		owner = strings.TrimSpace(*in.OwnerUserID)
		if owner == "" {
			return nil, ErrAgentOwnerRequired
		}
		if owner != agent.OwnerUserID {
			if err := ensureAgentOwner(ctx, deps, tenantID, owner); err != nil {
				return nil, err
			}
		}
	}
	var roles []string
	if in.Roles != nil {
		if roles, err = idmusecases.NormalizeRoles(*in.Roles); err != nil {
			return nil, err
		}
	}
	now := idmusecases.NormalizedNow(in.Now)
	var changed []string
	previousOwner := agent.OwnerUserID
	ownerChanged := false
	// 変わった項目の判定と停止の拒否は、読み直した最新の Agent に対して行う。先に読んだ Agent を
	// 保存すると、その間の所有者の停止に伴う無効化や停止を書き戻す。
	updated, err := deps.AgentRepo.Update(ctx, tenantID, agent.ID, func(latest *agentdomain.Agent) (bool, error) {
		if latest.Status == idmdomain.AgentStatusKilled {
			return false, ErrAgentKilled
		}
		changed, previousOwner, ownerChanged = []string{}, latest.OwnerUserID, false
		if in.Name != nil && name != latest.Name {
			latest.Name = name
			changed = append(changed, "name")
		}
		if in.Description != nil {
			if desc := idmusecases.NormalizeDescription(in.Description); !idmusecases.EqualOptionalString(latest.Description, desc) {
				latest.Description = desc
				changed = append(changed, "description")
			}
		}
		if in.Kind != nil && *in.Kind != latest.Kind {
			latest.Kind = *in.Kind
			changed = append(changed, "kind")
		}
		if in.OwnerUserID != nil && owner != latest.OwnerUserID {
			latest.OwnerUserID = owner
			changed = append(changed, "owner_sub")
			ownerChanged = true
		}
		if in.Roles != nil {
			if err := idmusecases.ValidateRoleAssignment(
				idmusecases.RoleTargetAgent, latest.TenantID, latest.Roles, roles,
			); err != nil {
				return false, err
			}
			if !slices.Equal(roles, latest.Roles) {
				latest.Roles = roles
				changed = append(changed, "roles")
			}
		}
		if len(changed) == 0 {
			return false, nil
		}
		latest.UpdatedAt = now
		return true, latest.Validate()
	})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrAgentNotFound
	}
	if len(changed) == 0 {
		return updated, nil
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentUpdated{
		At: now, TenantID: agent.TenantID, ActorUserID: in.ActorUserID, AgentID: agent.ID, ChangedFields: changed,
	}); err != nil {
		return nil, err
	}
	if ownerChanged {
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentOwnerChanged{
			At: now, TenantID: agent.TenantID, ActorUserID: in.ActorUserID, AgentID: agent.ID,
			PreviousOwnerUserID: previousOwner, NewOwnerUserID: updated.OwnerUserID,
		}); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// applyAgentState は、最新の Agent に状態の遷移を当てる。停止した Agent は変えずに拒否する。
func applyAgentState(transition func(*agentdomain.Agent)) agentports.AgentChange {
	return func(latest *agentdomain.Agent) (bool, error) {
		if latest.Status == idmdomain.AgentStatusKilled {
			return false, ErrAgentKilled
		}
		transition(latest)
		return true, latest.Validate()
	}
}

// SetAgentDisabled は Agent を運用停止 (disabled=true) / 再稼働 (disabled=false) する。
// Killed の Agent は復帰できないため reject する (一方向終端)。
func SetAgentDisabled(ctx context.Context, deps AdminAgentDeps, actorUserID, id string, disabled bool, now time.Time) (*agentdomain.Agent, error) {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	if agent.Status == idmdomain.AgentStatusKilled {
		return nil, ErrAgentKilled
	}
	if !disabled {
		if err := ensureAgentOwner(ctx, deps, tenantID, agent.OwnerUserID); errors.Is(err, ErrAgentOwnerNotFound) {
			return nil, ErrAgentOwnerInactive
		} else if err != nil {
			return nil, err
		}
	}
	now = idmusecases.NormalizedNow(now)
	updated, err := deps.AgentRepo.Update(ctx, tenantID, agent.ID, applyAgentState(func(latest *agentdomain.Agent) {
		latest.UpdatedAt = now
		if disabled {
			latest.Status = idmdomain.AgentStatusDisabled
			latest.DisabledAt = &now
		} else {
			latest.Status = idmdomain.AgentStatusActive
			latest.DisabledAt = nil
		}
	}))
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrAgentNotFound
	}
	var emitErr error
	if disabled {
		emitErr = idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentDisabled{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agent.ID})
	} else {
		emitErr = idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentEnabled{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agent.ID})
	}
	if emitErr != nil {
		return nil, emitErr
	}
	return updated, nil
}

// KillAgent は Agent を緊急停止し Killed (一方向終端) に遷移させる。既に Killed なら
// reject する (冪等ではなく irreversible なため明示エラー)。
func KillAgent(ctx context.Context, deps AdminAgentDeps, actorUserID, id string, now time.Time) (*agentdomain.Agent, error) {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	if agent.Status == idmdomain.AgentStatusKilled {
		return nil, ErrAgentKilled
	}
	now = idmusecases.NormalizedNow(now)
	updated, err := deps.AgentRepo.Update(ctx, tenantID, agent.ID, applyAgentState(func(latest *agentdomain.Agent) {
		latest.Status = idmdomain.AgentStatusKilled
		latest.KilledAt = &now
		latest.UpdatedAt = now
	}))
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrAgentNotFound
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentKilled{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agent.ID}); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteAgent は Agent を物理削除し、束縛は cascade で解除する。
func DeleteAgent(ctx context.Context, deps AdminAgentDeps, actorUserID, id string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}
	if agent.Status == idmdomain.AgentStatusKilled {
		return ErrAgentKilled
	}
	now = idmusecases.NormalizedNow(now)
	if err := deps.AgentRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	if deps.QuotaRepo != nil {
		if err := deps.QuotaRepo.Decrement(ctx, tenantID, tenancydomain.ResourceAgents, 1); err != nil {
			return err
		}
	}
	return idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentDeleted{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: id})
}

// BindCredential は Agent に同一テナントの OAuth2Client を束縛する。既束縛なら no-op
// で event も emit しない (冪等)。
func BindCredential(ctx context.Context, deps AdminAgentDeps, actorUserID, agentID, clientID string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}
	if agent.Status == idmdomain.AgentStatusKilled {
		return ErrAgentKilled
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ErrAgentClientNotFound
	}
	if deps.ClientRepo != nil {
		exists, provesKeyPossession, err := deps.ClientRepo.LookupBindableClient(ctx, tenantID, clientID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrAgentClientNotFound
		}
		// 束縛の後で認証方式を変えられるので、強制は OAuth2 のトークン要求ごとの判定
		// (REQ-OAUTH2-055) が担う。ここでは誤った構成を作った時点で理由を返す。
		if !provesKeyPossession {
			return ErrAgentClientAuthMethodNotAllowed
		}
	}
	existing, err := deps.AgentRepo.FindByClientID(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != agentID {
		return ErrAgentClientBound
	}
	now = idmusecases.NormalizedNow(now)
	added, err := deps.AgentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: agentID, ClientID: clientID, CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if !added {
		existing, err := deps.AgentRepo.FindByClientID(ctx, tenantID, clientID)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != agentID {
			return ErrAgentClientBound
		}
	}
	if added {
		return idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentCredentialBound{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agentID, ClientID: clientID,
		})
	}
	return nil
}

func ensureAgentOwner(ctx context.Context, deps AdminAgentDeps, tenantID, ownerUserID string) error {
	if deps.UserRepo == nil {
		return ErrAgentOwnerNotFound
	}
	user, err := deps.UserRepo.FindBySub(ctx, ownerUserID)
	if err != nil {
		return err
	}
	if user == nil || user.TenantID != tenantID || !user.IsActive() {
		return ErrAgentOwnerNotFound
	}
	return nil
}

// UnbindCredential は Agent から OAuth2Client の束縛を解除する。非束縛なら no-op で
// event も emit しない (冪等)。
func UnbindCredential(ctx context.Context, deps AdminAgentDeps, actorUserID, agentID, clientID string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}
	now = idmusecases.NormalizedNow(now)
	removed, err := deps.AgentRepo.RemoveBinding(ctx, tenantID, agentID, clientID)
	if err != nil {
		return err
	}
	if removed {
		return idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentCredentialUnbound{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agentID, ClientID: clientID,
		})
	}
	return nil
}

func agentClientIDs(ctx context.Context, deps AdminAgentDeps, tenantID, agentID string) ([]string, error) {
	bindings, err := deps.AgentRepo.ListBindings(ctx, tenantID, agentID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, binding.ClientID)
	}
	return out, nil
}

func ensureAgentNameAvailable(ctx context.Context, deps AdminAgentDeps, tenantID, name, excludeID string) error {
	agents, err := deps.AgentRepo.ListAll(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, agent := range agents {
		if agent.ID != excludeID && idmdomain.NameKey(agent.Name) == idmdomain.NameKey(name) {
			return ErrAgentNameConflict
		}
	}
	return nil
}
