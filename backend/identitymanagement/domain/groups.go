package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ===============================================================
// Group 集約 (ADR-038)
// ===============================================================

// Group は tenant-scoped なロール束集約。所属する User に roles[] を一斉付与する。
// 階層・deny ルール・属性自動所属は持たない (effective_roles は union のみ)。
// ID は不変の生成識別子 (group_<uuid>)、Name はテナント内で一意な編集可能ラベル。
type Group struct {
	ID             string              `json:"id"`
	TenantID       string              `json:"tenant_id"`
	Name           string              `json:"name"`
	Description    *string             `json:"description,omitempty"`
	Roles          []string            `json:"roles"`
	MembershipType GroupMembershipType `json:"membership_type"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
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
	"Roles":       z.Slice(z.String().Min(1)),
	"CreatedAt":   z.Time().Required(),
	"UpdatedAt":   z.Time().Required(),
})

func (g Group) Validate() error {
	if !g.MembershipType.Valid() {
		return fmt.Errorf("invalid group membership type %q", g.MembershipType)
	}
	return spec.Validate(groupSchema, &g)
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

// NewGroupID は不変の Group 識別子 group_<uuid> を生成する。
func NewGroupID() (string, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return "", err
	}
	return "group_" + id, nil
}

// EffectiveRoles は認可で用いる有効ロール集合を返す (ADR-038)。
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
