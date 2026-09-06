package domain

// メンバーシップ CSV の方言。転送ポリシー、解析器、直列化器、可逆なセル変換は
// idmdomain の種別非依存な CSV 基盤が持ち、ここにはメンバーシップ固有の列の語彙、
// 閉じた状態語彙、識別子、行計画の型だけを置く
// (docs/contexts/identity-management/internals.md)。

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// GroupMembershipCSVColumnMode は 1 列の役割。行が何をしたいかを表せるのは
// Intent の列だけであり、それが `membership_state` 1 つに限られていることが、
// この方言の安全性の骨格である。
type GroupMembershipCSVColumnMode string

const (
	// GroupMembershipCSVVerification は行が対象 Group の行かを確かめるだけの列。
	// 対象を決めるのは URL であって、この列ではない。
	GroupMembershipCSVVerification GroupMembershipCSVColumnMode = "verification"
	// GroupMembershipCSVIdentity は対象 User を解決する列。
	GroupMembershipCSVIdentity GroupMembershipCSVColumnMode = "identity"
	// GroupMembershipCSVIntent は望む所属状態を表す唯一の列。
	GroupMembershipCSVIntent GroupMembershipCSVColumnMode = "intent"
	// GroupMembershipCSVReadOnly は受理して無視する列。
	GroupMembershipCSVReadOnly GroupMembershipCSVColumnMode = "read_only"
)

type GroupMembershipCSVColumn struct {
	Key  string
	Mode GroupMembershipCSVColumnMode
}

type GroupMembershipCSVSchema struct {
	ordered []GroupMembershipCSVColumn
	byKey   map[string]GroupMembershipCSVColumn
}

// groupMembershipCSVColumns は import 互換列の閉じた集合であり、export の既定の
// 並びでもある。テナント定義の列は持たない。メンバーシップは Group と User を
// 結ぶ関係そのものであって、属性を載せる入れ物ではないからである。
var groupMembershipCSVColumns = []GroupMembershipCSVColumn{
	{Key: "group_id", Mode: GroupMembershipCSVVerification},
	{Key: "group_name", Mode: GroupMembershipCSVVerification},
	{Key: "user_id", Mode: GroupMembershipCSVIdentity},
	{Key: "preferred_username", Mode: GroupMembershipCSVIdentity},
	{Key: "membership_state", Mode: GroupMembershipCSVIntent},
	{Key: "source", Mode: GroupMembershipCSVReadOnly},
	{Key: "created_at", Mode: GroupMembershipCSVReadOnly},
}

func NewGroupMembershipCSVSchema() GroupMembershipCSVSchema {
	byKey := make(map[string]GroupMembershipCSVColumn, len(groupMembershipCSVColumns))
	for _, column := range groupMembershipCSVColumns {
		byKey[column.Key] = column
	}
	return GroupMembershipCSVSchema{ordered: slices.Clone(groupMembershipCSVColumns), byKey: byKey}
}

func (s GroupMembershipCSVSchema) Columns() []GroupMembershipCSVColumn {
	return slices.Clone(s.ordered)
}

func (s GroupMembershipCSVSchema) Column(key string) GroupMembershipCSVColumn { return s.byKey[key] }

// Accepts は共有解析器へ渡す、この方言が受理する機械キーの判定である。
func (s GroupMembershipCSVSchema) Accepts(key string) bool {
	_, ok := s.byKey[key]
	return ok
}

// ColumnKeys は export の既定列 (import 互換列の全体) を返す。
func (s GroupMembershipCSVSchema) ColumnKeys() []string {
	keys := make([]string, 0, len(s.ordered))
	for _, column := range s.ordered {
		keys = append(keys, column.Key)
	}
	return keys
}

// MissingRequiredColumn は、ヘッダーに欠けている必須列を返す。必須なのは意図を
// 表す列だけである。他の列を欠いたファイルは「その項目について何も言っていない」
// と読めるが、意図の列を欠いたファイルは何ひとつ言えていない。それを変更 0 件の
// プレビューとして受理すると、管理者は編集が読まれたうえで差分が無かったと
// 受け取ってしまう。
func (s GroupMembershipCSVSchema) MissingRequiredColumn(header []string) string {
	for _, column := range s.ordered {
		if column.Mode != GroupMembershipCSVIntent {
			continue
		}
		if !slices.Contains(header, column.Key) {
			return column.Key
		}
	}
	return ""
}

// GroupMembershipCSVState は行が望む所属状態。追加と解除のどちらでもない値へ
// 丸めないため、語彙は閉じており空セルも受理しない。
type GroupMembershipCSVState string

const (
	GroupMembershipStatePresent GroupMembershipCSVState = "present"
	GroupMembershipStateAbsent  GroupMembershipCSVState = "absent"
)

var ErrInvalidGroupMembershipCSVCell = errors.New("invalid group membership CSV cell")

func ParseGroupMembershipCSVState(raw string) (GroupMembershipCSVState, error) {
	switch GroupMembershipCSVState(raw) {
	case GroupMembershipStatePresent:
		return GroupMembershipStatePresent, nil
	case GroupMembershipStateAbsent:
		return GroupMembershipStateAbsent, nil
	default:
		return "", fmt.Errorf("%w: membership_state", ErrInvalidGroupMembershipCSVCell)
	}
}

type GroupMembershipCSVIdentifier struct {
	UserID            string
	PreferredUsername string
}

// GroupMembershipCSVIdentifierOf は行から User の識別子を読み出す。`user_id` を
// 優先し、無ければテナント内で一意な `preferred_username` にたどる。
func GroupMembershipCSVIdentifierOf(row idmdomain.CSVRow) (GroupMembershipCSVIdentifier, idmdomain.CSVErrorCode) {
	identifier := GroupMembershipCSVIdentifier{
		UserID:            row.TrimmedCell("user_id"),
		PreferredUsername: row.TrimmedCell("preferred_username"),
	}
	if identifier.UserID == "" && identifier.PreferredUsername == "" {
		return identifier, "missing_identifier"
	}
	return identifier, ""
}

// GroupMembershipNameKey は `group_name` の照合キー。Group の名前一意性が
// 大文字小文字を区別しないため、照合も同じ規則に従う。
func GroupMembershipNameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

type GroupMembershipImportAction string

const (
	GroupMembershipImportAdded     GroupMembershipImportAction = "added"
	GroupMembershipImportRemoved   GroupMembershipImportAction = "removed"
	GroupMembershipImportUnchanged GroupMembershipImportAction = "unchanged"
	GroupMembershipImportRejected  GroupMembershipImportAction = "rejected"
)

// GroupMembershipImportRowPlan は 1 行の判定結果。運ぶのは解決済みの User ID と
// 行操作だけである。メンバーシップは Group と User の ID 以外に状態を持たないため、
// Group 側の計画のように変更前後の Aggregate を抱える必要がない。
type GroupMembershipImportRowPlan struct {
	Row        int
	Action     GroupMembershipImportAction
	Identifier GroupMembershipCSVIdentifier
	UserID     string
	Error      *idmdomain.CSVError
}

// RejectedGroupMembershipImportRow は位置と安定コードだけを持つ拒否行を作る。
// セル値も解決した対象も載せない。
func RejectedGroupMembershipImportRow(row int, column string, code idmdomain.CSVErrorCode) GroupMembershipImportRowPlan {
	return GroupMembershipImportRowPlan{
		Row: row, Action: GroupMembershipImportRejected,
		Error: &idmdomain.CSVError{Row: row, Column: column, Code: code},
	}
}
