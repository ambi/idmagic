package domain

import "encoding/json"

// UnmappedTool は宣言のないツールの指標のラベルである。呼び出し元が決める名前を系列にしない。
const UnmappedTool = "unmapped"

const (
	unmappedResourceType = "mcp_tool"
	unmappedRelation     = "unmapped"
	maxEvaluatedToolName = 128
	// maxAttributesBytes は評価が受け付ける attributes の上限である。超える値は評価が 400 で拒否する。
	maxAttributesBytes = 16 << 10
)

// OperationRule は設定が宣言する一つの操作である。ResourceID と ResourceIDArgument はちょうど一方を持つ。
type OperationRule struct {
	ResourceType       string
	Relation           string
	ResourceID         string
	ResourceIDArgument string
	// AttributeArguments は評価の attributes として渡す arguments の直下のキーである。
	AttributeArguments []string
}

// Mapping はツールと方法から操作への対応である。
type Mapping struct {
	Catalog bool
	Methods map[string]OperationRule
	Tools   map[string]OperationRule
}

// Operation は評価へ渡す組である。
type Operation struct {
	ResourceType string
	Relation     string
	ResourceID   string
}

// EvaluationKind は要求の扱いである。
type EvaluationKind int

const (
	// EvaluateAndForward は評価が許可したときだけ転送する。
	EvaluateAndForward EvaluationKind = iota
	// EvaluateForAuditOnly は試行者の記録のために評価し、結果によらず転送しない。
	EvaluateForAuditOnly
	// DenyWithoutEvaluation は評価せず 403 で拒否する。
	DenyWithoutEvaluation
	// MethodNotFound は宣言のない方法である。
	MethodNotFound
)

// Evaluation は Decide の結果である。
type Evaluation struct {
	Kind      EvaluationKind
	Operation Operation
	// Attributes は評価へ渡す attributes の JSON オブジェクトである。空なら送らない。
	Attributes string
	// ToolLabel は指標のラベルである。
	ToolLabel string
	// Reason は評価せずに拒否した理由である。
	Reason string
}

// Decide は解釈した要求から評価する操作を選ぶ (REQ-MCPGATEWAY-003)。
func Decide(message Message, mapping Mapping) Evaluation {
	if message.Method != MethodToolsCall {
		rule, ok := mapping.Methods[message.Method]
		if !ok {
			if mapping.Catalog && (message.Method == "server/discover" || message.Method == "initialize" || message.Method == "notifications/initialized" || message.Method == "ping" || message.Method == "tools/list") {
				return Evaluation{Kind: EvaluateAndForward, Operation: Operation{ResourceType: "idmagic_mcp_method", Relation: "access", ResourceID: message.Method}}
			}
			return Evaluation{Kind: MethodNotFound, Reason: "method_not_mapped"}
		}
		return Evaluation{Kind: EvaluateAndForward, Operation: rule.operation(rule.ResourceID)}
	}
	rule, ok := mapping.Tools[message.ToolName]
	if !ok {
		if len([]rune(message.ToolName)) > maxEvaluatedToolName {
			return Evaluation{Kind: DenyWithoutEvaluation, ToolLabel: UnmappedTool, Reason: "tool_name_too_long"}
		}
		if mapping.Catalog {
			return Evaluation{Kind: EvaluateAndForward, ToolLabel: UnmappedTool, Operation: Operation{ResourceType: "idmagic_mcp_operation", Relation: "invoke", ResourceID: message.ToolName}}
		}
		return Evaluation{
			Kind:      EvaluateForAuditOnly,
			ToolLabel: UnmappedTool,
			Operation: Operation{ResourceType: unmappedResourceType, Relation: unmappedRelation, ResourceID: message.ToolName},
		}
	}
	resourceID := rule.ResourceID
	if rule.ResourceIDArgument != "" {
		var argument string
		if err := json.Unmarshal(message.Arguments[rule.ResourceIDArgument], &argument); err != nil || argument == "" {
			return Evaluation{Kind: DenyWithoutEvaluation, ToolLabel: message.ToolName, Reason: "resource_id_unresolved"}
		}
		resourceID = argument
	}
	attributes, ok := rule.attributes(message.Arguments)
	if !ok {
		return Evaluation{Kind: DenyWithoutEvaluation, ToolLabel: message.ToolName, Reason: "attributes_too_large"}
	}
	return Evaluation{Kind: EvaluateAndForward, ToolLabel: message.ToolName, Operation: rule.operation(resourceID), Attributes: attributes}
}

// attributes は名指したキーのうち引数にあるものだけを、値を変えずに JSON オブジェクトにする。
// 名指しがなければ空を返し、上限を超えれば偽を返す。
func (r OperationRule) attributes(arguments map[string]json.RawMessage) (string, bool) {
	if len(r.AttributeArguments) == 0 {
		return "", true
	}
	selected := map[string]json.RawMessage{}
	for _, key := range r.AttributeArguments {
		if value, ok := arguments[key]; ok {
			selected[key] = value
		}
	}
	// Inspect が検証した JSON の値だけを持つので、直列化は失敗しない。
	raw, err := json.Marshal(selected)
	if err != nil || len(raw) > maxAttributesBytes {
		return "", false
	}
	return string(raw), true
}

func (r OperationRule) operation(resourceID string) Operation {
	return Operation{ResourceType: r.ResourceType, Relation: r.Relation, ResourceID: resourceID}
}
