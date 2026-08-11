package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	z "github.com/Oudwins/zog"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// groupAttrKeyPattern は Group custom 属性キーの命名規則: snake_case、英字始まり。
// userdomain の同等パターンとは別定義 (unexported のため共有できない) だが規則は同じ。
var groupAttrKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// ===============================================================
// Group 集約
// ===============================================================

// Group は tenant-scoped なロール束集約。所属する User に roles[] を一斉付与する。
// 階層・deny ルール・属性自動所属は持たない (effective_roles は union のみ)。
// ID は不変の生成識別子 (group_<uuid>)、Name はテナント内で一意な編集可能ラベル。
// Email はグループ宛の連絡先アドレス (認証・検証フローは持たない)。Attributes は
// TenantGroupAttributeSchema に対して検証される sparse な custom 属性 bag で、値の型は
// User の AttributeValue をそのまま再利用する。
type Group struct {
	ID             string                               `json:"id"`
	TenantID       string                               `json:"tenant_id"`
	Name           string                               `json:"name"`
	Description    *string                              `json:"description,omitempty"`
	Email          *string                              `json:"email,omitempty"`
	Attributes     map[string]userdomain.AttributeValue `json:"attributes,omitempty"`
	Roles          []string                             `json:"roles"`
	MembershipType GroupMembershipType                  `json:"membership_type"`
	CreatedAt      time.Time                            `json:"created_at"`
	UpdatedAt      time.Time                            `json:"updated_at"`
}

type GroupMembershipType string

const (
	GroupMembershipManual  GroupMembershipType = "manual"
	GroupMembershipDynamic GroupMembershipType = "dynamic"
)

func (t GroupMembershipType) Effective() GroupMembershipType {
	if t == "" {
		return GroupMembershipManual
	}
	return t
}

func (t GroupMembershipType) Valid() bool {
	return t == "" || t == GroupMembershipManual || t == GroupMembershipDynamic
}

type GroupMembershipSource string

const (
	MembershipSourceManual      GroupMembershipSource = "manual"
	MembershipSourceDynamicRule GroupMembershipSource = "dynamic_rule"
)

func (s GroupMembershipSource) Effective() GroupMembershipSource {
	if s == "" {
		return MembershipSourceManual
	}
	return s
}

var groupSchema = z.Struct(z.Shape{
	"ID":          z.String().Min(1).Max(64).Required(),
	"TenantID":    z.String().Min(1).Required(),
	"Name":        z.String().Min(1).Max(100).Required(),
	"Description": z.Ptr(z.String().Max(500)),
	"Email":       z.Ptr(z.String().Email()),
	"Roles":       z.Slice(z.String().Min(1)),
	"CreatedAt":   z.Time().Required(),
	"UpdatedAt":   z.Time().Required(),
})

func (g Group) Validate() error {
	if !g.MembershipType.Valid() {
		return fmt.Errorf("invalid group membership type %q", g.MembershipType)
	}
	if err := spec.Validate(groupSchema, &g); err != nil {
		return err
	}
	for key, v := range g.Attributes {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("attribute %q: %w", key, err)
		}
	}
	return nil
}

