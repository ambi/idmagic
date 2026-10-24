package usecases

import (
	"context"
	"fmt"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// CheckResult は、完全な再同期で送るはずの User とコンピューターの項目の数と、送らないエントリである。
type CheckResult struct {
	Items            int
	Skipped          []domain.IncompleteEntryError
	Computers        int
	SkippedComputers []domain.IncompleteEntryError
}

// Check は同期と同じ接続、検索、対応付けで検索の範囲の全件を読み、IdMagic へは何も送らない
// (REQ-DIRECTORYCONNECTOR-006)。差分の位置も読み書きしない。computers が nil ならコンピューターを読まない。
func Check(ctx context.Context, directory ports.Directory, dialect domain.Dialect, scope domain.Scope, computers *domain.Scope, mapping domain.Mapping) (CheckResult, error) {
	session, err := directory.Connect(ctx)
	if err != nil {
		return CheckResult{}, fmt.Errorf("connect to ldap: %w", err)
	}
	defer func() { _ = session.Close() }()
	var result CheckResult
	err = readItems(ctx, session, dialect, scope.BaseDNs, scope.Filter, mapping,
		func(domain.Item) error {
			result.Items++
			return nil
		},
		func(incomplete *domain.IncompleteEntryError) { result.Skipped = append(result.Skipped, *incomplete) })
	if err == nil && computers != nil {
		err = readComputers(ctx, session, computers.BaseDNs, computers.Filter,
			func(domain.ComputerItem) error {
				result.Computers++
				return nil
			},
			func(incomplete *domain.IncompleteEntryError) {
				result.SkippedComputers = append(result.SkippedComputers, *incomplete)
			})
	}
	if err != nil {
		return result, fmt.Errorf("search: %w", err)
	}
	return result, nil
}
