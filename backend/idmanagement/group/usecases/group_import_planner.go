package usecases

// Group CSV の計画器。プレビューと適用が共有する 1 個の決定的な計算であり、
// リポジトリの読み取り以外に効果を持たない。適用はこの計画器を現在状態に対して
// 再び走らせるため、プレビューが暗黙の楽観的ロックの迂回路にならない
// (docs/contexts/identity-management/internals.md)。

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const groupImportPageSize = 500

type GroupImportPlanDeps struct {
	GroupRepo groupports.GroupRepository
	// SchemaRepo は動的規則の式が参照できる User 属性定義を供給する。nil なら
	// 組み込み定義だけを使う。
	SchemaRepo tenantports.TenantUserAttributeSchemaRepository
	// GroupSchemaReader は `custom:<key>` 列になりうる Group 属性定義を供給する。
	// nil のテナントは custom 列を持たない。
	GroupSchemaReader groupports.EffectiveGroupAttributeSchemaReader
	// OwnershipGuard は既存 Group が外部管理かどうかを返す。nil、または失敗は
	// 「外部管理」として fail-closed に扱う。
	OwnershipGuard groupports.GroupSourceOwnershipGuard
	PageSize       int
}

func (d GroupImportPlanDeps) pageSize() int {
	if d.PageSize > 0 {
		return d.PageSize
	}
	return groupImportPageSize
}

// GroupImportPlanSummary は行操作ごとの件数。削除は不可逆で cascade するため、
// 削除行数と巻き込まれる membership 件数を他の操作と分けて持つ。
type GroupImportPlanSummary struct {
	TotalRows          int `json:"total_rows"`
	CreatedRows        int `json:"created_rows"`
	UpdatedRows        int `json:"updated_rows"`
	UnchangedRows      int `json:"unchanged_rows"`
	DeletedRows        int `json:"deleted_rows"`
	DeletedMemberships int `json:"deleted_memberships"`
	RejectedRows       int `json:"rejected_rows"`
}

func (s *GroupImportPlanSummary) Observe(row groupdomain.GroupImportRowPlan) {
	s.TotalRows++
	switch row.Action {
	case groupdomain.GroupImportCreate:
		s.CreatedRows++
	case groupdomain.GroupImportUpdate:
		s.UpdatedRows++
	case groupdomain.GroupImportUnchanged:
		s.UnchangedRows++
	case groupdomain.GroupImportDeleted:
		s.DeletedRows++
		s.DeletedMemberships += len(row.DeletedMemberships)
	case groupdomain.GroupImportRejected:
		s.RejectedRows++
	}
}

// groupImportIndex は 1 回のページ走査で作る上限付きの索引。行ごとにリポジトリを
// 引かないためであり、所有権の判定も同じ走査でまとめて解決する。
type groupImportIndex struct {
	byID                 map[string]*groupdomain.Group
	byName               map[string]*groupdomain.Group
	rules                map[string]*groupdomain.DynamicGroupRule
	members              map[string][]string
	sourceManaged        map[string]bool
	ownershipUnavailable bool
}

func loadGroupImportIndex(ctx context.Context, deps GroupImportPlanDeps, tenantID string) (groupImportIndex, error) {
	index := groupImportIndex{
		byID: map[string]*groupdomain.Group{}, byName: map[string]*groupdomain.Group{},
		rules: map[string]*groupdomain.DynamicGroupRule{}, members: map[string][]string{},
		sourceManaged: map[string]bool{},
	}
	afterName, afterID := "", ""
	for {
		page, err := deps.GroupRepo.ListPage(ctx, tenantID, afterName, afterID, deps.pageSize())
		if err != nil {
			return groupImportIndex{}, err
		}
		if len(page) == 0 {
			break
		}
		ids := make([]string, 0, len(page))
		for _, group := range page {
			if group == nil || group.TenantID != tenantID {
				continue
			}
			index.byID[group.ID] = group
			index.byName[groupNameKey(group.Name)] = group
			ids = append(ids, group.ID)
		}
		if deps.OwnershipGuard == nil {
			index.ownershipUnavailable = true
		} else if managed, err := deps.OwnershipGuard.SourceManagedGroupIDs(ctx, tenantID, ids); err != nil {
			index.ownershipUnavailable = true
		} else {
			for id, value := range managed {
				if value {
					index.sourceManaged[id] = true
				}
			}
		}
		last := page[len(page)-1]
		if last == nil || (last.Name == afterName && last.ID == afterID) {
			return groupImportIndex{}, errors.New("group import pagination did not advance")
		}
		afterName, afterID = last.Name, last.ID
		if len(page) < deps.pageSize() {
			break
		}
	}
	rules, err := deps.GroupRepo.ListDynamicRules(ctx, tenantID)
	if err != nil {
		return groupImportIndex{}, err
	}
	for _, rule := range rules {
		if rule != nil {
			index.rules[rule.GroupID] = rule
		}
	}
	return index, nil
}

