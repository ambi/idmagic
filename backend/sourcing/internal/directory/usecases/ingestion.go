package usecases

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/google/uuid"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/sourceuser"
)

// 管理者の照会が返す件数の上限 (REQ-SOURCING-019)。
const (
	maxListedRuns     = 50
	maxListedFailures = 100
)

var (
	ErrRunNotFound   = errors.New("directory ingestion: run not found")
	ErrRunClosed     = errors.New("directory ingestion: run is closed")
	ErrRunIncomplete = errors.New("directory ingestion: item count does not match the recorded batches")
	ErrBatchConflict = errors.New("directory ingestion: batch id reused with different items")
)

// IngestionDeps は取り込みが使う保存先と IdManagement の操作である。
type IngestionDeps struct {
	Connectors   ports.ConnectorRepository
	Emit         func(spec.DomainEvent)
	Runs         ports.IngestionRunRepository
	Correlations ports.CorrelationRepository
	Users        userports.UserRepository
	Writes       directoryports.IdentityWrites
	Lifecycle    directoryports.UserLifecycle
}

// Ingestion は Connector が送る差分を、外部の不変 ID で対応する User へ反映する。
type Ingestion struct {
	deps IngestionDeps
	now  func() time.Time
}

func NewIngestion(deps IngestionDeps, options ...IngestionOption) *Ingestion {
	if deps.Connectors == nil || deps.Emit == nil {
		panic("directory ingestion requires connector settings and an event sink")
	}
	i := &Ingestion{deps: deps, now: time.Now}
	for _, option := range options {
		option(i)
	}
	return i
}

type IngestionOption func(*Ingestion)

func WithIngestionClock(now func() time.Time) IngestionOption {
	return func(i *Ingestion) { i.now = now }
}

// Failure は実行の失敗した項目である。
type Failure struct {
	BatchID    string
	ExternalID string
	Error      string
}

func (i *Ingestion) StartRun(ctx context.Context, tenantID, connectorID, mode string) (domain.IngestionRun, error) {
	return withConnectorLock(ctx, i, tenantID, connectorID, func() (domain.IngestionRun, error) {
		parsed, err := domain.ParseIngestionMode(mode)
		if err != nil {
			return domain.IngestionRun{}, err
		}
		run := domain.IngestionRun{
			TenantID: tenantID, ID: uuid.NewString(), ConnectorID: connectorID,
			Mode: parsed, Status: domain.RunRunning, StartedAt: i.now().UTC(),
		}
		if err := i.deps.Runs.Start(ctx, run); err != nil {
			return domain.IngestionRun{}, err
		}
		return run, nil
	})
}

// ApplyUserBatch は項目を順に一件ずつ確定し、項目ごとの結果を返す。適用し終えたバッチは記録し、
// 同じバッチの再送には適用し直さずに記録した結果を返す。
func (i *Ingestion) ApplyUserBatch(ctx context.Context, tenantID, connectorID, runID, batchID string, items []domain.UserItem) ([]domain.ItemResult, error) {
	return withConnectorLock(ctx, i, tenantID, connectorID, func() ([]domain.ItemResult, error) {
		run, err := i.findRun(ctx, tenantID, connectorID, runID)
		if err != nil {
			return nil, err
		}
		if err := domain.ValidateUserBatch(batchID, items); err != nil {
			return nil, err
		}
		digest := domain.BatchDigest(items)
		if recorded, err := i.deps.Runs.FindBatch(ctx, tenantID, runID, batchID); err != nil || recorded != nil {
			if err != nil {
				return nil, err
			}
			return replay(*recorded, digest)
		}
		if run.Status != domain.RunRunning {
			return nil, ErrRunClosed
		}
		results := make([]domain.ItemResult, len(items))
		for n, item := range items {
			if results[n], err = i.applyItem(ctx, run, item); err != nil {
				return nil, err
			}
		}
		recorded, err := i.deps.Runs.RecordBatch(ctx, domain.IngestionBatch{
			TenantID: tenantID, RunID: runID, BatchID: batchID, Digest: digest, Results: results, RecordedAt: i.now().UTC(),
		})
		if err != nil {
			return nil, err
		}
		return replay(recorded, digest)
	})
}

