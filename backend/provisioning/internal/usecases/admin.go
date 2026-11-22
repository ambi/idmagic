package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// AdminDeps are the admin (Application-detail "provisioning" subroute)
// usecases' dependencies (docs/modules/provisioning/ interfaces).
type AdminDeps struct {
	ConnectionRepo provisioningports.ProvisioningConnectionRepository
	TaskRepo       provisioningports.ProvisioningTaskRepository
	AssignmentRepo appports.AssignmentRepository
	UserRepo       userports.UserRepository
	// GroupRepo lets a full resync cover the Groups a push_groups connection
	// targets. nil resyncs Users only; a connection whose GroupPushConfig names
	// its groups explicitly still resyncs them, because that list needs no lookup.
	GroupRepo       groupports.GroupRepository
	NewTargetClient func(conn *provisioningdomain.ProvisioningConnection, secret string) (provisioningports.ProvisioningTargetClient, error)
	// Emit は接続の保存に成功した管理操作のイベントを発行する。nil なら発行しない。
	Emit func(spec.DomainEvent)
}

// InvalidInputError は、作用を持たない入力検証で識別した拒否だけを表す。
type InvalidInputError struct{ cause error }

func (e *InvalidInputError) Error() string { return e.cause.Error() }

func (d AdminDeps) captureDeps() CaptureDeps {
	return CaptureDeps{ConnectionRepo: d.ConnectionRepo, TaskRepo: d.TaskRepo, AssignmentRepo: d.AssignmentRepo}
}

