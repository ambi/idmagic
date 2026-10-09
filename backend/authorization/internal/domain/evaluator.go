package domain

import (
	"context"
	"slices"
	"sort"
)

// DefaultMaxDepth は関係グラフをたどる深さの上限。超えた探索は「判定不能」ではなく
// 拒否理由付きの不許可になる。
const DefaultMaxDepth = 8

// 拒否理由。判定結果に残し、運用者がモデルの誤りを追えるようにする。
// リソース識別子や主体識別子は含めない。
const (
	ReasonNoRelationPath    = "no_relation_path"
	ReasonUnknownRelation   = "unknown_relation"
	ReasonDepthExceeded     = "evaluation_depth_exceeded"
	ReasonCycleDetected     = "relation_cycle_detected"
	ReasonSubjectFormDenied = "subject_form_not_declared"
)

// TupleReader は評価器が必要とする最小の読み出し。ports の Repository が満たす。
type TupleReader interface {
	// ListSubjects は (resource, relation) に直接書かれた主体を返す。
	ListSubjects(ctx context.Context, tenantID string, resource ObjectRef, relation string) ([]SubjectRef, error)
}

// CheckOptions は評価のふるまいを絞る。ゼロ値は DefaultMaxDepth を使う。
type CheckOptions struct {
	MaxDepth int
}

// Decision は 1 回の関係判定の結果。Path はたどった関係名だけの経路であり、
// オブジェクト識別子と主体識別子を含まない。
type Decision struct {
	Permitted bool
	Path      []string
	Reasons   []string
}

// Check は subject が resource に対して relation を持つかを、深さ制限つきの
// 深さ優先探索で判定する。深さ超過・循環・未知の関係はいずれも許可しない。
// ストアの読み出しに失敗した場合だけ error を返す。呼び出し側はこれを許可へ
// 退避させてはならない。
func (m *AuthorizationModel) Check(
	ctx context.Context,
	reader TupleReader,
	tenantID string,
	resource ObjectRef,
	relation string,
	subject SubjectRef,
	opts CheckOptions,
) (Decision, error) {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	ev := &evaluation{
		model:    m,
		reader:   reader,
		tenantID: tenantID,
		subject:  subject,
		maxDepth: maxDepth,
		visiting: map[string]struct{}{},
		reasons:  map[string]struct{}{},
	}
	permitted, err := ev.check(ctx, resource, relation, 0)
	if err != nil {
		return Decision{}, err
	}
	if permitted {
		return Decision{Permitted: true, Path: ev.found}, nil
	}
	return Decision{Permitted: false, Reasons: ev.collectedReasons()}, nil
}

type evaluation struct {
	model    *AuthorizationModel
	reader   TupleReader
	tenantID string
	subject  SubjectRef
	maxDepth int
	visiting map[string]struct{}
	path     []string
	found    []string
	reasons  map[string]struct{}
}

func (e *evaluation) addReason(reason string) { e.reasons[reason] = struct{}{} }

func (e *evaluation) collectedReasons() []string {
	if len(e.reasons) == 0 {
		return []string{ReasonNoRelationPath}
	}
	out := make([]string, 0, len(e.reasons))
	for reason := range e.reasons {
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}

func (e *evaluation) check(ctx context.Context, resource ObjectRef, relation string, depth int) (bool, error) {
	if depth >= e.maxDepth {
		e.addReason(ReasonDepthExceeded)
		return false, nil
	}
	key := resource.String() + "#" + relation
	if _, ok := e.visiting[key]; ok {
		e.addReason(ReasonCycleDetected)
		return false, nil
	}
	def, ok := e.model.relation(resource.Type, relation)
	if !ok {
		e.addReason(ReasonUnknownRelation)
		return false, nil
	}

	e.visiting[key] = struct{}{}
	e.path = append(e.path, resource.Type+"#"+relation)
	defer func() {
		delete(e.visiting, key)
		e.path = e.path[:len(e.path)-1]
	}()

	for _, rw := range def.Rewrites {
		permitted, err := e.applyRewrite(ctx, resource, relation, rw, depth)
		if err != nil {
			return false, err
		}
		if permitted {
			// 最初に成立した最深フレームが完全な経路を持つ。巻き戻しの途中で
			// 短い経路に置き換えないよう、一度だけ確定する。
			if e.found == nil {
				e.found = slices.Clone(e.path)
			}
			return true, nil
		}
	}
	return false, nil
}

func (e *evaluation) applyRewrite(
	ctx context.Context,
	resource ObjectRef,
	relation string,
	rw RelationRewrite,
	depth int,
) (bool, error) {
	switch rw.Kind {
	case RewriteDirect:
		return e.applyDirect(ctx, resource, relation, rw, depth)
	case RewriteComputedUserset:
		return e.check(ctx, resource, rw.ComputedRelation, depth+1)
	case RewriteTupleToUserset:
		return e.applyTupleToUserset(ctx, resource, rw, depth)
	default:
		e.addReason(ReasonUnknownRelation)
		return false, nil
	}
}

func (e *evaluation) applyDirect(
	ctx context.Context,
	resource ObjectRef,
	relation string,
	rw RelationRewrite,
	depth int,
) (bool, error) {
	subjects, err := e.reader.ListSubjects(ctx, e.tenantID, resource, relation)
	if err != nil {
		return false, err
	}
	for _, s := range subjects {
		// モデルが受け入れる形から外れた既存タプルは数えない。モデルの縮小が
		// 古いタプルに追い越されないための多層防御である。
		if !slices.Contains(rw.DirectSubjectTypes, s.form()) {
			e.addReason(ReasonSubjectFormDenied)
			continue
		}
		if s.Relation != "" {
			permitted, err := e.check(ctx, s.Object(), s.Relation, depth+1)
			if err != nil {
				return false, err
			}
			if permitted {
				return true, nil
			}
			continue
		}
		if s.Type != e.subject.Type {
			continue
		}
		if s.ID == Wildcard || s.ID == e.subject.ID {
			return true, nil
		}
	}
	return false, nil
}

func (e *evaluation) applyTupleToUserset(
	ctx context.Context,
	resource ObjectRef,
	rw RelationRewrite,
	depth int,
) (bool, error) {
	parents, err := e.reader.ListSubjects(ctx, e.tenantID, resource, rw.TuplesetRelation)
	if err != nil {
		return false, err
	}
	for _, parent := range parents {
		// tupleset の先は必ず具体的なオブジェクトである。subject set や
		// ワイルドカードをたどる先には使えない。
		if parent.Relation != "" || parent.ID == Wildcard {
			continue
		}
		permitted, err := e.check(ctx, parent.Object(), rw.ComputedRelation, depth+1)
		if err != nil {
			return false, err
		}
		if permitted {
			return true, nil
		}
	}
	return false, nil
}
