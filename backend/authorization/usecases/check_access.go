package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// DefaultMaxEnumeratedResources は ListAccessibleResources が走査する候補の既定上限。
// 逆引きインデックスを持たない構成なので、無制限の走査を作らない。
const DefaultMaxEnumeratedResources = 500

// Actor は代行チェーンの 1 段。Type は認可モデルが宣言する主体型 (agent / user など)
// であり、判定でも有効性の解決でも同じ語彙を使う。
type Actor struct {
	Type string
	ID   string
}

// CheckAccessInput は 1 回の判定の入力。
type CheckAccessInput struct {
	TenantID           string
	Resource           domain.ObjectRef
	Relation           string
	Subject            domain.SubjectRef
	ActorChain         []Actor
	MinimumConsistency string
	// GrantedScopes は提示されたトークンのスコープ集合。
	GrantedScopes []string
	// RequiredScopes はこの関係を要求するために必要なスコープ。空なら制約しない。
	RequiredScopes []string
}

// CheckAccessResult は判定の結果。Reasons は拒否した規則名と関係評価の拒否理由で、
// 識別子を含まない。
type CheckAccessResult struct {
	Permitted    bool
	ModelVersion int
	Consistency  string
	RelationPath []string
	Reasons      []string
}

// CheckAccess は関係の成否を事実として組み立て、AuthZEN の Authorizer へ渡す。
// 合成規則そのものは評価器側にあり、ここは事実の供給に徹する。モデルの未登録、
// 整合トークンの不足、ストア障害、評価器への到達不能はいずれも error であり、
// 呼び出し側はこれを許可へ退避してはならない。
func CheckAccess(ctx context.Context, d Deps, input CheckAccessInput, now time.Time) (CheckAccessResult, error) {
	return checkAccess(ctx, d, input, now, true)
}

func checkAccess(ctx context.Context, d Deps, input CheckAccessInput, now time.Time, audit bool) (CheckAccessResult, error) {
	if d.Authorizer == nil {
		return CheckAccessResult{}, ErrAuthorizerUnavailable
	}
	model, err := loadModel(ctx, d, input.TenantID, 0)
	if err != nil {
		return CheckAccessResult{}, err
	}
	storeVersion, err := d.assertConsistency(ctx, input.TenantID, input.MinimumConsistency)
	if err != nil {
		return CheckAccessResult{}, err
	}

	facts, actors, err := d.relationshipFacts(ctx, model, input)
	if err != nil {
		return CheckAccessResult{}, err
	}

	response, err := d.Authorizer.Authorize(ctx, spec.AuthZRequest{
		Subject: spec.AuthZSubject{
			Type: "User", ID: input.Subject.ID,
			Properties: spec.AuthZSubjectProps{TenantID: input.TenantID, Scopes: input.GrantedScopes},
		},
		Action: spec.ActionResourceAccess,
		Resource: spec.AuthZResource{
			Type: input.Resource.Type, ID: input.Resource.ID,
			Properties: spec.AuthZResourceProps{TenantID: input.TenantID, Scopes: input.RequiredScopes},
		},
		Context: spec.AuthZContext{Now: now, ActorChain: actors, Relationship: &facts},
	})
	if err != nil {
		return CheckAccessResult{}, fmt.Errorf("%w: %w", ErrAuthorizerUnavailable, err)
	}

	result := CheckAccessResult{
		Permitted:    response.Permit,
		ModelVersion: model.Version,
		Consistency:  domain.EncodeConsistencyToken(input.TenantID, storeVersion),
		RelationPath: facts.RelationPath,
		Reasons:      mergeReasons(response.Reasons, facts.DenyReasons),
	}
	if audit {
		d.emit(&domain.FgaCheckEvaluated{
			At: now, TenantID: input.TenantID, ResourceType: input.Resource.Type,
			ResourceIDDigest: ResourceIDDigest(input.TenantID, input.Resource),
			Relation:         input.Relation, Permitted: result.Permitted, ModelVersion: model.Version,
			RelationPath: result.RelationPath, Reasons: result.Reasons, ActorChainDepth: len(input.ActorChain),
		})
	}
	return result, nil
}

// ListedResources は列挙の結果。Truncated が立った結果を完全な一覧として扱っては
// ならない。
type ListedResources struct {
	ResourceIDs  []string
	Truncated    bool
	ModelVersion int
	Consistency  string
}

// ListAccessibleResourcesInput は列挙の入力。
type ListAccessibleResourcesInput struct {
	TenantID           string
	ResourceType       string
	Relation           string
	Subject            domain.SubjectRef
	ActorChain         []Actor
	MinimumConsistency string
	GrantedScopes      []string
	RequiredScopes     []string
}

