package usecases

import (
	"context"
	"errors"
	"time"

	accessdomain "github.com/ambi/idmagic/backend/accesscontrol/internal/domain"
	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
)

// 主体の状態とモデルの未登録による不許可の理由。
const (
	reasonSubjectInactive = "subject_principal_inactive"
	reasonModelNotFound   = "authorization_model_not_found"
)

// RelationChecker は policyports.RelationChecker を CheckAccess の合成へ接続する。
type RelationChecker struct {
	Deps Deps
}

// CheckRelation は主体が有効なときだけ CheckAccess を評価する。診断の経路と異なり、結果が
// そのまま業務処理の許可になるので、トークンの発行後に無効にした主体をここで止める。
func (c RelationChecker) CheckRelation(ctx context.Context, check accessports.RelationCheck) (accessports.RelationDecision, error) {
	if !c.Deps.resolvePrincipalActive(ctx, check.TenantID, check.Subject.Type, check.Subject.ID) {
		return accessports.RelationDecision{Reasons: []string{reasonSubjectInactive}}, nil
	}
	actors := make([]Actor, 0, len(check.Actors))
	for _, actor := range check.Actors {
		actors = append(actors, Actor{Type: actor.Type, ID: actor.ID})
	}
	result, err := CheckAccess(ctx, c.Deps, CheckAccessInput{
		TenantID:       check.TenantID,
		Resource:       accessdomain.ObjectRef{Type: check.ResourceType, ID: check.ResourceID},
		Relation:       check.Relation,
		Subject:        accessdomain.SubjectRef{Type: check.Subject.Type, ID: check.Subject.ID},
		ActorChain:     actors,
		GrantedScopes:  check.GrantedScopes,
		RequiredScopes: check.RequiredScopes,
	}, time.Now().UTC())
	if errors.Is(err, accessdomain.ErrModelNotFound) {
		return accessports.RelationDecision{Reasons: []string{reasonModelNotFound}}, nil
	}
	if err != nil {
		return accessports.RelationDecision{}, err
	}
	return accessports.RelationDecision{Permitted: result.Permitted, Reasons: result.Reasons}, nil
}
