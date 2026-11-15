package usecases

import (
	"fmt"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

// resourceOperations は計画と適用で同じ比較を使い、作成だけに作用を許す。
type resourceOperations struct {
	apply bool
	items []domain.Operation
}

func (p *resourceOperations) ensure(key string, exists, same bool, create func() error) error {
	kind := domain.OperationCreate
	if exists {
		kind = domain.OperationNoop
		if !same {
			kind = domain.OperationConflict
		}
	}
	p.items = append(p.items, domain.Operation{LogicalKey: key, Kind: kind, Summary: "seed resource"})
	if kind == domain.OperationConflict {
		return fmt.Errorf("seed drift at %s", key)
	}
	if p.apply && kind == domain.OperationCreate {
		return create()
	}
	return nil
}
