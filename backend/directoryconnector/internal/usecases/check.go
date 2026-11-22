package usecases

import (
	"context"
	"fmt"

	directoryconnectordomain "github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	directoryconnectorports "github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// CheckResult は、完全な再同期で送るはずの項目の数と、送らないエントリである。
type CheckResult struct {
	Items   int
	Skipped []directoryconnectordomain.IncompleteEntryError
}

// Check は同期と同じ接続、検索、対応付けで検索の範囲の全件を読み、IdMagic へは何も送らない
// (REQ-DIRECTORYCONNECTOR-006)。差分の位置も読み書きしない。
func Check(ctx context.Context, directory directoryconnectorports.Directory, dialect directoryconnectordomain.Dialect, scope directoryconnectordomain.Scope, mapping directoryconnectordomain.Mapping) (CheckResult, error) {
	session, err := directory.Connect(ctx)
	if err != nil {
		return CheckResult{}, fmt.Errorf("connect to ldap: %w", err)
	}
	defer func() { _ = session.Close() }()
	var result CheckResult
	err = readItems(ctx, session, dialect, scope.BaseDNs, scope.Filter, mapping,
		func(directoryconnectordomain.Item) error {
			result.Items++
			return nil
		},
		func(incomplete *directoryconnectordomain.IncompleteEntryError) {
			result.Skipped = append(result.Skipped, *incomplete)
		})
	if err != nil {
		return result, fmt.Errorf("search: %w", err)
	}
	return result, nil
}
