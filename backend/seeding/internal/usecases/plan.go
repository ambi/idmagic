// Package usecases は Seeding の application service を定義する。
package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

var requestLocks = struct {
	sync.Mutex
	values map[string]*sync.Mutex
}{values: map[string]*sync.Mutex{}}

// Contributor は所有モジュールの公開操作を、seed の計画と適用へ結ぶ。
type Contributor interface {
	Plan(context.Context, domain.Request) (domain.Plan, error)
	Apply(context.Context, domain.Request) error
}

// Plan は contributor を接続する前にも要求の検証を実行する。
func Plan(request domain.Request) (domain.Plan, error) {
	if err := request.Validate(); err != nil {
		return domain.Plan{}, err
	}
	return domain.Plan{}, nil
}

// Run は dry-run と apply に同じ planner を使う。apply の直前・直後に再計画することで、
// stale plan や contributor が残した drift を成功として扱わない。
func Run(ctx context.Context, request domain.Request, contributor Contributor) (domain.Plan, error) {
	if err := request.Validate(); err != nil {
		return domain.Plan{}, err
	}
	plan, err := contributor.Plan(ctx, request)
	if err != nil {
		return plan, err
	}
	if plan.Count(domain.OperationConflict) != 0 {
		return plan, fmt.Errorf("seed plan contains %d conflict(s)", plan.Count(domain.OperationConflict))
	}
	if request.Mode == domain.ModeDryRun {
		return plan, nil
	}
	// メモリ構成も同じ要求を直列化する。PostgreSQL 構成の追加の排他は
	// 構成入口が再計画と適用後の確認までを包む。
	lock := lockFor(request)
	lock.Lock()
	defer lock.Unlock()
	plan, err = contributor.Plan(ctx, request)
	if err != nil {
		return domain.Plan{}, err
	}
	if plan.Count(domain.OperationConflict) != 0 {
		return plan, fmt.Errorf("seed plan contains %d conflict(s)", plan.Count(domain.OperationConflict))
	}
	if err := contributor.Apply(ctx, request); err != nil {
		return plan, err
	}
	verified, err := contributor.Plan(ctx, request)
	if err != nil {
		return plan, err
	}
	if verified.Count(domain.OperationConflict) != 0 || verified.Count(domain.OperationCreate) != 0 || verified.Count(domain.OperationUpdate) != 0 {
		return verified, fmt.Errorf("seed apply did not converge")
	}
	return verified, nil
}

func lockFor(request domain.Request) *sync.Mutex {
	key := string(request.Environment) + ":" + string(request.Profile) + ":" + request.TenantID
	requestLocks.Lock()
	defer requestLocks.Unlock()
	if lock := requestLocks.values[key]; lock != nil {
		return lock
	}
	lock := &sync.Mutex{}
	requestLocks.values[key] = lock
	return lock
}
