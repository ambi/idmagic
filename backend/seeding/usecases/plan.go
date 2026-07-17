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

// Contributor は各 record context の公開 command surface を、Seed の plan/apply 契約へ
// 変換する adapter である。Seeding 自身は resource の domain model を import しない。
type Contributor interface {
	Plan(context.Context, domain.Request) (domain.Plan, error)
	Apply(context.Context, domain.Request) error
}

// Plan は contributor を接続する前でも安全な request validation を一箇所に保つ。
// 次タスクで各 record context の published command surface を contributor として注入する。
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
		return domain.Plan{}, err
	}
	if plan.Count(domain.OperationConflict) != 0 {
		return plan, fmt.Errorf("seed plan contains %d conflict(s)", plan.Count(domain.OperationConflict))
	}
	if request.Mode == domain.ModeDryRun {
		return plan, nil
	}
	// 同じ profile を同じ process 内で二重に適用しない。PostgreSQL 構成では
	// contributor が process 境界の advisory lock を追加できるよう、ここは
	// 永続化契約を持たない最小の排他に留める。
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