// groupNameKey は name の一意性判定のキー。Group の名前一意性は大文字小文字を
// 区別しない (ensureGroupNameAvailable) ため、CSV の解決も同じ規則に従う。
func groupNameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// PlanGroupImport は 1 個の決定的な計画器で行を流す。emit は行計画とエラーを
// 逐次書き出せる。返す要約は行数に依らず上限付きである。
func PlanGroupImport(
	ctx context.Context,
	deps GroupImportPlanDeps,
	input io.Reader,
	policy idmdomain.CSVTransferPolicy,
	emit func(groupdomain.GroupImportRowPlan) error,
) (GroupImportPlanSummary, error) {
	var summary GroupImportPlanSummary
	if deps.GroupRepo == nil {
		return summary, errors.New("group import planner dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	schema, err := groupCSVSchemaFor(ctx, deps.GroupSchemaReader, tenantID)
	if err != nil {
		return summary, err
	}
	reader, err := idmdomain.NewCSVReader(input, schema.Accepts, policy)
	if err != nil {
		return summary, err
	}
	defs, err := effectiveGroupImportRuleDefs(ctx, deps, tenantID)
	if err != nil {
		return summary, err
	}
	index, err := loadGroupImportIndex(ctx, deps, tenantID)
	if err != nil {
		return summary, err
	}
	seenIDs := map[string]struct{}{}
	seenNames := map[string]struct{}{}
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return summary, nil
		}
		if err != nil {
			return summary, err
		}
		var planned groupdomain.GroupImportRowPlan
		if record.Error != nil {
			planned = groupdomain.RejectedGroupImportRow(record.Error.Row, record.Error.Column, record.Error.Code)
		} else {
			planned = planGroupImportRow(ctx, deps, *record.Row, schema, index, defs, tenantID, seenIDs, seenNames)
		}
		summary.Observe(planned)
		if emit != nil {
			if err := emit(planned); err != nil {
				return summary, err
			}
		}
	}
}

func effectiveGroupImportRuleDefs(ctx context.Context, deps GroupImportPlanDeps, tenantID string) ([]userdomain.UserAttributeDef, error) {
	defs := userdomain.BuiltinUserAttributeDefs()
	if deps.SchemaRepo == nil {
		return defs, nil
	}
	schema, err := deps.SchemaRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		defs = schema.EffectiveDefs()
	}
	return defs, nil
}