// ListAccessibleResources は候補を上限つきで走査し、CheckAccess と同じ合成で
// 許可されたものだけを返す。上限に達したら Truncated を立てる。
func ListAccessibleResources(ctx context.Context, d Deps, input ListAccessibleResourcesInput, now time.Time) (ListedResources, error) {
	limit := d.MaxEnumeratedResources
	if limit <= 0 {
		limit = DefaultMaxEnumeratedResources
	}
	// 上限より 1 件多く引き、打ち切りが起きたかどうかを取りこぼさない。
	candidates, err := d.Tuples.ListResourceIDs(ctx, input.TenantID, input.ResourceType, limit+1)
	if err != nil {
		return ListedResources{}, err
	}
	truncated := len(candidates) > limit
	if truncated {
		candidates = candidates[:limit]
	}

	out := ListedResources{ResourceIDs: []string{}, Truncated: truncated}
	for _, id := range candidates {
		// 1 件ごとの判定は監査へ展開しない。走査全体を 1 件の
		// FgaResourcesEnumerated にまとめる。
		result, err := checkAccess(ctx, d, CheckAccessInput{
			TenantID: input.TenantID,
			Resource: domain.ObjectRef{Type: input.ResourceType, ID: id},
			Relation: input.Relation, Subject: input.Subject, ActorChain: input.ActorChain,
			MinimumConsistency: input.MinimumConsistency,
			GrantedScopes:      input.GrantedScopes, RequiredScopes: input.RequiredScopes,
		}, now, false)
		if err != nil {
			return ListedResources{}, err
		}
		out.ModelVersion = result.ModelVersion
		out.Consistency = result.Consistency
		if result.Permitted {
			out.ResourceIDs = append(out.ResourceIDs, id)
		}
	}
	if out.Consistency == "" {
		// 候補が 1 件も無い場合でも、呼び出し側が読んだ時点の版は返す。
		version, err := d.Tuples.Version(ctx, input.TenantID)
		if err != nil {
			return ListedResources{}, err
		}
		model, err := loadModel(ctx, d, input.TenantID, 0)
		if err != nil {
			return ListedResources{}, err
		}
		out.ModelVersion = model.Version
		out.Consistency = domain.EncodeConsistencyToken(input.TenantID, version)
	}
	d.emit(&domain.FgaResourcesEnumerated{
		At: now, TenantID: input.TenantID, ResourceType: input.ResourceType, Relation: input.Relation,
		CandidateCount: len(candidates), PermittedCount: len(out.ResourceIDs), Truncated: out.Truncated,
		ModelVersion: out.ModelVersion, ActorChainDepth: len(input.ActorChain),
	})
	return out, nil
}

// assertConsistency は提示された整合トークンをストアの書き込み版と突き合わせる。
// トークンが別テナントのもの、壊れている、ストアがまだ追いついていない場合は拒否する。
func (d Deps) assertConsistency(ctx context.Context, tenantID, token string) (int64, error) {
	storeVersion, err := d.Tuples.Version(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	if token == "" {
		return storeVersion, nil
	}
	required, err := domain.DecodeConsistencyToken(token, tenantID)
	if err != nil {
		return 0, err
	}
	if storeVersion < required {
		return 0, fmt.Errorf("%w: store is at %d, %d required", domain.ErrConsistencyNotSatisfied, storeVersion, required)
	}
	return storeVersion, nil
}

// relationshipFacts は主体と代行チェーンそれぞれの関係判定を走らせ、AuthZEN の
// 判定 context に載せる事実を組み立てる。エージェントはユーザーの権限を超えられない
// ので、チェーン上の 1 段でも関係を欠けば ActorChainPermitted は立たない。
func (d Deps) relationshipFacts(
	ctx context.Context,
	model *domain.AuthorizationModel,
	input CheckAccessInput,
) (spec.AuthZRelationshipFacts, []spec.AuthZActor, error) {
	opts := domain.CheckOptions{MaxDepth: d.MaxDepth}
	subjectDecision, err := model.Check(ctx, d.Tuples, input.TenantID, input.Resource, input.Relation, input.Subject, opts)
	if err != nil {
		return spec.AuthZRelationshipFacts{}, nil, err
	}
	facts := spec.AuthZRelationshipFacts{
		Evaluated:           true,
		SubjectPermitted:    subjectDecision.Permitted,
		ActorChainPermitted: true,
		ModelVersion:        model.Version,
		RelationPath:        subjectDecision.Path,
		DenyReasons:         subjectDecision.Reasons,
	}

	actors := make([]spec.AuthZActor, 0, len(input.ActorChain))
	for _, actor := range input.ActorChain {
		decision, err := model.Check(ctx, d.Tuples, input.TenantID, input.Resource, input.Relation,
			domain.SubjectRef{Type: actor.Type, ID: actor.ID}, opts)
		if err != nil {
			return spec.AuthZRelationshipFacts{}, nil, err
		}
		if !decision.Permitted {
			facts.ActorChainPermitted = false
			facts.DenyReasons = mergeReasons(facts.DenyReasons, decision.Reasons)
		}
		actors = append(actors, spec.AuthZActor{
			Type: actor.Type, ID: actor.ID,
			Active: d.resolvePrincipalActive(ctx, input.TenantID, actor.Type, actor.ID),
		})
	}
	return facts, actors, nil
}

// ResourceIDDigest は監査に残すリソース識別子のダイジェスト。テナントと型を
// 混ぜるので、同じ資源への繰り返しアクセスは相関できるが、資源の名前そのものは
// 監査ログから復元できない。
func ResourceIDDigest(tenantID string, resource domain.ObjectRef) string {
	sum := sha256.Sum256([]byte(tenantID + ":" + resource.Type + ":" + resource.ID))
	return hex.EncodeToString(sum[:])[:16]
}

func mergeReasons(base, extra []string) []string {
	out := slices.Clone(base)
	for _, reason := range extra {
		if !slices.Contains(out, reason) {
			out = append(out, reason)
		}
	}
	return out
}
