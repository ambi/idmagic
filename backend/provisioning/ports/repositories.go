// Package ports defines the Provisioning bounded context's repository
// abstractions (spec/contexts/provisioning.yaml). Implementations live in
// backend/provisioning/{db_memory,db_postgres}.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// ErrConnectionAlreadyExists is returned by Register when the Application
// already has a ProvisioningConnection (spec/contexts/provisioning.yaml
// errors.ProvisioningConnectionAlreadyExistsError, "1 Application 1 connection").
var ErrConnectionAlreadyExists = errors.New("provisioning: connection already exists for this application")

// ProvisioningConnectionRepository persists ProvisioningConnection aggregates.
// CredentialSecret is a narrow accessor separate from Find: only the provisioning
// engine (T006) may call it to authenticate outbound requests, so admin read
// paths (which use Find) never see the plaintext/opaque secret
// (spec/contexts/provisioning.yaml credential write-only 契約).
type ProvisioningConnectionRepository interface {
	// Register inserts a new connection together with its credential secret.
	// Returns ErrConnectionAlreadyExists if the Application already has one.
	Register(ctx context.Context, conn *domain.ProvisioningConnection, secret string) error
	// Update replaces the full connection record. secret is non-nil only when
	// the caller is rotating the credential (ProvisioningCredentialRotated).
	Update(ctx context.Context, conn *domain.ProvisioningConnection, secret *string) error
	Find(ctx context.Context, tenantID, applicationID string) (*domain.ProvisioningConnection, error)
	// CredentialSecret returns the plaintext/opaque secret for outbound calls.
	CredentialSecret(ctx context.Context, tenantID, applicationID string) (string, error)
	Delete(ctx context.Context, tenantID, applicationID string) error
	ListAll(ctx context.Context, tenantID string) ([]*domain.ProvisioningConnection, error)
	// ListTenantsWithActiveConnections は有効な接続を持つテナントを返す。照合がテナントを越えて接続を巡る入口である。
	ListTenantsWithActiveConnections(ctx context.Context) ([]string, error)
}

// RemoteResourceLinkRepository persists the correlation between an idmagic
// User/Group and its downstream SCIM resource.
type RemoteResourceLinkRepository interface {
	Find(ctx context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) (*domain.RemoteResourceLink, error)
	// Upsert inserts the link on first sync, or updates it on a later sync. The
	// caller applies RemoteResourceLink.ApplySync's monotonicity check before
	// calling Upsert; Upsert itself does not re-derive ordering.
	Upsert(ctx context.Context, link *domain.RemoteResourceLink) error
	// ListByConnection は接続の sourceType のリンクを source_id の順に返す。照合が反映済みの状態として読む。
	ListByConnection(ctx context.Context, tenantID, connectionID string, sourceType domain.ProvisioningSourceType) ([]*domain.RemoteResourceLink, error)
	// Delete はリンクを消す。下流からリソースを削除したあと、リンクなしが「下流に何もない」を表すようにする。
	Delete(ctx context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) error
}

// ProvisioningTaskRepository persists ProvisioningTask records.
type ProvisioningTaskRepository interface {
	// Save inserts a new task. It returns created=false without error when
	// an existing task already has the same idempotency key
	// (tenant_id, connection_id, source_type, source_id, source_version).
	Save(ctx context.Context, d *domain.ProvisioningTask) (created bool, err error)
	Find(ctx context.Context, tenantID, taskID string) (*domain.ProvisioningTask, error)
	// ListByConnection lists tasks for a connection, most recent first.
	// status filters to a single status when non-nil.
	ListByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, limit int) ([]*domain.ProvisioningTask, error)
	// ListPageByConnection returns up to limit tasks for a connection
	// ordered by (created_at, id) descending — matching ListByConnection's
	// pre-existing "most recent first" order, with id as tie-break — status
	// filters to a single status when non-nil. afterCreatedAt/afterID are the
	// keyset continuation cursor (wi-159); zero/"" for the first page.
	ListPageByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, sourceType *domain.ProvisioningSourceType, afterCreatedAt time.Time, afterID string, limit int) ([]*domain.ProvisioningTask, error)
	ListPageBeforeByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, sourceType *domain.ProvisioningSourceType, beforeCreatedAt time.Time, beforeID string, limit int) ([]*domain.ProvisioningTask, error)
	// ListUnenqueued returns pending tasks with no Jobs.Job associated yet
	// (dispatcher recovery, LifecycleWorkflowRunLifecycle precedent).
	ListUnenqueued(ctx context.Context, limit int) ([]*domain.ProvisioningTask, error)
	// AttachJob associates a Jobs.Job with a pending, unattached task and
	// moves it to in_flight (ProvisioningTaskLifecycle の pending → in_flight).
	// attached is false when the task is no longer pending or already has a job.
	AttachJob(ctx context.Context, tenantID, taskID, jobID string) (attached bool, err error)
	// UpdateStatus transitions a task's status and records the last error
	// (nil clears it). Callers are responsible for using a
	// domain.TransitionProvisioningTaskLifecycle-valid target status.
	UpdateStatus(ctx context.Context, tenantID, taskID string, status domain.ProvisioningTaskStatus, lastError *string) error
	// RetryDeadLetter resets a dead_letter task to pending and clears its
	// job_id so the dispatcher picks it up again. Returns false if the task
	// is not currently dead_letter.
	RetryDeadLetter(ctx context.Context, tenantID, taskID string) (bool, error)
	// ListUnsettledByConnection は接続の sourceType のタスクのうち succeeded 以外を返す。
	// 照合が、決着を待つべき対象を見分けるために読む。
	ListUnsettledByConnection(ctx context.Context, tenantID, connectionID string, sourceType domain.ProvisioningSourceType) ([]*domain.ProvisioningTask, error)

	// 猶予期間つき削除の予約はプロビジョニングタスクの前段であり、実体化で予約の遷移とプロビジョニングタスクの挿入を
	// 同時に行うため、プロビジョニングタスクと同じリポジトリが持つ。

	// ScheduleDeprovision は予約を保存する。同じ (tenant, connection, user) に scheduled の
	// 予約があれば、その期限を保つために作らず created=false を返す。
	ScheduleDeprovision(ctx context.Context, s *domain.ScheduledDeprovision) (created bool, err error)
	// CancelScheduledDeprovisions は (tenant, connection, user) の scheduled の予約のうち、
	// SourceVersion が beforeVersion より小さいものを cancelled にし、その件数を返す。
	CancelScheduledDeprovisions(ctx context.Context, tenantID, connectionID, userID string, beforeVersion int64, now time.Time) (int, error)
	// ListDueDeprovisions は全テナントから、now の時点で期限に達した scheduled の予約を期限の古い順に返す。
	ListDueDeprovisions(ctx context.Context, now time.Time, limit int) ([]*domain.ScheduledDeprovision, error)
	// MaterializeDeprovision は予約がまだ scheduled のときだけ materialized にし、d を挿入する。
	// 取消と競合して予約が scheduled でなくなっていれば、何も変えず false を返す。
	MaterializeDeprovision(ctx context.Context, s *domain.ScheduledDeprovision, d *domain.ProvisioningTask) (bool, error)
}
