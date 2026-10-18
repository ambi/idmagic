// Package application は Application の保存先、状態更新とアクセス判定を組み立てる。
package application

import (
	"context"

	"github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type idManagementSubjectDirectory struct {
	users  userports.UserRepository
	groups groupports.GroupRepository
}

// SubjectDirectory は、管理とあるべき状態の割り当てが使う同じテナントの主体の照会を組み立てる。
func SubjectDirectory(users userports.UserRepository, groups groupports.GroupRepository) appports.SubjectDirectory {
	return idManagementSubjectDirectory{users: users, groups: groups}
}

func (d idManagementSubjectDirectory) SubjectExists(
	ctx context.Context,
	tenantID string,
	subjectType domain.AssignmentSubjectType,
	subjectID string,
) (bool, error) {
	switch subjectType {
	case domain.AssignmentSubjectUser:
		if d.users == nil {
			return false, nil
		}
		user, err := d.users.FindBySub(ctx, subjectID)
		if err != nil {
			return false, err
		}
		return user != nil && user.TenantID == tenantID, nil
	case domain.AssignmentSubjectGroup:
		if d.groups == nil {
			return false, nil
		}
		group, err := d.groups.FindByID(ctx, tenantID, subjectID)
		if err != nil {
			return false, err
		}
		return group != nil, nil
	default:
		return false, nil
	}
}

// Module は application context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	Repo                    appports.ApplicationRepository
	IconStore               appports.ApplicationIconStore
	AssignmentRepo          appports.AssignmentRepository
	OrderingRepo            appports.ApplicationOrderingRepository
	CategoryRepo            appports.ApplicationCategoryRepository
	SignInPolicyRepo        appports.SignInPolicyRepository
	DefaultSignInPolicyRepo appports.DefaultSignInPolicyRepository
	// ProvisioningNotifier is the outbound Provisioning boundary port (wi-45). nil means outbound provisioning is not wired.
	ProvisioningNotifier appports.ProvisioningNotifier
}

// Gate は Application 割当を fail-closed で判定する published capability を組み立てる
// (Application の公開契約 ApplicationAssignmentRef)。oauth2 / saml /
// wsfederation の federation 開始経路がこれを消費する。
func (m Module) Gate(groupRepo groupports.GroupRepository, trustedForwardedHops int) *appusecases.AccessGate {
	gate := &appusecases.AccessGate{
		ApplicationRepo:             m.Repo,
		ApplicationAssignmentRepo:   m.AssignmentRepo,
		ApplicationSignInPolicyRepo: m.SignInPolicyRepo,
		DefaultSignInPolicyRepo:     m.DefaultSignInPolicyRepo,
		GateTrustedForwardedHops:    trustedForwardedHops,
	}
	if groupRepo != nil {
		gate.GroupMemberships = idManagementGroupMemberships{groups: groupRepo}
	}
	return gate
}

// idManagementGroupMemberships は、割り当てのゲートが照合する Group の所属を IdManagement の
// Group の保存先から引く。
type idManagementGroupMemberships struct {
	groups groupports.GroupRepository
}

func (m idManagementGroupMemberships) GroupIDsOfUser(ctx context.Context, tenantID, userID string) ([]string, error) {
	groups, err := m.groups.ListGroupsByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids, nil
}

// DesiredStateAssignments は、ほかのモジュールが User への直接割り当てをあるべき状態として
// 適用する内部インターフェースを組み立てる。主体の実在は HTTP の割り当てと同じく
// IdManagement の保存先で確かめる。actorUserID は発行するイベントの actor になる。
func (m Module) DesiredStateAssignments(
	userRepo userports.UserRepository, groupRepo groupports.GroupRepository,
	emit func(spec.DomainEvent), actorUserID string,
) appusecases.DesiredStateAssignments {
	return appusecases.DesiredStateAssignments{
		Deps: appusecases.AssignmentDeps{
			Repo: m.Repo, AssignmentRepo: m.AssignmentRepo,
			SubjectDirectory:     SubjectDirectory(userRepo, groupRepo),
			Emit:                 emit,
			ProvisioningNotifier: m.ProvisioningNotifier,
		},
		ActorUserID: actorUserID,
	}
}

// ClientDisplayNames は client_id をアプリ表示名へ解決するリゾルバを組み立てる
// (oauth2 / authentication の監査・表示系が消費する)。
func (m Module) ClientDisplayNames(clientRepo appports.ClientNames) *ClientDisplayNameResolver {
	return &ClientDisplayNameResolver{ClientRepo: clientRepo, ApplicationRepo: m.Repo}
}
