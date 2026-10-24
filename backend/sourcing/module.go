// Package sourcing は Sourcing モジュールの DI 組立を所有する。
// source slice は scim (SCIM 2.0 server) と directory (Directory Connector) である。source 非依存の
// コアは、二つの source に実在する共通点が判明した時点で切り出す (決定 3・4)。
package sourcing

import (
	"context"
	"errors"
	"time"

	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	directorywrites "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	directoryhttp "github.com/ambi/idmagic/backend/sourcing/internal/directory/handlers_http"
	directoryports "github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	directoryusecases "github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
	scimhttp "github.com/ambi/idmagic/backend/sourcing/internal/scim/handlers_http"
	scimusecases "github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type Module struct {
	ScimRepo ports.ScimRepository
	Writes   ports.IdentityWrites
	begin    func(context.Context) (pgx.Tx, error)
	// UserLifecycle は、SCIM が User を止める、削除を予約するときに通す IdManagement の操作である。
	// イベントの出力先が決まった後で、組み立ての地点が設定する。
	UserLifecycle ports.UserLifecycle
	// DirectoryWrites と DirectoryUserLifecycle は、Directory Connector の取り込みが通す
	// IdManagement の操作である。actor を directory_connector にして組み立て地点が設定する。
	DirectoryWrites        directorywrites.IdentityWrites
	DirectoryUserLifecycle directorywrites.UserLifecycle
	connectors             directoryports.ConnectorRepository
	correlations           directoryports.CorrelationRepository
	// memoryCorrelations は、トランザクションを持たないメモリ構成で対応を保存する先である。
	memoryCorrelations directorywrites.CorrelationWriter
	runs               directoryports.IngestionRunRepository
}

// Register は SCIM と Directory Connector の経路を登録する。replay は Connector の JWT の
// jti を記録する共有の保存先で、Sourcing の用途の接頭辞を付けて使う。
func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	userRepo userports.UserRepository, groupRepo groupports.GroupRepository, emit func(spec.DomainEvent),
	apiTokenAuthenticator apitokenports.Authenticator, replay tokens_jose.ReplayStore,
) {
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps: deps, Authenticator: authenticator,
		Usecases:              scimusecases.NewUsecases(m.ScimRepo, userRepo, groupRepo, m.UserLifecycle, emit, m.Writes),
		ApiTokenAuthenticator: apiTokenAuthenticator,
	})
	directoryhttp.RegisterRoutes(g, directoryhttp.Deps{
		Deps: deps, Authenticator: authenticator,
		Service: directoryusecases.New(m.connectors, prefixedReplay{replay}, emit),
		Ingestion: directoryusecases.NewIngestion(directoryusecases.IngestionDeps{
			Runs: m.runs, Correlations: m.correlations, Users: userRepo,
			Writes: m.DirectoryWrites, Lifecycle: m.DirectoryUserLifecycle,
		}),
	})
}

// prefixedReplay は、Connector の jti を、同じ保存先を使うほかの用途の jti と分ける。
type prefixedReplay struct{ store tokens_jose.ReplayStore }

func (p prefixedReplay) RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error) {
	if p.store == nil {
		return false, errReplayStoreMissing
	}
	return p.store.RecordIfNew(ctx, "directory_connector:"+jti, windowSeconds, now)
}

var errReplayStoreMissing = errors.New("sourcing: the directory connector replay store is not configured")
