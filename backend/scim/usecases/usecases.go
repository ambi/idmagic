package usecases

import (
	"errors"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var ErrNotFound = errors.New("SCIM resource not found")

// ErrDuplicate signals a uniqueness conflict (userName/displayName already
// used within the tenant). Wrapped with errors.Is-compatible context by
// callers; handlers map it to HTTP 409 with scimType "uniqueness".
var ErrDuplicate = errors.New("SCIM resource already exists")

type Usecases struct {
	ScimRepo  ports.ScimRepository
	UserRepo  userports.UserRepository
	GroupRepo groupports.GroupRepository
	Emit      func(spec.DomainEvent)
}

func NewUsecases(
	scimRepo ports.ScimRepository,
	userRepo userports.UserRepository,
	groupRepo groupports.GroupRepository,
	emit func(spec.DomainEvent),
) *Usecases {
	return &Usecases{
		ScimRepo:  scimRepo,
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
		Emit:      emit,
	}
}

// ListQuery is the normalized input to ListUsers/ListGroups (SCL
// interfaces.ListScimUsers / ListScimGroups). StartIndex/Count are nil when
// the caller omitted the corresponding query parameter; HasCount
// distinguishes an omitted count from an explicit 0.
type ListQuery struct {
	Filter     string
	StartIndex *int
	Count      *int
	HasCount   bool
}

// ListResult is a filtered, paginated SCIM collection page.
type ListResult struct {
	Total        int
	Items        []map[string]any
	StartIndex   int
	ItemsPerPage int
}
