package sourcingdeps

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
)

// directoryWrites は、Directory Connector の対応と所有者の作成を結び、IdManagement の拒否を
// 取り込みの port のエラーへ写す。
type directoryWrites struct{ writes *writes }

func NewDirectoryWrites(identity idmanagement.Module, lifecycle idmanagement.LifecycleDeps, source sourcing.Module, emit func(spec.DomainEvent)) directoryports.IdentityWrites {
	return directoryWrites{writes: newWrites(identity, lifecycle, source, emit, directoryConnectorActor)}
}

func (w directoryWrites) CreateUser(ctx context.Context, correlation directoryports.UserCorrelation, user *userdomain.User) (*userdomain.User, error) {
	var created *userdomain.User
	var correlations directoryports.CorrelationWriter
	inTransaction := func(body func(pgx.Tx) error) error {
		return w.writes.source.CreateDirectoryUserInTransaction(ctx, func(tx pgx.Tx, bound directoryports.CorrelationWriter) error {
			correlations = bound
			return body(tx)
		})
	}
	err := w.writes.create(inTransaction, func(commands idmanagement.SourceCommands) error {
		var err error
		created, err = commands.CreateUser(ctx, user)
		if err != nil {
			return err
		}
		correlation.TenantID, correlation.UserID = created.TenantID, created.ID
		return correlations.SaveUserCorrelation(ctx, correlation)
	})
	if err != nil {
		return nil, refusal(err)
	}
	return created, nil
}

func (w directoryWrites) UpdateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error) {
	updated, err := w.writes.UpdateUser(ctx, user)
	return updated, refusal(err)
}

// refusal は IdManagement の拒否に、取り込みが項目の失敗として読む port のエラーを添える。
func refusal(err error) error {
	switch {
	case errors.Is(err, idmanagement.ErrSourceUsernameConflict):
		return errors.Join(directoryports.ErrUsernameConflict, err)
	case errors.Is(err, idmanagement.ErrSourceEmailConflict):
		return errors.Join(directoryports.ErrEmailConflict, err)
	case errors.Is(err, idmanagement.ErrSourceInvalidAttribute):
		return errors.Join(directoryports.ErrInvalidAttribute, err)
	case errors.As(err, new(*userports.UniqueAttributeConflictError)):
		return errors.Join(directoryports.ErrAttributeConflict, err)
	default:
		return err
	}
}