// defaultSCIMUserMapping seeds the SCIM core User mapping table
// (docs/modules/provisioning/ §属性マッピングのセマンティクス 既定マッピング).
func defaultSCIMUserMapping() []provisioningdomain.AttributeMappingRule {
	return []provisioningdomain.AttributeMappingRule{
		{TargetPath: "externalId", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "id", ApplyOn: provisioningdomain.ApplyCreateOnly, Required: true},
		{TargetPath: "userName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: provisioningdomain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "active", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "active", ApplyOn: provisioningdomain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "name.givenName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "given_name", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
		{TargetPath: "name.familyName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "family_name", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
		{TargetPath: "displayName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "display_name", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
		{TargetPath: `emails[type eq "work"].value`, SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "email", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
	}
}

// RegisterConnectionInput is RegisterConnection's input.
type RegisterConnectionInput struct {
	TenantID      string
	ApplicationID string
	BaseURL       string
	Credential    provisioningdomain.ProvisioningCredentialInput
	Now           time.Time
}

// RegisterConnection creates a ProvisioningConnection with seeded defaults
// (docs/modules/provisioning/ interfaces.RegisterProvisioningConnection).
func RegisterConnection(ctx context.Context, deps AdminDeps, in RegisterConnectionInput) (*provisioningdomain.ProvisioningConnection, error) {
	if err := provisioningdomain.ValidateOutboundBaseURL(in.BaseURL); err != nil {
		return nil, &InvalidInputError{cause: err}
	}
	if err := validateCredential(in.Credential); err != nil {
		return nil, &InvalidInputError{cause: err}
	}
	credentialID, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	now := in.Now.UTC()
	conn := &provisioningdomain.ProvisioningConnection{
		ApplicationID:                     in.ApplicationID,
		TenantID:                          in.TenantID,
		Status:                            provisioningdomain.ConnectionActive,
		BaseURL:                           in.BaseURL,
		Credential:                        credentialMetadata(credentialID, in.Credential, now, nil),
		FeatureFlags:                      provisioningdomain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true},
		Scope:                             provisioningdomain.ScopeAssignedOnly,
		AttributeMappings:                 defaultSCIMUserMapping(),
		Matching:                          provisioningdomain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy:                 provisioningdomain.DeprovisionPolicy{OnUnassign: provisioningdomain.DeprovisionDeactivate, OnDelete: provisioningdomain.DeprovisionDeactivate, OnGroupDeletedOrUnassigned: provisioningdomain.DeprovisionNone},
		RateLimitPerMinute:                60,
		MaxAttempts:                       8,
		QuarantineAfterConsecutiveFailure: 10,
		Health:                            provisioningdomain.HealthOK,
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
	if err := conn.Validate(); err != nil {
		return nil, &InvalidInputError{cause: err}
	}
	if err := deps.ConnectionRepo.Register(ctx, conn, in.Credential.Secret()); err != nil {
		return nil, err
	}
	emit(deps.Emit, &provisioningdomain.ProvisioningConnectionRegistered{At: now, TenantID: conn.TenantID, ApplicationID: conn.ApplicationID})
	return conn, nil
}

// UpdateConnectionInput is UpdateConnection's input; nil fields are left
// unchanged (partial update).
type UpdateConnectionInput struct {
	TenantID                          string
	ApplicationID                     string
	BaseURL                           *string
	Status                            *provisioningdomain.ProvisioningConnectionStatus
	Credential                        *provisioningdomain.ProvisioningCredentialInput
	FeatureFlags                      *provisioningdomain.ProvisioningFeatureFlags
	Scope                             *provisioningdomain.ProvisioningScope
	GroupPush                         *provisioningdomain.GroupPushConfig
	AttributeMappings                 *[]provisioningdomain.AttributeMappingRule
	Matching                          *provisioningdomain.MatchingRule
	DeprovisionPolicy                 *provisioningdomain.DeprovisionPolicy
	RateLimitPerMinute                *int
	MaxAttempts                       *int
	NotificationEmail                 *string
	QuarantineAfterConsecutiveFailure *int
	Now                               time.Time
}

// UpdateConnection applies a partial update to an existing connection
// (docs/modules/provisioning/ interfaces.UpdateProvisioningConnection).
func UpdateConnection(ctx context.Context, deps AdminDeps, in UpdateConnectionInput) (*provisioningdomain.ProvisioningConnection, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, in.TenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	wasDisabled := conn.Status == provisioningdomain.ConnectionDisabled
	if in.BaseURL != nil {
		if err := provisioningdomain.ValidateOutboundBaseURL(*in.BaseURL); err != nil {
			return nil, &InvalidInputError{cause: err}
		}
		conn.BaseURL = *in.BaseURL
	}
	if in.Status != nil {
		conn.Status = *in.Status
	}
	if in.FeatureFlags != nil {
		conn.FeatureFlags = *in.FeatureFlags
	}
	if in.Scope != nil {
		conn.Scope = *in.Scope
	}
	if in.GroupPush != nil {
		conn.GroupPush = in.GroupPush
	}
	if in.AttributeMappings != nil {
		conn.AttributeMappings = *in.AttributeMappings
	}
	if in.Matching != nil {
		conn.Matching = *in.Matching
	}
	if in.DeprovisionPolicy != nil {
		conn.DeprovisionPolicy = *in.DeprovisionPolicy
	}
	if in.RateLimitPerMinute != nil {
		conn.RateLimitPerMinute = *in.RateLimitPerMinute
	}
	if in.MaxAttempts != nil {
		conn.MaxAttempts = *in.MaxAttempts
	}
	if in.NotificationEmail != nil {
		conn.NotificationEmail = in.NotificationEmail
	}
	if in.QuarantineAfterConsecutiveFailure != nil {
		conn.QuarantineAfterConsecutiveFailure = *in.QuarantineAfterConsecutiveFailure
	}
	conn.UpdatedAt = in.Now.UTC()
	if err := conn.Validate(); err != nil {
		return nil, &InvalidInputError{cause: err}
	}
	var secret *string
	if in.Credential != nil {
		if err := validateCredential(*in.Credential); err != nil {
			return nil, &InvalidInputError{cause: err}
		}
		credentialID, err := spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
		conn.Credential = credentialMetadata(credentialID, *in.Credential, conn.Credential.CreatedAt, &conn.UpdatedAt)
		s := in.Credential.Secret()
		secret = &s
	}
	if err := deps.ConnectionRepo.Update(ctx, conn, secret); err != nil {
		return nil, err
	}
	isDisabled := conn.Status == provisioningdomain.ConnectionDisabled
	if in.changesSettings() || (wasDisabled && !isDisabled) {
		emit(deps.Emit, &provisioningdomain.ProvisioningConnectionUpdated{At: conn.UpdatedAt, TenantID: conn.TenantID, ApplicationID: conn.ApplicationID})
	}
	if !wasDisabled && isDisabled {
		emit(deps.Emit, &provisioningdomain.ProvisioningConnectionDisabled{At: conn.UpdatedAt, TenantID: conn.TenantID, ApplicationID: conn.ApplicationID})
	}
	if secret != nil {
		emit(deps.Emit, &provisioningdomain.ProvisioningCredentialRotated{
			At: conn.UpdatedAt, TenantID: conn.TenantID, ApplicationID: conn.ApplicationID, CredentialID: conn.Credential.CredentialID,
		})
	}
	return conn, nil
}

// changesSettings は、資格情報と status 以外の設定を指定しているかを返す。
// 値が保存済みと同じでも、指定したこと自体を管理者の変更として記録する。
func (in UpdateConnectionInput) changesSettings() bool {
	return in.BaseURL != nil || in.FeatureFlags != nil || in.Scope != nil || in.GroupPush != nil ||
		in.AttributeMappings != nil || in.Matching != nil || in.DeprovisionPolicy != nil ||
		in.RateLimitPerMinute != nil || in.MaxAttempts != nil || in.NotificationEmail != nil ||
		in.QuarantineAfterConsecutiveFailure != nil
}

func validateCredential(c provisioningdomain.ProvisioningCredentialInput) error {
	if !c.AuthMethod.Valid() {
		return errors.New("provisioning: invalid auth_method")
	}
	if c.AuthMethod == provisioningdomain.AuthBearerToken && c.BearerToken == "" {
		return errors.New("provisioning: bearer_token is required for auth_method=bearer_token")
	}
	if c.AuthMethod == provisioningdomain.AuthOAuth2ClientCredentials && (c.OAuth2TokenURL == "" || c.OAuth2ClientID == "" || c.OAuth2ClientSecret == "") {
		return errors.New("provisioning: oauth2 credential fields are required for auth_method=oauth2_client_credentials")
	}
	return nil
}

// GetConnection returns the connection for applicationID, or nil if none exists.
func GetConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string) (*provisioningdomain.ProvisioningConnection, error) {
	return deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
}

// DeleteConnection removes the connection (hard delete, Application precedent).
func DeleteConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) error {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrConnectionNotFound
	}
	if err := deps.ConnectionRepo.Delete(ctx, tenantID, applicationID); err != nil {
		return err
	}
	emit(deps.Emit, &provisioningdomain.ProvisioningConnectionDeleted{At: now.UTC(), TenantID: tenantID, ApplicationID: applicationID})
	return nil
}

// TestConnectionResult is TestConnection's output.
type TestConnectionResult struct {
	Reachable    bool                                         `json:"reachable"`
	Capabilities *provisioningdomain.ProvisioningCapabilities `json:"capabilities,omitempty"`
	Error        string                                       `json:"error,omitempty"`
}

// TestConnection calls the downstream /ServiceProviderConfig and persists the
// discovered capabilities (docs/modules/provisioning/
// interfaces.TestProvisioningConnection).
func TestConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (TestConnectionResult, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return TestConnectionResult{}, err
	}
	if conn == nil {
		return TestConnectionResult{}, ErrConnectionNotFound
	}
	secret, err := deps.ConnectionRepo.CredentialSecret(ctx, tenantID, applicationID)
	if err != nil {
		return TestConnectionResult{}, err
	}
	client, err := deps.NewTargetClient(conn, secret)
	if err != nil {
		return TestConnectionResult{}, err
	}
	caps, discoverErr := client.Discover(ctx)
	if discoverErr != nil {
		logging.Error(ctx, "provisioning discovery failed", "error", discoverErr.Error(), "application_id", applicationID)
		// 下流へ到達できないことは接続テストの結果として返す。
		return TestConnectionResult{Reachable: false, Error: "The provisioning target could not be reached."}, nil
	}
	caps.DiscoveredAt = now.UTC()
	conn.Capabilities = &caps
	conn.UpdatedAt = now.UTC()
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return TestConnectionResult{}, err
	}
	return TestConnectionResult{Reachable: true, Capabilities: &caps}, nil
}

