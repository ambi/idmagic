package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// Wildcard は「その型の主体すべて」を表す主体識別子。direct 規則が
// `<type>:*` を宣言している場合にだけ受け付ける。
const Wildcard = "*"

// 型名・関係名は小文字の識別子に限る。永続化の CHECK 制約と同じ書式であり、
// 表示名や自由記述をここへ入れさせない。
var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ObjectRef は `type:id` 形式のオブジェクト参照。
type ObjectRef struct {
	Type string
	ID   string
}

func (o ObjectRef) String() string { return o.Type + ":" + o.ID }

// SubjectRef は関係タプルの主体。Relation が空でなければ subject set
// (`group:eng#member`) を、ID が Wildcard ならワイルドカードを表す。
type SubjectRef struct {
	Type     string
	ID       string
	Relation string
}

func (s SubjectRef) String() string {
	if s.Relation == "" {
		return s.Type + ":" + s.ID
	}
	return s.Type + ":" + s.ID + "#" + s.Relation
}

// Object は subject set / ワイルドカードでない主体を、たどる先のオブジェクトとして返す。
func (s SubjectRef) Object() ObjectRef { return ObjectRef{Type: s.Type, ID: s.ID} }

// form は direct_subject_types の宣言と突き合わせるための形の表現を返す。
// `user` (個別)、`group#member` (subject set)、`user:*` (ワイルドカード) の 3 形。
func (s SubjectRef) form() string {
	switch {
	case s.Relation != "":
		return s.Type + "#" + s.Relation
	case s.ID == Wildcard:
		return s.Type + ":" + Wildcard
	default:
		return s.Type
	}
}

// RelationTuple は (resource, relation, subject) の関係事実。テナント内で
// この組が一意であり、同じ組の再書き込みは冪等である。
type RelationTuple struct {
	Resource ObjectRef
	Relation string
	Subject  SubjectRef
}

func (t RelationTuple) String() string {
	return t.Resource.String() + "#" + t.Relation + "@" + t.Subject.String()
}

// Key は writes と deletes の重複検出、およびメモリアダプターの一意キーに使う。
func (t RelationTuple) Key() string { return t.String() }

// Validate は書式だけを検査する。モデルへの適合は AuthorizationModel.ValidateTuple が見る。
func (t RelationTuple) Validate() error {
	if !nameRe.MatchString(t.Resource.Type) {
		return fmt.Errorf("%w: resource type %q", ErrTupleInvalid, t.Resource.Type)
	}
	if !nameRe.MatchString(t.Relation) {
		return fmt.Errorf("%w: relation %q", ErrTupleInvalid, t.Relation)
	}
	if !nameRe.MatchString(t.Subject.Type) {
		return fmt.Errorf("%w: subject type %q", ErrTupleInvalid, t.Subject.Type)
	}
	if t.Subject.Relation != "" && !nameRe.MatchString(t.Subject.Relation) {
		return fmt.Errorf("%w: subject relation %q", ErrTupleInvalid, t.Subject.Relation)
	}
	if err := validateIdentifier("resource id", t.Resource.ID); err != nil {
		return err
	}
	if err := validateIdentifier("subject id", t.Subject.ID); err != nil {
		return err
	}
	if t.Subject.ID == Wildcard && t.Subject.Relation != "" {
		return fmt.Errorf("%w: wildcard subject must not carry a relation", ErrTupleInvalid)
	}
	if t.Resource.ID == Wildcard {
		return fmt.Errorf("%w: resource id must not be a wildcard", ErrTupleInvalid)
	}
	return nil
}

// 識別子は idmagic が採番せず呼び出し側の資源空間から来るため書式を強制できない。
// 空・過長・区切り文字の混入だけを拒否し、解析可能性を保つ。
func validateIdentifier(label, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrTupleInvalid, label)
	}
	if utf8.RuneCountInString(value) > spec.LengthExternalID {
		return fmt.Errorf("%w: %s exceeds %d characters", ErrTupleInvalid, label, spec.LengthExternalID)
	}
	if strings.ContainsAny(value, ":#") {
		return fmt.Errorf("%w: %s must not contain ':' or '#'", ErrTupleInvalid, label)
	}
	return nil
}
