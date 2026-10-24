package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// コンピューターの項目の値の長さの上限 (REQ-SOURCING-020)。
const (
	MaxSAMAccountNameLength = 256
	MaxObjectSIDLength      = 184
	MaxDomainLength         = 253
)

// ItemErrorComputerConflict は、別の Connector の有効なコンピューターが同じ SID か主体名を持つ項目の失敗である。
const ItemErrorComputerConflict = "computer_conflict"

// ErrComputerConflict は、保存が有効なコンピューターの SID か主体名の一意性に反したことを表す。
var ErrComputerConflict = errors.New("directory ingestion: another enabled computer holds the sid or principal name")

var objectSIDPattern = regexp.MustCompile(`^S-1-\d+(-\d+)+$`)

// ComputerItem は Connector が正規化して送る AD のコンピューターアカウントの一件である。
type ComputerItem struct {
	ExternalID     string `json:"external_id"`
	Enabled        bool   `json:"enabled"`
	SAMAccountName string `json:"sam_account_name"`
	ObjectSID      string `json:"object_sid"`
	Domain         string `json:"domain"`
	DNSHostName    string `json:"dns_host_name"`
}

// PrincipalKey は Kerberos の主体名 <sAMAccountName>@<realm> を、大文字と小文字を区別せずに比べる鍵にする。
func PrincipalKey(samAccountName, realm string) string {
	return strings.ToLower(samAccountName) + "@" + strings.ToLower(realm)
}

// Computer は Connector と外部の不変 ID の組ごとに保存するコンピューターの写しである。
type Computer struct {
	TenantID    string
	ConnectorID string
	ComputerItem
	LastSeenRunID string
	UpdatedAt     time.Time
}

func (c Computer) PrincipalKey() string { return PrincipalKey(c.SAMAccountName, c.Domain) }

// Holds は、有効な c が item と同じ SID か主体名を持つかを返す。
func (c Computer) Holds(item ComputerItem) bool {
	return c.Enabled && (c.ObjectSID == item.ObjectSID || c.PrincipalKey() == PrincipalKey(item.SAMAccountName, item.Domain))
}

// ValidateComputerBatch は、どの項目も適用する前にバッチの形を検査する。
func ValidateComputerBatch(batchID string, items []ComputerItem) error {
	if err := validateBatchShape(batchID, len(items)); err != nil {
		return err
	}
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if err := validateExternalID(item.ExternalID, seen); err != nil {
			return err
		}
		switch {
		case !strings.HasSuffix(item.SAMAccountName, "$") || utf8.RuneCountInString(item.SAMAccountName) > MaxSAMAccountNameLength:
			return fmt.Errorf("%w: sam_account_name must end with $ and be at most %d characters", ErrInvalidBatch, MaxSAMAccountNameLength)
		case len(item.ObjectSID) > MaxObjectSIDLength || !objectSIDPattern.MatchString(item.ObjectSID):
			return fmt.Errorf("%w: object_sid is not a security identifier", ErrInvalidBatch)
		case item.Domain == "" || utf8.RuneCountInString(item.Domain) > MaxDomainLength:
			return fmt.Errorf("%w: domain must be 1 to %d characters", ErrInvalidBatch, MaxDomainLength)
		case utf8.RuneCountInString(item.DNSHostName) > MaxDomainLength:
			return fmt.Errorf("%w: dns_host_name must be at most %d characters", ErrInvalidBatch, MaxDomainLength)
		}
	}
	return nil
}

// ComputerBatchDigest は、同じ ID の再送が同じ項目を持つかを比べる値を返す。種類を含めるので、
// 同じ ID の User のバッチのダイジェストとは一致しない。
func ComputerBatchDigest(items []ComputerItem) string {
	encoded, err := json.Marshal(items)
	if err != nil {
		panic(err) // 文字列と真偽値だけの構造体は常に符号化できる。
	}
	digest := sha256.Sum256(append([]byte("computers:"), encoded...))
	return hex.EncodeToString(digest[:])
}
