package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// RelationRewriteKind は関係の成立条件を構成する規則の種別。
type RelationRewriteKind string

const (
	// RewriteDirect は (resource, relation) に直接書かれた RelationTuple を見る。
	RewriteDirect RelationRewriteKind = "direct"
	// RewriteComputedUserset は同じオブジェクト上の別関係へ委ねる。
	RewriteComputedUserset RelationRewriteKind = "computed_userset"
	// RewriteTupleToUserset は TuplesetRelation でたどった先のオブジェクトで
	// ComputedRelation を判定する。
	RewriteTupleToUserset RelationRewriteKind = "tuple_to_userset"
)

// RelationRewrite は関係の成立条件を構成する 1 つの規則。RelationDefinition は
// これらの和で成立条件を表す。交差と差集合は持たないので、規則やタプルの追加が
// 既存の許可を取り消すことはない。
type RelationRewrite struct {
	Kind RelationRewriteKind `json:"kind"`
	// ComputedRelation は computed_userset と tuple_to_userset で評価する関係名。
	ComputedRelation string `json:"computed_relation,omitempty"`
	// TuplesetRelation は tuple_to_userset で親オブジェクトへたどる関係名。
	TuplesetRelation string `json:"tupleset_relation,omitempty"`
	// DirectSubjectTypes は direct が受け入れる主体の形。`user` / `group#member` /
	// `user:*` の 3 形を取る。
	DirectSubjectTypes []string `json:"direct_subject_types,omitempty"`
}

// RelationDefinition はリソース型が公開する 1 つの関係。Rewrites が空の関係は
// 決して成立しない。
type RelationDefinition struct {
	Name     string            `json:"name"`
	Rewrites []RelationRewrite `json:"rewrites"`
}

// ResourceTypeDefinition は認可モデルが宣言するリソース型。
type ResourceTypeDefinition struct {
	Name      string               `json:"name"`
	Relations []RelationDefinition `json:"relations"`
}

