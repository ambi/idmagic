package domain

import (
	"slices"
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
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Roles       []string  `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	return spec.Validate(groupSchema, &g)
}

// GroupMember は User と Group の所属関係。group_id × user_sub で一意。
type GroupMember struct {
	GroupID   string    `json:"group_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

var groupMemberSchema = z.Struct(z.Shape{
	"GroupID":   z.String().Min(1).Required(),
	"UserID":    z.String().Min(1).Required(),
	"CreatedAt": z.Time().Required(),
})

func (m GroupMember) Validate() error {
	return spec.Validate(groupMemberSchema, &m)
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
