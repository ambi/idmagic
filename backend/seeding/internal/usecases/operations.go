package usecases

import (
	"fmt"

	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
)

// resourceOperations は計画と適用で同じ比較を使い、作成だけに作用を許す。
type resourceOperations struct {
	apply bool
	items []seedingdomain.Operation
}

func (p *resourceOperations) ensure(key string, exists, same bool, create func() error) error {
	kind := seedingdomain.OperationCreate
	if exists {
		kind = seedingdomain.OperationNoop
		if !same {
			kind = seedingdomain.OperationConflict
		}
	}
	p.items = append(p.items, seedingdomain.Operation{LogicalKey: key, Kind: kind, Summary: "seed resource"})
	if kind == seedingdomain.OperationConflict {
		return fmt.Errorf("seed drift at %s", key)
	}
	if p.apply && kind == seedingdomain.OperationCreate {
		return create()
	}
	return nil
}
