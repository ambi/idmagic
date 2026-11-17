package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ignoreRevocation は、失効エポックを確かめないテストが、止める操作へ明示して渡す何もしない反応器である。
type ignoreRevocation struct{}

func (ignoreRevocation) React(context.Context, spec.DomainEvent) error { return nil }