// replay は記録したバッチの結果を返す。同じ ID で項目が違えば、どちらの結果も返さない。
func replay(recorded domain.IngestionBatch, digest string) ([]domain.ItemResult, error) {
	if recorded.Digest != digest {
		return nil, ErrBatchConflict
	}
	return recorded.Results, nil
}

// CompleteRun は実行を completed にする。full の実行では、この実行に現れなかった対応の Active の
// User を無効にする。報告した項目数が記録した数と違えば、何も変えない。
func (i *Ingestion) CompleteRun(ctx context.Context, tenantID, connectorID, runID string, itemCount int) (domain.IngestionRun, error) {
	return withConnectorLock(ctx, i, tenantID, connectorID, func() (domain.IngestionRun, error) {
		run, err := i.findRun(ctx, tenantID, connectorID, runID)
		if err != nil {
			return domain.IngestionRun{}, err
		}
		switch {
		case run.Status == domain.RunCompleted || run.Status == domain.RunAwaitingApproval:
			return run, nil
		case run.Status != domain.RunRunning:
			return domain.IngestionRun{}, ErrRunClosed
		case itemCount != run.Counts.Items:
			return domain.IngestionRun{}, ErrRunIncomplete
		}
		disabled := 0
		if run.Mode == domain.ModeFull {
			settings, err := i.deps.Connectors.Find(ctx, tenantID, connectorID)
			if err != nil {
				return domain.IngestionRun{}, err
			}
			if settings == nil {
				return domain.IngestionRun{}, ErrRunNotFound
			}
			if err := settings.Thresholds.Validate(); err != nil {
				return domain.IngestionRun{}, err
			}
			candidates, active, err := i.disableCandidates(ctx, run)
			if err != nil {
				return domain.IngestionRun{}, err
			}
			if settings.Thresholds.RequiresApproval(len(candidates), active) {
				run.Status = domain.RunAwaitingApproval
				run.Protection = &domain.IngestionProtection{Thresholds: settings.Thresholds, ActiveCount: active, HeldAt: i.now().UTC(), Candidates: candidates}
				if err := i.deps.Runs.SaveProgress(ctx, run); err != nil {
					return domain.IngestionRun{}, err
				}
				i.deps.Emit(&domain.DirectoryIngestionRunHeld{At: run.Protection.HeldAt, TenantID: tenantID, ConnectorID: connectorID, RunID: runID, CandidateCount: len(candidates), ActiveCount: active, CountThreshold: settings.Thresholds.Count, PercentThreshold: settings.Thresholds.Percent})
				return run, nil
			}
			if disabled, err = i.disableNotSeen(ctx, run); err != nil {
				return domain.IngestionRun{}, err
			}
		}
		completedAt := i.now().UTC()
		changed, err := i.deps.Runs.Complete(ctx, tenantID, runID, completedAt, disabled)
		if err != nil {
			return domain.IngestionRun{}, err
		}
		if !changed {
			return domain.IngestionRun{}, ErrRunClosed
		}
		run.Status, run.CompletedAt, run.Counts.Disabled = domain.RunCompleted, &completedAt, disabled
		return run, nil
	})
}

func (i *Ingestion) disableNotSeen(ctx context.Context, run domain.IngestionRun) (int, error) {
	correlations, err := i.deps.Correlations.ListNotSeenIn(ctx, run.TenantID, run.ConnectorID, run.ID)
	if err != nil {
		return 0, err
	}
	disabled := 0
	for _, correlation := range correlations {
		user, err := i.deps.Users.FindBySub(ctx, correlation.UserID)
		if err != nil {
			return 0, err
		}
		if user == nil || user.TenantID != run.TenantID || user.Lifecycle.EffectiveStatus() != idmdomain.UserStatusActive {
			continue
		}
		changed, err := i.deps.Lifecycle.SetUserDisabled(ctx, run.TenantID, user.ID, true, i.now())
		if err != nil {
			return 0, err
		}
		if changed {
			disabled++
		}
	}
	return disabled, nil
}

func withConnectorLock[T any](ctx context.Context, i *Ingestion, tenantID, connectorID string, apply func() (T, error)) (T, error) {
	var result T
	err := i.deps.Runs.WithConnectorLock(ctx, tenantID, connectorID, func() error {
		var err error
		result, err = apply()
		return err
	})
	return result, err
}