var ErrSubjectNotInScope = errors.New("provisioning: subject is not in this connection's scope")

// ProvisionOnDemand creates an immediate pending task for a single subject
// (docs/modules/provisioning/ interfaces.ProvisionOnDemand).
func ProvisionOnDemand(ctx context.Context, deps AdminDeps, tenantID, applicationID string, sourceType provisioningdomain.ProvisioningSourceType, sourceID string, now time.Time) (*provisioningdomain.ProvisioningTask, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	if conn.Health == provisioningdomain.HealthQuarantined {
		return nil, ErrConnectionQuarantined
	}
	if sourceType == provisioningdomain.SourceTypeUser {
		ok, err := inScope(ctx, deps.captureDeps(), *conn, tenantID, sourceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrSubjectNotInScope
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	task := &provisioningdomain.ProvisioningTask{
		ID: id, TenantID: tenantID, ConnectionID: applicationID, SourceType: sourceType, SourceID: sourceID,
		SourceVersion: now.UnixNano(), Operation: provisioningdomain.OperationUpdate, Status: provisioningdomain.TaskPending,
		CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if _, err := deps.TaskRepo.Save(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// StartFullResync enqueues an update task for every subject in the
// connection's scope (docs/modules/provisioning/ interfaces.StartFullResync).
// 対象数を確定した FullResync を先に保存し、各プロビジョニングタスクをそれへ関連付ける。
// FullResyncCompleted は最後のプロビジョニングタスクを終端にしたジョブが発行し、対象が 0 件ならここで発行する
// (docs/modules/provisioning/features/connection.md)。
func StartFullResync(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (int, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectionNotFound
	}
	if conn.Health == provisioningdomain.HealthQuarantined {
		return 0, ErrConnectionQuarantined
	}
	var subjectIDs []string
	if conn.Scope == provisioningdomain.ScopeAllUsers {
		if deps.UserRepo == nil {
			return 0, errors.New("provisioning: all_users resync requires UserRepo")
		}
		users, err := deps.UserRepo.FindAll(ctx, tenantID)
		if err != nil {
			return 0, err
		}
		for _, u := range users {
			subjectIDs = append(subjectIDs, u.ID)
		}
	} else if deps.AssignmentRepo != nil {
		assignments, err := deps.AssignmentRepo.ListByApplication(ctx, tenantID, applicationID)
		if err != nil {
			return 0, err
		}
		for _, a := range assignments {
			if a.SubjectType == "user" {
				subjectIDs = append(subjectIDs, a.SubjectID)
			}
		}
	}
	groupIDs, err := resyncGroupIDs(ctx, deps, *conn, tenantID)
	if err != nil {
		return 0, err
	}
	// 同じ subject を二度数えると、冪等キーで作られないプロビジョニングタスクの分だけ対象数に届かず完了しなくなる。
	slices.Sort(subjectIDs)
	subjectIDs = slices.Compact(subjectIDs)
	slices.Sort(groupIDs)
	groupIDs = slices.Compact(groupIDs)
	resyncID, err := spec.NewUUIDv4()
	if err != nil {
		return 0, err
	}
	resync := provisioningdomain.NewFullResync(resyncID, tenantID, applicationID, len(subjectIDs)+len(groupIDs), now)
	if err := deps.TaskRepo.SaveFullResync(ctx, resync); err != nil {
		return 0, err
	}

	enqueued := 0
	enqueue := func(sourceType provisioningdomain.ProvisioningSourceType, ids []string) error {
		for _, sourceID := range ids {
			id, err := spec.NewUUIDv4()
			if err != nil {
				return err
			}
			task := &provisioningdomain.ProvisioningTask{
				ID: id, TenantID: tenantID, ConnectionID: applicationID, SourceType: sourceType, SourceID: sourceID,
				SourceVersion: now.UnixNano(), Operation: provisioningdomain.OperationUpdate, Status: provisioningdomain.TaskPending,
				CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
			}
			created, err := deps.TaskRepo.SaveFullResyncTask(ctx, resync.ID, task)
			if err != nil {
				return err
			}
			if created {
				enqueued++
			}
		}
		return nil
	}
	if err := enqueue(provisioningdomain.SourceTypeUser, subjectIDs); err != nil {
		return enqueued, err
	}
	if err := enqueue(provisioningdomain.SourceTypeGroup, groupIDs); err != nil {
		return enqueued, err
	}
	conn.LastFullSyncAt = new(now.UTC())
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return enqueued, err
	}
	// 対象が 0 件のとき、または作り終える前にすべて終端になったときは、ここで完了する。
	if err := settleFullResync(ctx, deps.TaskRepo, deps.Emit, resync, now); err != nil {
		return enqueued, err
	}
	return enqueued, nil
}

// ListTasks lists a connection's tasks, most recent first.
// afterCreatedAt/afterID are the keyset continuation cursor (wi-159)
// ; zero/"" for the first page.
func ListTasks(ctx context.Context, deps AdminDeps, tenantID, applicationID string, status *provisioningdomain.ProvisioningTaskStatus, sourceType *provisioningdomain.ProvisioningSourceType, afterCreatedAt time.Time, afterID string, limit int) ([]*provisioningdomain.ProvisioningTask, error) {
	if limit <= 0 {
		limit = 50
	}
	return deps.TaskRepo.ListPageByConnection(ctx, tenantID, applicationID, status, sourceType, afterCreatedAt, afterID, limit)
}

func ListTasksBefore(ctx context.Context, deps AdminDeps, tenantID, applicationID string, status *provisioningdomain.ProvisioningTaskStatus, sourceType *provisioningdomain.ProvisioningSourceType, beforeCreatedAt time.Time, beforeID string, limit int) ([]*provisioningdomain.ProvisioningTask, error) {
	if limit <= 0 {
		limit = 50
	}
	return deps.TaskRepo.ListPageBeforeByConnection(ctx, tenantID, applicationID, status, sourceType, beforeCreatedAt, beforeID, limit)
}

// GetTask returns one task, verifying it belongs to applicationID.
func GetTask(ctx context.Context, deps AdminDeps, tenantID, applicationID, taskID string) (*provisioningdomain.ProvisioningTask, error) {
	d, err := deps.TaskRepo.Find(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	if d == nil || d.ConnectionID != applicationID {
		return nil, ErrTaskNotFound
	}
	return d, nil
}

var ErrTaskNotRetryable = errors.New("provisioning: task is not dead_letter")

// RetryTask resets a dead_letter task to pending
// (docs/modules/provisioning/ interfaces.RetryProvisioningTask).
func RetryTask(ctx context.Context, deps AdminDeps, tenantID, applicationID, taskID string) (*provisioningdomain.ProvisioningTask, error) {
	d, err := GetTask(ctx, deps, tenantID, applicationID, taskID)
	if err != nil {
		return nil, err
	}
	if d.Status != provisioningdomain.TaskDeadLetter {
		return nil, ErrTaskNotRetryable
	}
	retried, err := deps.TaskRepo.RetryDeadLetter(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	if !retried {
		return nil, ErrTaskNotRetryable
	}
	return deps.TaskRepo.Find(ctx, tenantID, taskID)
}

// ResumeConnection clears quarantine (docs/modules/provisioning/
// interfaces.ResumeProvisioningConnection).
func ResumeConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (*provisioningdomain.ProvisioningConnection, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	if err := conn.Resume(now); err != nil {
		return nil, err
	}
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return nil, err
	}
	emit(deps.Emit, &provisioningdomain.ProvisioningConnectionQuarantineCleared{At: now.UTC(), TenantID: conn.TenantID, ApplicationID: conn.ApplicationID})
	return conn, nil
}

// ListTenantConnections lists every connection in the tenant
// (docs/modules/provisioning/ interfaces.ListTenantProvisioningConnections).
func ListTenantConnections(ctx context.Context, deps AdminDeps, tenantID string) ([]*provisioningdomain.ProvisioningConnection, error) {
	return deps.ConnectionRepo.ListAll(ctx, tenantID)
}

// credentialMetadata は書き込み専用の資格情報入力から、保存する非秘密の投影を作る。
// client_secret は含めない —— それは Secret() が返し、エンベロープ暗号化して
// credential_secret へ保存する。トークン URL / client_id / scope は秘密ではないが、
// トークン取得に必要なので保存しなければならない (wi-440 以前は検証だけして捨てていた)。
func credentialMetadata(
	credentialID string,
	in provisioningdomain.ProvisioningCredentialInput,
	createdAt time.Time,
	rotatedAt *time.Time,
) provisioningdomain.ProvisioningConnectionCredentialMetadata {
	metadata := provisioningdomain.ProvisioningConnectionCredentialMetadata{
		CredentialID: credentialID, AuthMethod: in.AuthMethod,
		CreatedAt: createdAt, RotatedAt: rotatedAt,
	}
	if in.AuthMethod == provisioningdomain.AuthOAuth2ClientCredentials {
		metadata.OAuth2TokenURL = in.OAuth2TokenURL
		metadata.OAuth2ClientID = in.OAuth2ClientID
		metadata.OAuth2Scope = in.OAuth2Scope
	}
	return metadata
}

// resyncGroupIDs returns the Groups a full resync should re-send for conn.
//
// It applies the same two guards capture does, in the same order: the
// push_groups feature flag, then the GroupPushConfig selection. An unset config
// yields nothing, so enabling the capability without choosing targets does not
// start writing every Group downstream.
func resyncGroupIDs(
	ctx context.Context,
	deps AdminDeps,
	conn provisioningdomain.ProvisioningConnection,
	tenantID string,
) ([]string, error) {
	if !conn.FeatureFlags.PushGroups || conn.GroupPush == nil {
		return nil, nil
	}
	if conn.GroupPush.Selection == provisioningdomain.GroupSelectionExplicit {
		// 明示指定は一覧を引く必要が無い。GroupRepo が未配線でも再同期できる。
		return slices.Clone(conn.GroupPush.ExplicitGroupIDs), nil
	}
	if deps.GroupRepo == nil {
		return nil, nil
	}
	groups, err := deps.GroupRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		if groupInScope(conn, group.ID) {
			ids = append(ids, group.ID)
		}
	}
	return ids, nil
}
