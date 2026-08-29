package domain

// Group CSV の方言。転送ポリシー、解析器、直列化器、可逆なセル変換は idmdomain の
// 種別非依存な CSV 基盤が持ち、ここには Group 固有の列の語彙、字句形、行計画の型だけを置く。
// 語彙が閉じているのは、未検証の項目が CSV という便宜的な経路から出入りしないためである
// (docs/contexts/identity-management/internals.md)。

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// GroupCSVColumn は 1 列の定義。ReadOnly は受理して無視する列、WriteOnly は
// export が常に空を書く列である。`lifecycle_action` を WriteOnly にするのが、
// 無編集の export→apply を全行 unchanged に保つ仕掛けそのものである。
type GroupCSVColumn struct {
	Key       string
	ReadOnly  bool
	WriteOnly bool
}

type GroupCSVSchema struct {
	ordered []GroupCSVColumn
	byKey   map[string]GroupCSVColumn
}

// groupCSVColumns は import 互換列の閉じた集合であり、export の既定の並びでもある。
// `email` と `attributes` は意図的に含まない (wi-350 Out of Scope)。
var groupCSVColumns = []GroupCSVColumn{
	{Key: "id"},
	{Key: "name"},
	{Key: "description"},
	{Key: "membership_type"},
	{Key: "roles"},
	{Key: "dynamic_rule_expression"},
	{Key: "dynamic_rule_enabled"},
	{Key: "lifecycle_action", WriteOnly: true},
	{Key: "created_at", ReadOnly: true},
	{Key: "updated_at", ReadOnly: true},
}

func NewGroupCSVSchema() GroupCSVSchema {
	byKey := make(map[string]GroupCSVColumn, len(groupCSVColumns))
	for _, column := range groupCSVColumns {
		byKey[column.Key] = column
	}
	return GroupCSVSchema{ordered: slices.Clone(groupCSVColumns), byKey: byKey}
}

func (s GroupCSVSchema) Columns() []GroupCSVColumn { return slices.Clone(s.ordered) }

func (s GroupCSVSchema) Column(key string) GroupCSVColumn { return s.byKey[key] }

// Accepts は共有解析器へ渡す、この方言が受理する機械キーの判定である。
func (s GroupCSVSchema) Accepts(key string) bool {
	_, ok := s.byKey[key]
	return ok
}

// ColumnKeys は export の既定列 (import 互換列の全体) を返す。
func (s GroupCSVSchema) ColumnKeys() []string {
	keys := make([]string, 0, len(s.ordered))
	for _, column := range s.ordered {
		keys = append(keys, column.Key)
	}
	return keys
}

// GroupCSVLifecycleAction は upsert を超える行操作。Group には削除予約状態が無いため
// 語彙は `delete` だけの閉じた集合であり、未知の値は既知の値へ丸めず拒否する。
type GroupCSVLifecycleAction string

const (
	GroupCSVLifecycleNone   GroupCSVLifecycleAction = ""
	GroupCSVLifecycleDelete GroupCSVLifecycleAction = "delete"
)

var ErrInvalidGroupCSVCell = errors.New("invalid group CSV cell")

func ParseGroupCSVLifecycleAction(raw string) (GroupCSVLifecycleAction, error) {
	switch GroupCSVLifecycleAction(raw) {
	case GroupCSVLifecycleNone:
		return GroupCSVLifecycleNone, nil
	case GroupCSVLifecycleDelete:
		return GroupCSVLifecycleDelete, nil
	default:
		return GroupCSVLifecycleNone, fmt.Errorf("%w: lifecycle_action", ErrInvalidGroupCSVCell)
	}
}

func ParseGroupCSVMembershipType(raw string) (GroupMembershipType, error) {
	switch GroupMembershipType(raw) {
	case "", GroupMembershipManual:
		return GroupMembershipManual, nil
	case GroupMembershipDynamic:
		return GroupMembershipDynamic, nil
	default:
		return "", fmt.Errorf("%w: membership_type", ErrInvalidGroupCSVCell)
	}
}

// ParseGroupCSVBoolean は真偽値列の字句形。表計算が生む揺れを受け入れず、
// export が書く 2 つの綴りだけを受理する。
func ParseGroupCSVBoolean(raw string) (bool, error) {
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%w: boolean", ErrInvalidGroupCSVCell)
	}
}

// ParseGroupCSVRoles は `|` 区切りの字句形を解く。空セルは空集合であり、
// 空要素は綴り誤りと区別できないため拒否する。
func ParseGroupCSVRoles(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}
	parts := strings.Split(raw, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, fmt.Errorf("%w: roles", ErrInvalidGroupCSVCell)
		}
	}
	return parts, nil
}

func FormatGroupCSVRoles(roles []string) string { return strings.Join(roles, "|") }

type GroupCSVIdentifier struct {
	ID   string
	Name string
}

// GroupCSVIdentifierOf は行から Group の識別子を読み出す。`id` を優先し、
// 無ければテナント内で一意な `name` にたどる。
func GroupCSVIdentifierOf(row idmdomain.CSVRow) (GroupCSVIdentifier, idmdomain.CSVErrorCode) {
	id := GroupCSVIdentifier{ID: row.TrimmedCell("id"), Name: row.TrimmedCell("name")}
	if id.ID == "" && id.Name == "" {
		return id, "missing_identifier"
	}
	return id, ""
}

type GroupImportAction string

const (
	GroupImportCreate    GroupImportAction = "created"
	GroupImportUpdate    GroupImportAction = "updated"
	GroupImportUnchanged GroupImportAction = "unchanged"
	GroupImportDeleted   GroupImportAction = "deleted"
	GroupImportRejected  GroupImportAction = "rejected"
)

// GroupImportRowPlan は 1 行の判定結果。Group 本体と dynamic rule を組で運ぶのは、
// 両者が 1 行 1 トランザクションで確定する 1 つの変更境界だからである。
// DeletedMemberships は削除が巻き込む membership の件数で、プレビューが他の操作と
// 分けて表示するために持つ。
type GroupImportRowPlan struct {
	Row                int
	Action             GroupImportAction
	Identifier         GroupCSVIdentifier
	Before             *Group
	Group              *Group
	Rule               *DynamicGroupRule
	RuleChanged        bool
	DeletedMemberships []string
	Changed            []string
	Error              *idmdomain.CSVError
}

// RejectedGroupImportRow は位置と安定コードだけを持つ拒否行を作る。セル値は
// 決して載せない。
func RejectedGroupImportRow(row int, column string, code idmdomain.CSVErrorCode) GroupImportRowPlan {
	return GroupImportRowPlan{
		Row: row, Action: GroupImportRejected,
		Error: &idmdomain.CSVError{Row: row, Column: column, Code: code},
	}
}