// GroupMember は User と Group の所属関係。group_id × user_sub で一意。
type GroupMember struct {
	GroupID     string                `json:"group_id"`
	UserID      string                `json:"user_id"`
	Source      GroupMembershipSource `json:"source"`
	RuleVersion *int64                `json:"rule_version,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

var groupMemberSchema = z.Struct(z.Shape{
	"GroupID":   z.String().Min(1).Required(),
	"UserID":    z.String().Min(1).Required(),
	"CreatedAt": z.Time().Required(),
})

func (m GroupMember) Validate() error {
	if m.Source == "" {
		m.Source = MembershipSourceManual
	}
	if m.Source != MembershipSourceManual && m.Source != MembershipSourceDynamicRule {
		return fmt.Errorf("invalid membership source %q", m.Source)
	}
	if m.Source == MembershipSourceDynamicRule && m.RuleVersion == nil {
		return fmt.Errorf("dynamic membership requires rule version")
	}
	return spec.Validate(groupMemberSchema, &m)
}

type DynamicGroupRule struct {
	GroupID              string    `json:"group_id"`
	TenantID             string    `json:"tenant_id"`
	Expression           string    `json:"expression"`
	Enabled              bool      `json:"enabled"`
	Version              int64     `json:"version"`
	ReferencedAttributes []string  `json:"referenced_attributes"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (r DynamicGroupRule) Validate() error {
	if r.GroupID == "" || r.TenantID == "" || strings.TrimSpace(r.Expression) == "" {
		return fmt.Errorf("dynamic group rule identifiers and expression are required")
	}
	if len(r.Expression) > 4096 || r.Version < 1 {
		return fmt.Errorf("invalid dynamic group rule limits")
	}
	return nil
}

// GroupAttributeDef は Group の custom 属性 1 件の定義。User の UserAttributeDef と異なり、
// Group には builtin OIDC/SCIM カタログ・self-service 編集主体・claim 露出が無いため、
// 型と要否だけを持つ簡略版。
type GroupAttributeDef struct {
	Key         string                  `json:"key"`
	Label       string                  `json:"label,omitempty"`
	Type        idmdomain.AttributeType `json:"type"`
	MultiValued bool                    `json:"multi_valued"`
	Required    bool                    `json:"required"`
}

var groupAttributeDefSchema = z.Struct(z.Shape{
	"Key": z.String().TestFunc(
		func(value *string, _ z.Ctx) bool {
			return value != nil && groupAttrKeyPattern.MatchString(*value)
		},
		z.Message("attribute key must be snake_case starting with a letter"),
	).Required(),
	"Type": z.StringLike[idmdomain.AttributeType]().TestFunc(
		func(value *idmdomain.AttributeType, _ z.Ctx) bool { return value.Valid() },
		z.Message("attribute type is not in enum"),
	).Required(),
	"Label": z.String().Max(100),
})

func (d GroupAttributeDef) Validate() error { return spec.Validate(groupAttributeDefSchema, &d) }

// TenantGroupAttributeSchema は tenant 単位の Group custom 属性定義集合。User と異なり
// builtin catalog が無いため、本集合がそのまま実効定義になる。tenant aggregate には
// 埋め込まず独立 aggregate として持ち、tenant 削除時に cascade する。
type TenantGroupAttributeSchema struct {
	TenantID   string              `json:"tenant_id"`
	Attributes []GroupAttributeDef `json:"attributes"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

func (s TenantGroupAttributeSchema) Validate() error {
	seen := map[string]bool{}
	for _, d := range s.Attributes {
		if err := d.Validate(); err != nil {
			return err
		}
		if seen[d.Key] {
			return fmt.Errorf("duplicate custom attribute key %q", d.Key)
		}
		seen[d.Key] = true
	}
	return nil
}

// EffectiveDefs は実効定義を返す。Group には union すべき builtin catalog が無いため、
// tenant 定義そのものが実効定義になる。
func (s TenantGroupAttributeSchema) EffectiveDefs() []GroupAttributeDef {
	return s.Attributes
}

// ValidateGroupAttributeValue は属性値 1 件を定義に対して検証する (required は見ない)。
func ValidateGroupAttributeValue(value userdomain.AttributeValue, def GroupAttributeDef) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Type != def.Type {
		return fmt.Errorf("expects type %q, got %q", def.Type, value.Type)
	}
	if def.MultiValued != (def.Type == idmdomain.AttributeTypeStringArray) {
		return fmt.Errorf("multi_valued/type mismatch")
	}
	return nil
}

// ValidateGroupAttributes は Group.Attributes を実効属性定義に対して検証する。
// 未定義 key の拒否、型の一致、multi_valued の整合、required の充足を見る。
func ValidateGroupAttributes(values map[string]userdomain.AttributeValue, defs []GroupAttributeDef) error {
	byKey := make(map[string]GroupAttributeDef, len(defs))
	for _, d := range defs {
		byKey[d.Key] = d
	}
	for key, v := range values {
		def, ok := byKey[key]
		if !ok {
			return fmt.Errorf("attribute %q is not defined", key)
		}
		if err := ValidateGroupAttributeValue(v, def); err != nil {
			return fmt.Errorf("attribute %q: %w", key, err)
		}
	}
	for _, def := range defs {
		if def.Required {
			if _, ok := values[def.Key]; !ok {
				return fmt.Errorf("required attribute %q is missing", def.Key)
			}
		}
	}
	return nil
}

// NewGroupID は不変の Group 識別子 (UUID v4) を生成する。
func NewGroupID() (string, error) {
	return spec.NewUUIDv4()
}

// EffectiveRoles は認可で用いる有効ロール集合を返す。
// effective_roles(user) = user.roles ∪ ⋃_{g ∈ groups} g.roles。
// 結果はソート済みで重複を含まない。所属グループが空なら user.roles に一致する。
func EffectiveRoles(userRoles []string, groups []*Group) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(userRoles))
	add := func(roles []string) {
		for _, role := range roles {
			if role == "" {
				continue
			}
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		}
	}
	add(userRoles)
	for _, group := range groups {
		if group != nil {
			add(group.Roles)
		}
	}
	slices.Sort(out)
	return out
}
