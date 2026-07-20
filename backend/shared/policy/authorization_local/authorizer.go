package authorization_local

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type Local struct{}

func (Local) Authorize(_ context.Context, req spec.AuthZRequest) (spec.AuthZResponse, error) {
	return spec.Evaluate(req), nil
}