func (i *Ingestion) disableCandidates(ctx context.Context, run domain.IngestionRun) ([]domain.DisableCandidate, int, error) {
	correlations, err := i.deps.Correlations.List(ctx, run.TenantID, run.ConnectorID)
	if err != nil {
		return nil, 0, err
	}
	candidates := []domain.DisableCandidate{}
	active := 0
	for _, c := range correlations {
		user, err := i.deps.Users.FindBySub(ctx, c.UserID)
		if err != nil {
			return nil, 0, err
		}
		if user == nil || user.TenantID != run.TenantID || user.Lifecycle.EffectiveStatus() != idmdomain.UserStatusActive {
			continue
		}
		active++
		if c.LastSeenRunID != run.ID {
			candidates = append(candidates, domain.DisableCandidate{UserID: c.UserID, ExternalID: c.ExternalID})
		}
	}
	return candidates, active, nil
}

func (i *Ingestion) ApproveRun(ctx context.Context, tenantID, connectorID, runID, actorID string) (domain.IngestionRun, error) {
	return withConnectorLock(ctx, i, tenantID, connectorID, func() (domain.IngestionRun, error) {
		run, err := i.findRun(ctx, tenantID, connectorID, runID)
		if err != nil {
			return domain.IngestionRun{}, err
		}
		if run.Status == domain.RunCompleted {
			return run, nil
		}
		if run.Status != domain.RunAwaitingApproval && run.Status != domain.RunApplying {
			return domain.IngestionRun{}, ErrRunClosed
		}
		if run.Protection == nil {
			return domain.IngestionRun{}, errors.New("directory ingestion: missing saved candidates")
		}
		if run.Status == domain.RunAwaitingApproval {
			now := i.now().UTC()
			run.Status = domain.RunApplying
			run.Protection.ApprovedAt = &now
			run.Protection.ApprovedBy = actorID
			if err := i.deps.Runs.SaveProgress(ctx, run); err != nil {
				return domain.IngestionRun{}, err
			}
			i.deps.Emit(&domain.DirectoryIngestionRunApproved{At: now, TenantID: tenantID, ConnectorID: connectorID, RunID: runID, ActorUserID: actorID})
		}
		for n, c := range run.Protection.Candidates {
			if c.Done {
				continue
			}
			user, err := i.deps.Users.FindBySub(ctx, c.UserID)
			if err != nil {
				return domain.IngestionRun{}, err
			}
			correlation, err := i.deps.Correlations.Find(ctx, tenantID, connectorID, c.ExternalID)
			if err != nil {
				return domain.IngestionRun{}, err
			}
			if user != nil && user.TenantID == tenantID && correlation != nil && correlation.UserID == c.UserID && (user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusActive || (c.Attempted && user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusDisabled)) {
				// 所有者の保存後に外向きの作用が失敗しても、再送は残る作用まで再実行する。
				run.Protection.Candidates[n].Attempted = true
				if err := i.deps.Runs.SaveProgress(ctx, run); err != nil {
					return domain.IngestionRun{}, err
				}
				changed, err := i.deps.Lifecycle.SetUserDisabled(ctx, tenantID, c.UserID, true, i.now())
				if err != nil {
					return domain.IngestionRun{}, err
				}
				if changed {
					run.Counts.Disabled++
				}
			}
			run.Protection.Candidates[n].Done = true
			if err := i.deps.Runs.SaveProgress(ctx, run); err != nil {
				return domain.IngestionRun{}, err
			}
		}
		now := i.now().UTC()
		run.Status = domain.RunCompleted
		run.CompletedAt = &now
		return run, i.deps.Runs.SaveProgress(ctx, run)
	})
}