func planGroupImportRow(
	ctx context.Context,
	deps GroupImportPlanDeps,
	row idmdomain.CSVRow,
	schema groupdomain.GroupCSVSchema,
	index groupImportIndex,
	defs []userdomain.UserAttributeDef,
	tenantID string,
	seenIDs, seenNames map[string]struct{},
) groupdomain.GroupImportRowPlan {
	identifier, code := groupdomain.GroupCSVIdentifierOf(row)
	if code != "" {
		return groupdomain.RejectedGroupImportRow(row.Number, "", code)
	}
	if identifier.ID != "" {
		if _, duplicate := seenIDs[identifier.ID]; duplicate {
			return groupdomain.RejectedGroupImportRow(row.Number, "id", "duplicate_target")
		}
		seenIDs[identifier.ID] = struct{}{}
	}
	if identifier.Name != "" {
		key := groupNameKey(identifier.Name)
		if _, duplicate := seenNames[key]; duplicate {
			return groupdomain.RejectedGroupImportRow(row.Number, "name", "duplicate_name")
		}
		seenNames[key] = struct{}{}
	}

	action, err := groupdomain.ParseGroupCSVLifecycleAction(row.TrimmedCell("lifecycle_action"))
	if err != nil {
		return groupdomain.RejectedGroupImportRow(row.Number, "lifecycle_action", "invalid_lifecycle_action")
	}

	existing, resolveCode := resolveGroupImportTarget(identifier, index)
	if resolveCode != "" {
		return groupdomain.RejectedGroupImportRow(row.Number, "", resolveCode)
	}
	// 所有権の判定不能と外部管理は、更新でも削除でも同じく fail-closed に拒否する。
	if existing != nil && (index.ownershipUnavailable || index.sourceManaged[existing.ID]) {
		return groupdomain.RejectedGroupImportRow(row.Number, "", "source_managed")
	}
	// 対象を解決できない削除行は作成に落ちない。綴り誤りや古いファイルを
	// unchanged として通すと、破壊的操作の直前にプレビューが与える情報が失われる。
	if action == groupdomain.GroupCSVLifecycleDelete && existing == nil {
		return groupdomain.RejectedGroupImportRow(row.Number, "lifecycle_action", "target_not_found")
	}

	candidate, column, cellCode := groupImportCandidate(existing, row, schema, identifier, tenantID)
	if cellCode != "" {
		return groupdomain.RejectedGroupImportRow(row.Number, column, cellCode)
	}
	rule, ruleChanged, ruleCode := planGroupImportRule(row, existing, candidate, index, defs)
	if ruleCode != "" {
		return groupdomain.RejectedGroupImportRow(row.Number, "dynamic_rule_expression", ruleCode)
	}

	if action == groupdomain.GroupCSVLifecycleDelete {
		// 削除と更新の意図は 1 行に同居できない。Group は削除で消えるため、直前の
		// 更新を監査に残すと存在しなかった状態の記録になる。
		if len(groupImportChangedFields(*existing, *candidate)) > 0 || ruleChanged {
			return groupdomain.RejectedGroupImportRow(row.Number, "lifecycle_action", "conflicting_lifecycle_action")
		}
		members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, existing.ID)
		if err != nil {
			return groupdomain.RejectedGroupImportRow(row.Number, "", "apply_failed")
		}
		userIDs := make([]string, 0, len(members))
		for _, member := range members {
			if member != nil {
				userIDs = append(userIDs, member.UserID)
			}
		}
		return groupdomain.GroupImportRowPlan{
			Row: row.Number, Action: groupdomain.GroupImportDeleted, Identifier: identifier,
			Before: existing, DeletedMemberships: userIDs,
		}
	}

	if existing == nil {
		return groupdomain.GroupImportRowPlan{
			Row: row.Number, Action: groupdomain.GroupImportCreate, Identifier: identifier,
			Group: candidate, Rule: rule, RuleChanged: rule != nil,
		}
	}
	changed := groupImportChangedFields(*existing, *candidate)
	planned := groupdomain.GroupImportRowPlan{
		Row: row.Number, Action: groupdomain.GroupImportUpdate, Identifier: identifier,
		Before: existing, Group: candidate, Rule: rule, RuleChanged: ruleChanged, Changed: changed,
	}
	if len(changed) == 0 && !ruleChanged {
		planned.Action = groupdomain.GroupImportUnchanged
	}
	return planned
}

// resolveGroupImportTarget は `id` を優先し、無ければ `name` で解決する。`id` が
// 既存を指し、同じ行の `name` が別の既存を指す場合は改名にならないため拒否する。
func resolveGroupImportTarget(identifier groupdomain.GroupCSVIdentifier, index groupImportIndex) (*groupdomain.Group, idmdomain.CSVErrorCode) {
	if identifier.ID != "" {
		group := index.byID[identifier.ID]
		if group == nil {
			return nil, "target_not_found"
		}
		if identifier.Name != "" {
			if named := index.byName[groupNameKey(identifier.Name)]; named != nil && named.ID != group.ID {
				return nil, "identifier_mismatch"
			}
		}
		return group, ""
	}
	return index.byName[groupNameKey(identifier.Name)], ""
}

