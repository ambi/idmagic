// Package domain は、Directory Connector が LDAP をどこまで読んだかと、エントリを User の項目へ
// 写す規則を、接続を持たない計算として持つ。
package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Dialect は読み取るディレクトリの種類である。不変 ID、無効の表し方、差分の方式が変わる。
type Dialect string

const (
	ActiveDirectory Dialect = "active_directory"
	OpenLDAP        Dialect = "openldap"
)

func ParseDialect(value string) (Dialect, error) {
	switch dialect := Dialect(value); dialect {
	case ActiveDirectory, OpenLDAP:
		return dialect, nil
	default:
		return "", fmt.Errorf("unknown ldap dialect %q", value)
	}
}

func DefaultUserFilter(dialect Dialect) string {
	if dialect == ActiveDirectory {
		return "(&(objectCategory=person)(objectClass=user))"
	}
	return "(objectClass=inetOrgPerson)"
}

// Scope は検索の範囲である。変わると前回の位置と比べられないので、完全な再同期にする。
type Scope struct {
	BaseDNs []string `json:"base_dns"`
	Filter  string   `json:"filter"`
}

func (s Scope) Equal(other Scope) bool {
	return s.Filter == other.Filter && slices.Equal(s.BaseDNs, other.BaseDNs)
}

// ValidateBaseDNs は、起点が 1 件以上あり、どの起点もほかの起点と同じでもその配下でもないことを確かめる
// (REQ-DIRECTORYCONNECTOR-002)。識別名は大文字と小文字、RDN の区切りの後の空白を区別せずに比べる。
func ValidateBaseDNs(baseDNs []string) error {
	if len(baseDNs) == 0 {
		return errors.New("ldap.base_dns needs at least one base dn")
	}
	normalized := make([]string, len(baseDNs))
	for i, dn := range baseDNs {
		parts := strings.Split(dn, ",")
		for j := range parts {
			parts[j] = strings.ToLower(strings.TrimSpace(parts[j]))
		}
		normalized[i] = strings.Join(parts, ",")
		if normalized[i] == "" {
			return errors.New("ldap.base_dns has an empty base dn")
		}
	}
	for i, dn := range normalized {
		for j, other := range normalized {
			if i != j && (dn == other || strings.HasSuffix(dn, ","+other)) {
				return fmt.Errorf("ldap.base_dns: %q overlaps %q", baseDNs[i], baseDNs[j])
			}
		}
	}
	return nil
}

// RootInfo は検索の前に RootDSE から読む値である。AD だけが値を持つ。
type RootInfo struct {
	Server     string
	HighestUSN int64
}

// Cursor は最後に完了した実行の位置である (SourceCursor)。
type Cursor struct {
	Dialect       Dialect   `json:"dialect"`
	Scope         Scope     `json:"scope"`
	Computers     *Scope    `json:"computers,omitempty"`
	Server        string    `json:"server,omitempty"`
	USN           int64     `json:"usn,omitempty"`
	ModifiedSince time.Time `json:"modified_since,omitzero"`
	LastFullAt    time.Time `json:"last_full_at"`
}

// Plan は一回の実行の種類、検索のフィルター、完了したときに保存する位置である。ComputerFilter は
// コンピューターを読まない設定では空である。
type Plan struct {
	Mode           Mode
	Filter         string
	ComputerFilter string
	Next           Cursor
}

// modifiedSinceOverlap は、OpenLDAP の差分で検索の最中の変更と時計のずれを重ねて読む幅である。
const modifiedSinceOverlap = 5 * time.Minute

// PlanRun は前回の位置から実行の種類を選ぶ (REQ-DIRECTORYCONNECTOR-003)。root は検索の前に読んだ値で、
// startedAt は検索を始める時刻である。computers はコンピューターの検索の範囲で、読まない設定では nil である
// (REQ-DIRECTORYCONNECTOR-007)。
func PlanRun(dialect Dialect, scope Scope, computers *Scope, prev *Cursor, root RootInfo, startedAt time.Time, fullInterval time.Duration) Plan {
	next := Cursor{Dialect: dialect, Scope: scope, Computers: computers, LastFullAt: startedAt}
	switch dialect {
	case ActiveDirectory:
		next.Server, next.USN = root.Server, root.HighestUSN
	case OpenLDAP:
		next.ModifiedSince = startedAt.Add(-modifiedSinceOverlap).UTC()
	}
	if needsFullSync(dialect, scope, computers, prev, root, startedAt, fullInterval) {
		plan := Plan{Mode: ModeFull, Filter: scope.Filter, Next: next}
		if computers != nil {
			plan.ComputerFilter = computers.Filter
		}
		return plan
	}
	next.LastFullAt = prev.LastFullAt
	var since string
	if dialect == ActiveDirectory {
		since = fmt.Sprintf("(uSNChanged>=%d)", prev.USN+1)
	} else {
		since = "(modifyTimestamp>=" + prev.ModifiedSince.UTC().Format("20060102150405Z") + ")"
	}
	plan := Plan{Mode: ModeIncremental, Filter: "(&" + scope.Filter + since + ")", Next: next}
	if computers != nil {
		plan.ComputerFilter = "(&" + computers.Filter + since + ")"
	}
	return plan
}

func needsFullSync(dialect Dialect, scope Scope, computers *Scope, prev *Cursor, root RootInfo, startedAt time.Time, fullInterval time.Duration) bool {
	switch {
	case prev == nil, prev.Dialect != dialect, !prev.Scope.Equal(scope):
		return true
	case (prev.Computers == nil) != (computers == nil), computers != nil && !prev.Computers.Equal(*computers):
		return true
	case dialect == ActiveDirectory && prev.Server != root.Server:
		return true
	default:
		return !startedAt.Before(prev.LastFullAt.Add(fullInterval))
	}
}
