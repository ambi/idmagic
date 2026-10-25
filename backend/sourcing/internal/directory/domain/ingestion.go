package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

// 一つのバッチと、その識別子の大きさの上限 (REQ-SOURCING-016)。
const (
	MaxBatchItems       = 100
	MaxBatchIDLength    = 128
	MaxExternalIDLength = 256
)

var (
	ErrInvalidBatch = errors.New("directory ingestion: invalid batch")
	ErrInvalidMode  = errors.New("directory ingestion: invalid run mode")
)

type IngestionMode string

const (
	ModeFull        IngestionMode = "full"
	ModeIncremental IngestionMode = "incremental"
)

func ParseIngestionMode(value string) (IngestionMode, error) {
	switch mode := IngestionMode(value); mode {
	case ModeFull, ModeIncremental:
		return mode, nil
	default:
		return "", ErrInvalidMode
	}
}

type IngestionRunStatus string

const (
	RunRunning          IngestionRunStatus = "running"
	RunAwaitingApproval IngestionRunStatus = "awaiting_approval"
	RunApplying         IngestionRunStatus = "applying"
	RunDiscarded        IngestionRunStatus = "discarded"
	RunCompleted        IngestionRunStatus = "completed"
	RunSuperseded       IngestionRunStatus = "superseded"
)

// IngestionRun は Connector が始めた一回の取り込みである。Counts は記録したバッチの結果の和で、
// バッチを記録するときにだけ増える。
type IngestionRun struct {
	Protection  *IngestionProtection
	TenantID    string
	ID          string
	ConnectorID string
	Mode        IngestionMode
	Status      IngestionRunStatus
	StartedAt   time.Time
	CompletedAt *time.Time
	Counts      RunCounts
}

type RunCounts struct {
	Batches   int
	Items     int
	Created   int
	Updated   int
	Unchanged int
	Deleted   int
	Failed    int
	Disabled  int
}

// UserItem は Connector が正規化して送る User の一件である。削除の項目では ExternalID だけを読む。
type UserItem struct {
	ExternalID        string `json:"external_id"`
	Deleted           bool   `json:"deleted"`
	Enabled           bool   `json:"enabled"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Email             string `json:"email"`
	EmployeeNumber    string `json:"employee_number"`
	Department        string `json:"department"`
	// OnPremisesImmutableID は Entra の sourceAnchor であり、User の同名の組み込みの属性に保存する。
	OnPremisesImmutableID string `json:"on_premises_immutable_id"`
}

type ItemOutcome string

const (
	OutcomeCreated   ItemOutcome = "created"
	OutcomeUpdated   ItemOutcome = "updated"
	OutcomeUnchanged ItemOutcome = "unchanged"
	OutcomeDeleted   ItemOutcome = "deleted"
	OutcomeFailed    ItemOutcome = "failed"
)

// 失敗した項目のエラーコード。
const (
	ItemErrorUsernameConflict  = "username_conflict"
	ItemErrorEmailConflict     = "email_conflict"
	ItemErrorAttributeConflict = "attribute_conflict"
	ItemErrorInvalidAttribute  = "invalid_attribute"
	ItemErrorQuotaExceeded     = "quota_exceeded"
	ItemErrorUserDeleted       = "user_deleted"
)

type ItemResult struct {
	ExternalID string      `json:"external_id"`
	Outcome    ItemOutcome `json:"outcome"`
	Error      string      `json:"error,omitempty"`
}

// IngestionBatch は適用し終えたバッチの記録であり、同じバッチの再送に同じ結果を返すために使う。
type IngestionBatch struct {
	TenantID   string
	RunID      string
	BatchID    string
	Digest     string
	Results    []ItemResult
	RecordedAt time.Time
}

// ValidateUserBatch は、どの項目も適用する前にバッチの形を検査する。
func ValidateUserBatch(batchID string, items []UserItem) error {
	if batchID == "" || utf8.RuneCountInString(batchID) > MaxBatchIDLength {
		return fmt.Errorf("%w: batch_id must be 1 to %d characters", ErrInvalidBatch, MaxBatchIDLength)
	}
	if len(items) == 0 || len(items) > MaxBatchItems {
		return fmt.Errorf("%w: a batch carries 1 to %d items", ErrInvalidBatch, MaxBatchItems)
	}
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if item.ExternalID == "" || utf8.RuneCountInString(item.ExternalID) > MaxExternalIDLength {
			return fmt.Errorf("%w: external_id must be 1 to %d characters", ErrInvalidBatch, MaxExternalIDLength)
		}
		if seen[item.ExternalID] {
			return fmt.Errorf("%w: external_id %q appears twice", ErrInvalidBatch, item.ExternalID)
		}
		seen[item.ExternalID] = true
		if !item.Deleted && item.PreferredUsername == "" {
			return fmt.Errorf("%w: preferred_username is required unless the item is deleted", ErrInvalidBatch)
		}
	}
	return nil
}

// BatchDigest は、同じ ID の再送が同じ項目を持つかを比べる値を返す。
// 解析した後の項目から計算するので、JSON の空白とキーの順序に左右されない。
func BatchDigest(items []UserItem) string {
	encoded, err := json.Marshal(items)
	if err != nil {
		panic(err) // 文字列と真偽値だけの構造体は常に符号化できる。
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// Add は記録したバッチの結果を件数へ足す。
func (c RunCounts) Add(results []ItemResult) RunCounts {
	c.Batches++
	c.Items += len(results)
	for _, result := range results {
		switch result.Outcome {
		case OutcomeCreated:
			c.Created++
		case OutcomeUpdated:
			c.Updated++
		case OutcomeUnchanged:
			c.Unchanged++
		case OutcomeDeleted:
			c.Deleted++
		case OutcomeFailed:
			c.Failed++
		}
	}
	return c
}