// groupImportCandidate は行のセルを適用した最終状態を作る。列が無ければ維持し、
// 存在する空セルは optional 項目を消す。
func groupImportCandidate(
	existing *groupdomain.Group,
	row idmdomain.CSVRow,
	schema groupdomain.GroupCSVSchema,
	identifier groupdomain.GroupCSVIdentifier,
	tenantID string,
) (*groupdomain.Group, string, idmdomain.CSVErrorCode) {
	var candidate groupdomain.Group
	if existing != nil {
		candidate = *existing
		candidate.Roles = slices.Clone(existing.Roles)
		candidate.Attributes = cloneGroupImportAttributes(existing.Attributes)
	} else {
		candidate = groupdomain.Group{
			ID: "preview", TenantID: tenantID, Name: identifier.Name, Roles: []string{},
			MembershipType: groupdomain.GroupMembershipManual,
		}
	}
	if candidate.Attributes == nil {
		candidate.Attributes = map[string]userdomain.AttributeValue{}
	}

	if cell, ok := row.Cell("name"); ok {
		name := strings.TrimSpace(cell.Raw)
		if name == "" {
			return nil, "name", "required"
		}
		candidate.Name = name
	}
	if cell, ok := row.Cell("description"); ok {
		candidate.Description = idmusecases.NormalizeDescription(&cell.Raw)
	}
	if cell, ok := row.Cell("email"); ok {
		email, _, err := groupdomain.ParseGroupCSVEmailCell(cell.Raw)
		if err != nil {
			return nil, "email", "invalid_email"
		}
		candidate.Email = email
	}
	if cell, ok := row.Cell("roles"); ok {
		roles, err := groupdomain.ParseGroupCSVRoles(cell.Raw)
		if err != nil {
			return nil, "roles", "invalid_roles"
		}
		normalized, err := idmusecases.NormalizeRoles(roles)
		if err != nil {
			return nil, "roles", "invalid_roles"
		}
		candidate.Roles = normalized
	}
	if cell, ok := row.Cell("membership_type"); ok {
		membership, err := groupdomain.ParseGroupCSVMembershipType(strings.TrimSpace(cell.Raw))
		if err != nil {
			return nil, "membership_type", "invalid_membership_type"
		}
		// 作成時にだけ選べる。既存 Group では暗黙変換せず、現在値と異なる指定を拒否する。
		if existing == nil {
			candidate.MembershipType = membership
		} else if membership != existing.MembershipType.Effective() {
			return nil, "membership_type", "immutable_membership_type"
		}
	}
	if existing == nil {
		candidate.MembershipType = candidate.MembershipType.Effective()
	}
	// テナント定義の属性は、列の定義が固定した型の正規字句形で読む。空セルは
	// その属性を消す意味であり、`required` な属性では拒否される。
	attributeDefs := make([]groupdomain.GroupAttributeDef, 0)
	for _, column := range schema.Columns() {
		if column.Attribute == nil {
			continue
		}
		attributeDefs = append(attributeDefs, *column.Attribute)
		cell, present := row.Cell(column.Key)
		if !present {
			continue
		}
		value, shouldClear, err := groupdomain.ParseGroupCSVAttributeCell(cell.Raw, *column.Attribute)
		if err != nil {
			return nil, column.Key, "invalid_attribute"
		}
		if shouldClear {
			delete(candidate.Attributes, column.Attribute.Key)
		} else {
			candidate.Attributes[column.Attribute.Key] = value
		}
	}
	if err := groupdomain.ValidateGroupAttributes(candidate.Attributes, attributeDefs); err != nil {
		return nil, "", "invalid_attribute"
	}
	if err := groupImportValidateCandidate(candidate); err != nil {
		return nil, "", "invalid_group"
	}
	return &candidate, "", ""
}

// groupImportValidateCandidate は Aggregate の不変条件を、まだ生成していない
// 識別子とタイムスタンプを除いて検査する。
func groupImportValidateCandidate(candidate groupdomain.Group) error {
	probe := candidate
	if probe.ID == "" {
		probe.ID = "preview"
	}
	if probe.CreatedAt.IsZero() {
		probe.CreatedAt = groupImportValidationStamp
	}
	if probe.UpdatedAt.IsZero() {
		probe.UpdatedAt = groupImportValidationStamp
	}
	return probe.Validate()
}