func (i *Ingestion) DiscardRun(ctx context.Context, tenantID, connectorID, runID, actorID string) (domain.IngestionRun, error) {
	return withConnectorLock(ctx, i, tenantID, connectorID, func() (domain.IngestionRun, error) {
		run, err := i.findRun(ctx, tenantID, connectorID, runID)
		if err != nil {
			return domain.IngestionRun{}, err
		}
		if run.Status == domain.RunDiscarded {
			return run, nil
		}
		if run.Status != domain.RunAwaitingApproval || run.Protection == nil {
			return domain.IngestionRun{}, ErrRunClosed
		}
		now := i.now().UTC()
		run.Status = domain.RunDiscarded
		run.Protection.DiscardedAt = &now
		if err := i.deps.Runs.SaveProgress(ctx, run); err != nil {
			return domain.IngestionRun{}, err
		}
		i.deps.Emit(&domain.DirectoryIngestionRunDiscarded{At: now, TenantID: tenantID, ConnectorID: connectorID, RunID: runID, ActorUserID: actorID})
		return run, nil
	})
}

// ListRuns は Connector の実行を開始時刻の新しい順に返す。connectorID は呼び出し側が
// テナントの Connector であることを確かめておく。
func (i *Ingestion) ListRuns(ctx context.Context, tenantID, connectorID string) ([]domain.IngestionRun, error) {
	return i.deps.Runs.List(ctx, tenantID, connectorID, maxListedRuns)
}

// GetRun は実行と、記録した順の失敗した項目を返す。
func (i *Ingestion) GetRun(ctx context.Context, tenantID, connectorID, runID string) (domain.IngestionRun, []Failure, error) {
	run, err := i.findRun(ctx, tenantID, connectorID, runID)
	if err != nil {
		return domain.IngestionRun{}, nil, err
	}
	batches, err := i.deps.Runs.ListBatches(ctx, tenantID, runID)
	if err != nil {
		return domain.IngestionRun{}, nil, err
	}
	failures := []Failure{}
	for _, batch := range batches {
		for _, result := range batch.Results {
			if result.Outcome != domain.OutcomeFailed {
				continue
			}
			if len(failures) == maxListedFailures {
				return run, failures, nil
			}
			failures = append(failures, Failure{BatchID: batch.BatchID, ExternalID: result.ExternalID, Error: result.Error})
		}
	}
	return run, failures, nil
}

func (i *Ingestion) findRun(ctx context.Context, tenantID, connectorID, runID string) (domain.IngestionRun, error) {
	run, err := i.deps.Runs.Find(ctx, tenantID, connectorID, runID)
	if err != nil {
		return domain.IngestionRun{}, err
	}
	if run == nil {
		return domain.IngestionRun{}, ErrRunNotFound
	}
	return *run, nil
}

// applyItem は一つの項目を確定する。項目の失敗は結果として返し、保存先の失敗だけをエラーで返す。
func (i *Ingestion) applyItem(ctx context.Context, run domain.IngestionRun, item domain.UserItem) (domain.ItemResult, error) {
	correlation, err := i.deps.Correlations.Find(ctx, run.TenantID, run.ConnectorID, item.ExternalID)
	if err != nil {
		return domain.ItemResult{}, err
	}
	if correlation == nil {
		if item.Deleted {
			return outcome(item, domain.OutcomeUnchanged), nil
		}
		return i.createUser(ctx, run, item)
	}
	result, err := i.applyToCorrelatedUser(ctx, run, correlation.UserID, item)
	if err != nil {
		return domain.ItemResult{}, err
	}
	// 失敗した項目にも印を付け、属性の誤りで変えられなかった User を完全な再同期で無効にしない。
	if err := i.deps.Correlations.MarkSeen(ctx, run.TenantID, run.ConnectorID, item.ExternalID, run.ID); err != nil {
		return domain.ItemResult{}, err
	}
	return result, nil
}

func (i *Ingestion) createUser(ctx context.Context, run domain.IngestionRun, item domain.UserItem) (domain.ItemResult, error) {
	now := i.now()
	status := idmdomain.UserStatusActive
	if !item.Enabled {
		status = idmdomain.UserStatusDisabled
	}
	user := &userdomain.User{
		TenantID: run.TenantID, EmailVerified: true, Roles: []string{},
		Lifecycle:  userdomain.UserLifecycle{Status: status, StatusChangedAt: &now},
		Attributes: map[string]userdomain.AttributeValue{}, CreatedAt: now, UpdatedAt: now,
	}
	sourceAttributes(item).Apply(user)
	_, err := i.deps.Writes.CreateUser(ctx, directoryports.UserCorrelation{
		TenantID: run.TenantID, ConnectorID: run.ConnectorID, ExternalID: item.ExternalID, LastSeenRunID: run.ID,
	}, user)
	return writeOutcome(item, domain.OutcomeCreated, err)
}

