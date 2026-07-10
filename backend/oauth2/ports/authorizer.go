package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type Authorizer interface {
	Authorize(ctx context.Context, req spec.AuthZRequest) (spec.AuthZResponse, error)
}