// planGroupImportRule は 2 つの列を、維持された相方と組み合わせた最終状態として
// 検証する。列ごとに見ると、どちらの列単体では不正に見えない組み合わせを通す。
func planGroupImportRule(
	row idmdomain.CSVRow,
	existing *groupdomain.Group,
	candidate *groupdomain.Group,
	index groupImportIndex,
	defs []userdomain.UserAttributeDef,
) (*groupdomain.DynamicGroupRule, bool, idmdomain.CSVErrorCode) {
	expressionCell, hasExpression := row.Cell("dynamic_rule_expression")
	enabledCell, hasEnabled := row.Cell("dynamic_rule_enabled")
	if !hasExpression && !hasEnabled {
		return nil, false, ""
	}
	var current *groupdomain.DynamicGroupRule
	if existing != nil {
		current = index.rules[existing.ID]
	}

	expression := ""
	if current != nil {
		expression = current.Expression
	}
	if hasExpression {
		expression = strings.TrimSpace(expressionCell.Raw)
	}
	enabled := false
	if current != nil {
		enabled = current.Enabled
	}
	if hasEnabled {
		raw := strings.TrimSpace(enabledCell.Raw)
		if raw != "" {
			parsed, err := groupdomain.ParseGroupCSVBoolean(raw)
			if err != nil {
				return nil, false, "invalid_dynamic_rule"
			}
			enabled = parsed
		}
	}

	if expression == "" {
		// 規則を持つ Group の式を空にする経路は持たない。空欄を「規則なし」と読むと、
		// 列を落としたファイルと空欄のファイルの意味が入れ替わる。
		if current != nil || enabled {
			return nil, false, "invalid_dynamic_rule"
		}
		return nil, false, ""
	}
	if candidate.MembershipType.Effective() != groupdomain.GroupMembershipDynamic {
		return nil, false, "invalid_dynamic_rule"
	}
	compiled, err := groupdomain.CompileDynamicGroupRule(expression, defs)
	if err != nil {
		return nil, false, "invalid_dynamic_rule"
	}

	version := int64(1)
	if current != nil {
		version = current.Version
	}
	changed := current == nil || current.Expression != expression || current.Enabled != enabled
	if changed && current != nil {
		version = current.Version + 1
	}
	rule := &groupdomain.DynamicGroupRule{
		GroupID: candidate.ID, TenantID: candidate.TenantID, Expression: expression, Enabled: enabled,
		Version: version, ReferencedAttributes: compiled.References(),
	}
	if current != nil {
		rule.CreatedAt = current.CreatedAt
	}
	return rule, changed, ""
}

// groupImportAttributesEqual は属性なしを 1 つの意味として扱う。属性を持たない
// Group は対応表が nil のまま保存されており、候補は常に空の対応表から始まる。
// 両者を別物と見ると、無編集の往復が全行 updated になってしまう。
func groupImportAttributesEqual(before, after map[string]userdomain.AttributeValue) bool {
	if len(before) == 0 && len(after) == 0 {
		return true
	}
	return reflect.DeepEqual(before, after)
}

// cloneGroupImportAttributes は StringArray まで複製する。浅い複製だと候補が
// 保存済み集約とスライスを共有し、計画が状態を書き換えうる。
func cloneGroupImportAttributes(values map[string]userdomain.AttributeValue) map[string]userdomain.AttributeValue {
	out := make(map[string]userdomain.AttributeValue, len(values))
	for key, value := range values {
		value.StringArray = slices.Clone(value.StringArray)
		out[key] = value
	}
	return out
}

func groupImportChangedFields(before, after groupdomain.Group) []string {
	changed := make([]string, 0, 4)
	if before.Name != after.Name {
		changed = append(changed, "name")
	}
	if !idmusecases.EqualOptionalString(before.Description, after.Description) {
		changed = append(changed, "description")
	}
	if !idmusecases.EqualOptionalString(before.Email, after.Email) {
		changed = append(changed, "email")
	}
	if !groupImportAttributesEqual(before.Attributes, after.Attributes) {
		changed = append(changed, "attributes")
	}
	if !slices.Equal(before.Roles, after.Roles) {
		changed = append(changed, "roles")
	}
	if before.MembershipType.Effective() != after.MembershipType.Effective() {
		changed = append(changed, "membership_type")
	}
	return changed
}

// groupImportValidationStamp は、まだ生成していないタイムスタンプの代わりに
// 検査だけへ渡す固定値。計画は時刻を持たず、実際の値は適用時に注入する。
var groupImportValidationStamp = time.Unix(1, 0).UTC()