func (i *Ingestion) applyToCorrelatedUser(ctx context.Context, run domain.IngestionRun, userID string, item domain.UserItem) (domain.ItemResult, error) {
	user, err := i.deps.Users.FindBySub(ctx, userID)
	if err != nil {
		return domain.ItemResult{}, err
	}
	if user == nil || user.TenantID != run.TenantID || user.IsSoftDeleted() || user.IsDeleted() {
		if item.Deleted {
			return outcome(item, domain.OutcomeUnchanged), nil
		}
		return failed(item, domain.ItemErrorUserDeleted), nil
	}
	if item.Deleted {
		if err := i.deps.Lifecycle.ScheduleUserDeletion(ctx, run.TenantID, user.ID, i.now()); err != nil {
			return domain.ItemResult{}, err
		}
		return outcome(item, domain.OutcomeDeleted), nil
	}

	attributes := sourceAttributes(item)
	attributesChanged := !attributes.Matches(*user)
	if attributesChanged {
		// 保存先が返す値を書き換えないよう、複製へ当てる。
		next := *user
		next.Attributes = maps.Clone(user.Attributes)
		attributes.Apply(&next)
		next.UpdatedAt = i.now()
		if _, err := i.deps.Writes.UpdateUser(ctx, &next); err != nil {
			return writeOutcome(item, domain.OutcomeUpdated, err)
		}
	}
	lifecycleChanged := false
	if active := user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusActive; active != item.Enabled {
		if lifecycleChanged, err = i.deps.Lifecycle.SetUserDisabled(ctx, run.TenantID, user.ID, !item.Enabled, i.now()); err != nil {
			return domain.ItemResult{}, err
		}
	}
	if attributesChanged || lifecycleChanged {
		return outcome(item, domain.OutcomeUpdated), nil
	}
	return outcome(item, domain.OutcomeUnchanged), nil
}

// sourceAttributes は項目を取り込み元の属性へ写す。Directory Connector は manager を管理せず、
// sourceAnchor を管理する。
func sourceAttributes(item domain.UserItem) sourceuser.Attributes {
	return sourceuser.Attributes{
		PreferredUsername: item.PreferredUsername, Name: item.Name, GivenName: item.GivenName,
		FamilyName: item.FamilyName, Email: item.Email,
		EmployeeNumber: item.EmployeeNumber, Department: item.Department,
		OnPremisesImmutableID: new(item.OnPremisesImmutableID),
	}
}

// writeOutcome は IdManagement の拒否を項目の失敗へ写し、ほかのエラーはそのまま返す。
func writeOutcome(item domain.UserItem, success domain.ItemOutcome, err error) (domain.ItemResult, error) {
	var quota interface{ IsQuotaExceeded() bool }
	switch {
	case err == nil:
		return outcome(item, success), nil
	case errors.Is(err, directoryports.ErrUsernameConflict):
		return failed(item, domain.ItemErrorUsernameConflict), nil
	case errors.Is(err, directoryports.ErrEmailConflict):
		return failed(item, domain.ItemErrorEmailConflict), nil
	case errors.Is(err, directoryports.ErrInvalidAttribute):
		return failed(item, domain.ItemErrorInvalidAttribute), nil
	case errors.Is(err, directoryports.ErrAttributeConflict):
		return failed(item, domain.ItemErrorAttributeConflict), nil
	case errors.As(err, &quota) && quota.IsQuotaExceeded():
		return failed(item, domain.ItemErrorQuotaExceeded), nil
	default:
		return domain.ItemResult{}, err
	}
}

func outcome(item domain.UserItem, outcome domain.ItemOutcome) domain.ItemResult {
	return domain.ItemResult{ExternalID: item.ExternalID, Outcome: outcome}
}

func failed(item domain.UserItem, code string) domain.ItemResult {
	return domain.ItemResult{ExternalID: item.ExternalID, Outcome: domain.OutcomeFailed, Error: code}
}
