package domain

// 規則の実行回数の上限 (REQ-OAUTH2-062、REQ-OAUTH2-064)。

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ResourceAccessLimitUnit は上限の回数を分ける単位である。
type ResourceAccessLimitUnit string

const (
	ResourceAccessLimitByAgent    ResourceAccessLimitUnit = "agent"
	ResourceAccessLimitByUser     ResourceAccessLimitUnit = "user"
	ResourceAccessLimitByResource ResourceAccessLimitUnit = "resource"
)

// ResourceAccessLimit は操作を許可する回数の固定窓の上限である。By が空なら操作の全体で一つの回数を持つ。
type ResourceAccessLimit struct {
	By            []ResourceAccessLimitUnit `json:"by,omitempty"`
	MaxRequests   int                       `json:"max_requests"`
	WindowSeconds int                       `json:"window_seconds"`
}

// ResourceAccessLimitSubject は評価から導いた単位の値である。値のない単位は空である。
type ResourceAccessLimitSubject struct {
	AgentID    string
	UserID     string
	ResourceID string
}

const (
	// MaxResourceAccessLimits は 1 つの規則が持てる上限の数である。
	MaxResourceAccessLimits      = 8
	maxResourceAccessLimitCount  = 1_000_000_000
	maxResourceAccessLimitWindow = 30 * 24 * 60 * 60
)

// Bucket は回数を分ける値を、By の順序によらない表現で返す。単位の値がなければ、上限を適用しない。
// 引用するので、区切り文字を含む識別子でも衝突しない。
func (l ResourceAccessLimit) Bucket(subject ResourceAccessLimitSubject) (string, bool) {
	var bucket strings.Builder
	bucket.WriteString(strconv.Itoa(l.WindowSeconds))
	for _, unit := range []struct {
		name  ResourceAccessLimitUnit
		value string
	}{
		{ResourceAccessLimitByAgent, subject.AgentID},
		{ResourceAccessLimitByUser, subject.UserID},
		{ResourceAccessLimitByResource, subject.ResourceID},
	} {
		if !slices.Contains(l.By, unit.name) {
			continue
		}
		if unit.value == "" {
			return "", false
		}
		fmt.Fprintf(&bucket, ":%s=%q", unit.name, unit.value)
	}
	return bucket.String(), true
}

// OrderedLimits は数える順に上限を返す。狭い単位と短い窓を先に数え、超えた要求が共有の回数と
// 長い窓の回数を減らしにくくする。順序は宣言に依らない。
func (r ResourceAccessRule) OrderedLimits() []ResourceAccessLimit {
	return slices.SortedFunc(slices.Values(r.Limits), func(a, b ResourceAccessLimit) int {
		return cmp.Or(
			cmp.Compare(len(b.By), len(a.By)),
			cmp.Compare(a.WindowSeconds, b.WindowSeconds),
			cmp.Compare(a.unitsKey(), b.unitsKey()),
		)
	})
}

// unitsKey は By の順序によらない表現で、同じ単位の集合を同じ値にする。
func (l ResourceAccessLimit) unitsKey() string {
	units := make([]string, 0, len(l.By))
	for _, unit := range l.By {
		units = append(units, string(unit))
	}
	slices.Sort(units)
	return strings.Join(units, ",")
}

func validateResourceAccessLimits(limits []ResourceAccessLimit) error {
	if len(limits) > MaxResourceAccessLimits {
		return fmt.Errorf("access_rules limits must not exceed %d", MaxResourceAccessLimits)
	}
	seen := make(map[string]struct{}, len(limits))
	for _, limit := range limits {
		if limit.MaxRequests < 1 || limit.MaxRequests > maxResourceAccessLimitCount {
			return fmt.Errorf("access_rules limits max_requests must be between 1 and %d", maxResourceAccessLimitCount)
		}
		if limit.WindowSeconds < 1 || limit.WindowSeconds > maxResourceAccessLimitWindow {
			return fmt.Errorf("access_rules limits window_seconds must be between 1 and %d", maxResourceAccessLimitWindow)
		}
		units := make(map[ResourceAccessLimitUnit]struct{}, len(limit.By))
		for _, unit := range limit.By {
			if unit != ResourceAccessLimitByAgent && unit != ResourceAccessLimitByUser && unit != ResourceAccessLimitByResource {
				return errors.New("access_rules limits by must be agent, user or resource")
			}
			if _, duplicate := units[unit]; duplicate {
				return errors.New("access_rules limits by must not repeat a unit")
			}
			units[unit] = struct{}{}
		}
		key := fmt.Sprintf("%s|%d", limit.unitsKey(), limit.WindowSeconds)
		if _, duplicate := seen[key]; duplicate {
			return errors.New("access_rules limits must not repeat a by and window_seconds pair")
		}
		seen[key] = struct{}{}
	}
	return nil
}