// AuthorizationModel はテナントの認可モデルの 1 版。版は追記のみで、最新版が判定に使われる。
type AuthorizationModel struct {
	ID            string                   `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	Version       int                      `json:"version"`
	ResourceTypes []ResourceTypeDefinition `json:"resource_types"`
	CreatedAt     time.Time                `json:"created_at"`
}

func (m *AuthorizationModel) resourceType(name string) (ResourceTypeDefinition, bool) {
	for _, t := range m.ResourceTypes {
		if t.Name == name {
			return t, true
		}
	}
	return ResourceTypeDefinition{}, false
}

func (m *AuthorizationModel) relation(typeName, relation string) (RelationDefinition, bool) {
	t, ok := m.resourceType(typeName)
	if !ok {
		return RelationDefinition{}, false
	}
	for _, r := range t.Relations {
		if r.Name == relation {
			return r, true
		}
	}
	return RelationDefinition{}, false
}

// Validate はモデル全体の整合を検査する。未知の型・関係の参照、名前の書式違反、
// computed_userset の循環はいずれも登録を拒否する。実行時に初めて壊れるモデルを
// 受け入れないための、書き込み時点の fail-closed である。
func (m *AuthorizationModel) Validate() error {
	if len(m.ResourceTypes) == 0 {
		return fmt.Errorf("%w: at least one resource type is required", ErrModelInvalid)
	}
	seenTypes := map[string]struct{}{}
	for _, t := range m.ResourceTypes {
		if !nameRe.MatchString(t.Name) {
			return fmt.Errorf("%w: resource type %q", ErrModelInvalid, t.Name)
		}
		if _, dup := seenTypes[t.Name]; dup {
			return fmt.Errorf("%w: duplicate resource type %q", ErrModelInvalid, t.Name)
		}
		seenTypes[t.Name] = struct{}{}
		seenRelations := map[string]struct{}{}
		for _, r := range t.Relations {
			if !nameRe.MatchString(r.Name) {
				return fmt.Errorf("%w: relation %q on %q", ErrModelInvalid, r.Name, t.Name)
			}
			if _, dup := seenRelations[r.Name]; dup {
				return fmt.Errorf("%w: duplicate relation %q on %q", ErrModelInvalid, r.Name, t.Name)
			}
			seenRelations[r.Name] = struct{}{}
		}
	}
	for _, t := range m.ResourceTypes {
		for _, r := range t.Relations {
			if err := m.validateRelation(t, r); err != nil {
				return err
			}
		}
	}
	return m.validateNoComputedCycle()
}

func (m *AuthorizationModel) validateRelation(t ResourceTypeDefinition, r RelationDefinition) error {
	if len(r.Rewrites) == 0 {
		return fmt.Errorf("%w: relation %s#%s declares no rewrite", ErrModelInvalid, t.Name, r.Name)
	}
	for _, rw := range r.Rewrites {
		switch rw.Kind {
		case RewriteDirect:
			if rw.ComputedRelation != "" || rw.TuplesetRelation != "" {
				return fmt.Errorf("%w: direct rewrite on %s#%s must not name a relation", ErrModelInvalid, t.Name, r.Name)
			}
			if len(rw.DirectSubjectTypes) == 0 {
				return fmt.Errorf("%w: direct rewrite on %s#%s declares no subject type", ErrModelInvalid, t.Name, r.Name)
			}
			for _, form := range rw.DirectSubjectTypes {
				if err := m.validateSubjectForm(t, r, form); err != nil {
					return err
				}
			}
		case RewriteComputedUserset:
			if rw.TuplesetRelation != "" || len(rw.DirectSubjectTypes) > 0 {
				return fmt.Errorf("%w: computed_userset on %s#%s takes only computed_relation", ErrModelInvalid, t.Name, r.Name)
			}
			if _, ok := m.relation(t.Name, rw.ComputedRelation); !ok {
				return fmt.Errorf("%w: computed_userset on %s#%s references unknown relation %q", ErrModelInvalid, t.Name, r.Name, rw.ComputedRelation)
			}
			if rw.ComputedRelation == r.Name {
				return fmt.Errorf("%w: computed_userset on %s#%s references itself", ErrModelInvalid, t.Name, r.Name)
			}
		case RewriteTupleToUserset:
			if len(rw.DirectSubjectTypes) > 0 {
				return fmt.Errorf("%w: tuple_to_userset on %s#%s must not declare subject types", ErrModelInvalid, t.Name, r.Name)
			}
			tupleset, ok := m.relation(t.Name, rw.TuplesetRelation)
			if !ok {
				return fmt.Errorf("%w: tuple_to_userset on %s#%s references unknown tupleset relation %q", ErrModelInvalid, t.Name, r.Name, rw.TuplesetRelation)
			}
			targets := tuplesetTargetTypes(tupleset)
			if len(targets) == 0 {
				return fmt.Errorf("%w: tupleset relation %s#%s must accept plain object subjects", ErrModelInvalid, t.Name, rw.TuplesetRelation)
			}
			reachable := false
			for _, target := range targets {
				if _, ok := m.relation(target, rw.ComputedRelation); ok {
					reachable = true
				}
			}
			if !reachable {
				return fmt.Errorf("%w: tuple_to_userset on %s#%s references relation %q that no tupleset target declares", ErrModelInvalid, t.Name, r.Name, rw.ComputedRelation)
			}
		default:
			return fmt.Errorf("%w: unknown rewrite kind %q on %s#%s", ErrModelInvalid, rw.Kind, t.Name, r.Name)
		}
	}
	return nil
}

// validateSubjectForm は direct_subject_types の 1 要素を検査する。
// `user` / `group#member` / `user:*` の 3 形だけを受け付ける。
func (m *AuthorizationModel) validateSubjectForm(t ResourceTypeDefinition, r RelationDefinition, form string) error {
	invalid := fmt.Errorf("%w: direct subject type %q on %s#%s", ErrModelInvalid, form, t.Name, r.Name)
	switch {
	case strings.HasSuffix(form, ":"+Wildcard):
		typeName := strings.TrimSuffix(form, ":"+Wildcard)
		if _, ok := m.resourceType(typeName); !ok {
			return invalid
		}
	case strings.Contains(form, "#"):
		typeName, relation, _ := strings.Cut(form, "#")
		if _, ok := m.relation(typeName, relation); !ok {
			return invalid
		}
	default:
		if _, ok := m.resourceType(form); !ok {
			return invalid
		}
	}
	return nil
}

// tuplesetTargetTypes は tuple_to_userset がたどれる親オブジェクトの型を返す。
// subject set とワイルドカードは親として扱えないので除く。
func tuplesetTargetTypes(tupleset RelationDefinition) []string {
	var out []string
	for _, rw := range tupleset.Rewrites {
		if rw.Kind != RewriteDirect {
			continue
		}
		for _, form := range rw.DirectSubjectTypes {
			if strings.Contains(form, "#") || strings.HasSuffix(form, ":"+Wildcard) {
				continue
			}
			if !slices.Contains(out, form) {
				out = append(out, form)
			}
		}
	}
	return out
}

// validateNoComputedCycle は同一型内の computed_userset の連なりに循環がないことを
// 確かめる。tuple_to_userset の循環はタプルの内容に依存するため、探索側の
// 既訪問集合が受け持つ。
func (m *AuthorizationModel) validateNoComputedCycle() error {
	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var walk func(typeName, relation string) error
	walk = func(typeName, relation string) error {
		key := typeName + "#" + relation
		switch state[key] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("%w: computed_userset cycle through %s", ErrModelInvalid, key)
		}
		state[key] = visiting
		def, ok := m.relation(typeName, relation)
		if ok {
			for _, rw := range def.Rewrites {
				if rw.Kind != RewriteComputedUserset {
					continue
				}
				if err := walk(typeName, rw.ComputedRelation); err != nil {
					return err
				}
			}
		}
		state[key] = done
		return nil
	}
	for _, t := range m.ResourceTypes {
		for _, r := range t.Relations {
			if err := walk(t.Name, r.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateTuple はタプルが本モデルに適合するかを検査する。宣言のない型・関係、
// direct 規則が受け入れない主体の形は拒否する。
func (m *AuthorizationModel) ValidateTuple(t RelationTuple) error {
	if err := t.Validate(); err != nil {
		return err
	}
	def, ok := m.relation(t.Resource.Type, t.Relation)
	if !ok {
		return fmt.Errorf("%w: %s#%s is not declared", ErrTupleInvalid, t.Resource.Type, t.Relation)
	}
	if _, ok := m.resourceType(t.Subject.Type); !ok {
		return fmt.Errorf("%w: subject type %q is not declared", ErrTupleInvalid, t.Subject.Type)
	}
	if t.Subject.Relation != "" {
		if _, ok := m.relation(t.Subject.Type, t.Subject.Relation); !ok {
			return fmt.Errorf("%w: subject set %s is not declared", ErrTupleInvalid, t.Subject)
		}
	}
	for _, rw := range def.Rewrites {
		if rw.Kind == RewriteDirect && slices.Contains(rw.DirectSubjectTypes, t.Subject.form()) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s#%s does not accept subject form %q", ErrTupleInvalid, t.Resource.Type, t.Relation, t.Subject.form())
}
