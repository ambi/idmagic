package domain

import "time"

// AuthorizationModelPublished は管理者が認可モデルの新しい版を published したことを表す。
type AuthorizationModelPublished struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	ModelID           string    `json:"modelId"`
	Version           int       `json:"version"`
	ResourceTypeCount int       `json:"resourceTypeCount"`
}

func (e *AuthorizationModelPublished) EventType() string     { return "AuthorizationModelPublished" }
func (e *AuthorizationModelPublished) OccurredAt() time.Time { return e.At }

// RelationTupleWritten は関係タプルを書き込んだことを表す。個々のタプルは監査へ
// 複製せず、件数と整合トークンだけを残す。
type RelationTupleWritten struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	WrittenCount int       `json:"writtenCount"`
	Consistency  string    `json:"consistency"`
}

func (e *RelationTupleWritten) EventType() string     { return "RelationTupleWritten" }
func (e *RelationTupleWritten) OccurredAt() time.Time { return e.At }

// RelationTupleDeleted は関係タプルを削除したことを表す。オブジェクト単位の
// 削除の波及もこのイベントで表す。
type RelationTupleDeleted struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	DeletedCount int       `json:"deletedCount"`
	Consistency  string    `json:"consistency"`
}

func (e *RelationTupleDeleted) EventType() string     { return "RelationTupleDeleted" }
func (e *RelationTupleDeleted) OccurredAt() time.Time { return e.At }

// FgaCheckEvaluated は CheckAccess が判定を下したことを表す。リソース識別子は
// ダイジェストにし、主体識別子とタプルの内容は残さない。
type FgaCheckEvaluated struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	ResourceType     string    `json:"resourceType"`
	ResourceIDDigest string    `json:"resourceIdDigest"`
	Relation         string    `json:"relation"`
	Permitted        bool      `json:"permitted"`
	ModelVersion     int       `json:"modelVersion"`
	RelationPath     []string  `json:"relationPath,omitempty"`
	Reasons          []string  `json:"reasons,omitempty"`
	ActorChainDepth  int       `json:"actorChainDepth"`
}

func (e *FgaCheckEvaluated) EventType() string     { return "FgaCheckEvaluated" }
func (e *FgaCheckEvaluated) OccurredAt() time.Time { return e.At }

// FgaResourcesEnumerated は ListAccessibleResources が走査を終えたことを表す。
// 1 件ごとの判定は監査へ展開しない。列挙 1 回で候補数だけのイベントが並ぶと、
// 本当に見るべき単発の判定が埋もれるからである。
type FgaResourcesEnumerated struct {
	At              time.Time `json:"-"`
	TenantID        string    `json:"tenantId"`
	ResourceType    string    `json:"resourceType"`
	Relation        string    `json:"relation"`
	CandidateCount  int       `json:"candidateCount"`
	PermittedCount  int       `json:"permittedCount"`
	Truncated       bool      `json:"truncated"`
	ModelVersion    int       `json:"modelVersion"`
	ActorChainDepth int       `json:"actorChainDepth"`
}

func (e *FgaResourcesEnumerated) EventType() string     { return "FgaResourcesEnumerated" }
func (e *FgaResourcesEnumerated) OccurredAt() time.Time { return e.At }
