// Package postgres implements the Provisioning bounded context's repositories
// on PostgreSQL using hand-written SQL via pgx (LifecycleWorkflowRunRepository
// precedent; sqlc is not required for every context). credential_secret
// is stored as plaintext for now (dev/test grade, see infra/schema/postgres.sql
// comment and wi-97 envelope-encryption-at-rest).
package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

func pgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func pgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func pgUUID(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	_ = u.Scan(*s)
	return u
}

func pgUUIDVal(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func fromPgUUID(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := sharedpg.UUIDString(u)
	return &s
}

// ProvisioningConnectionRepository is the PostgreSQL ports.ProvisioningConnectionRepository.
type ProvisioningConnectionRepository struct{ Pool sharedpg.DB }

var _ ports.ProvisioningConnectionRepository = (*ProvisioningConnectionRepository)(nil)

func (r *ProvisioningConnectionRepository) Register(ctx context.Context, conn *domain.ProvisioningConnection, secret string) error {
	if err := conn.Validate(); err != nil {
		return err
	}
	j, err := marshalConnectionJSON(conn)
	if err != nil {
		return err
	}
	_, err = New(r.Pool).InsertProvisioningConnection(ctx, InsertProvisioningConnectionParams{
		ApplicationID:                      conn.ApplicationID,
		TenantID:                           conn.TenantID,
		Status:                             string(conn.Status),
		BaseUrl:                            conn.BaseURL,
		CredentialID:                       conn.Credential.CredentialID,
		AuthMethod:                         string(conn.Credential.AuthMethod),
		CredentialSecret:                   secret,
		CredentialOauth2TokenUrl:           conn.Credential.OAuth2TokenURL,
		CredentialOauth2ClientID:           conn.Credential.OAuth2ClientID,
		CredentialOauth2Scope:              conn.Credential.OAuth2Scope,
		CredentialCreatedAt:                conn.Credential.CreatedAt,
		CredentialRotatedAt:                pgTimestamptz(conn.Credential.RotatedAt),
		Capabilities:                       j.capabilities,
		FeatureFlags:                       j.featureFlags,
		Scope:                              string(conn.Scope),
		GroupPush:                          j.groupPush,
		AttributeMappings:                  j.mappings,
		Matching:                           j.matching,
		DeprovisionPolicy:                  j.deprovision,
		RateLimitPerMinute:                 int32(conn.RateLimitPerMinute), //nolint:gosec // safe downcast
		MaxAttempts:                        int32(conn.MaxAttempts),        //nolint:gosec // safe downcast
		NotificationEmail:                  pgText(conn.NotificationEmail),
		QuarantineAfterConsecutiveFailures: int32(conn.QuarantineAfterConsecutiveFailure), //nolint:gosec // safe downcast
		Health:                             string(conn.Health),
		ConsecutiveFailureCount:            int32(conn.ConsecutiveFailureCount), //nolint:gosec // safe downcast
		LastFullSyncAt:                     pgTimestamptz(conn.LastFullSyncAt),
		QuarantinedAt:                      pgTimestamptz(conn.QuarantinedAt),
		QuarantineReason:                   pgText(conn.QuarantineReason),
		CreatedAt:                          conn.CreatedAt,
		UpdatedAt:                          conn.UpdatedAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrConnectionAlreadyExists
	}
	return err
}

func (r *ProvisioningConnectionRepository) Update(ctx context.Context, conn *domain.ProvisioningConnection, secret *string) error {
	if err := conn.Validate(); err != nil {
		return err
	}
	j, err := marshalConnectionJSON(conn)
	if err != nil {
		return err
	}
	if secret != nil {
		err = New(r.Pool).UpdateProvisioningConnectionWithSecret(ctx, UpdateProvisioningConnectionWithSecretParams{
			TenantID:                           conn.TenantID,
			ApplicationID:                      conn.ApplicationID,
			Status:                             string(conn.Status),
			BaseUrl:                            conn.BaseURL,
			CredentialID:                       conn.Credential.CredentialID,
			AuthMethod:                         string(conn.Credential.AuthMethod),
			CredentialSecret:                   *secret,
			CredentialOauth2TokenUrl:           conn.Credential.OAuth2TokenURL,
			CredentialOauth2ClientID:           conn.Credential.OAuth2ClientID,
			CredentialOauth2Scope:              conn.Credential.OAuth2Scope,
			CredentialRotatedAt:                pgTimestamptz(conn.Credential.RotatedAt),
			Capabilities:                       j.capabilities,
			FeatureFlags:                       j.featureFlags,
			Scope:                              string(conn.Scope),
			GroupPush:                          j.groupPush,
			AttributeMappings:                  j.mappings,
			Matching:                           j.matching,
			DeprovisionPolicy:                  j.deprovision,
			RateLimitPerMinute:                 int32(conn.RateLimitPerMinute), //nolint:gosec // safe downcast
			MaxAttempts:                        int32(conn.MaxAttempts),        //nolint:gosec // safe downcast
			NotificationEmail:                  pgText(conn.NotificationEmail),
			QuarantineAfterConsecutiveFailures: int32(conn.QuarantineAfterConsecutiveFailure), //nolint:gosec // safe downcast
			Health:                             string(conn.Health),
			ConsecutiveFailureCount:            int32(conn.ConsecutiveFailureCount), //nolint:gosec // safe downcast
			LastFullSyncAt:                     pgTimestamptz(conn.LastFullSyncAt),
			QuarantinedAt:                      pgTimestamptz(conn.QuarantinedAt),
			QuarantineReason:                   pgText(conn.QuarantineReason),
			UpdatedAt:                          conn.UpdatedAt,
		})
		return err
	}
	err = New(r.Pool).UpdateProvisioningConnection(ctx, UpdateProvisioningConnectionParams{
		TenantID:                           conn.TenantID,
		ApplicationID:                      conn.ApplicationID,
		Status:                             string(conn.Status),
		BaseUrl:                            conn.BaseURL,
		Capabilities:                       j.capabilities,
		FeatureFlags:                       j.featureFlags,
		Scope:                              string(conn.Scope),
		GroupPush:                          j.groupPush,
		AttributeMappings:                  j.mappings,
		Matching:                           j.matching,
		DeprovisionPolicy:                  j.deprovision,
		RateLimitPerMinute:                 int32(conn.RateLimitPerMinute), //nolint:gosec // safe downcast
		MaxAttempts:                        int32(conn.MaxAttempts),        //nolint:gosec // safe downcast
		NotificationEmail:                  pgText(conn.NotificationEmail),
		QuarantineAfterConsecutiveFailures: int32(conn.QuarantineAfterConsecutiveFailure), //nolint:gosec // safe downcast
		Health:                             string(conn.Health),
		ConsecutiveFailureCount:            int32(conn.ConsecutiveFailureCount), //nolint:gosec // safe downcast
		LastFullSyncAt:                     pgTimestamptz(conn.LastFullSyncAt),
		QuarantinedAt:                      pgTimestamptz(conn.QuarantinedAt),
		QuarantineReason:                   pgText(conn.QuarantineReason),
		UpdatedAt:                          conn.UpdatedAt,
	})
	return err
}

// connectionJSON holds a ProvisioningConnection's JSONB column payloads.
// Grouped into a struct (rather than many named results) to stay under
// gocritic's function result count limit.
type connectionJSON struct {
	featureFlags, capabilities, groupPush, mappings, matching, deprovision []byte
}

func marshalConnectionJSON(conn *domain.ProvisioningConnection) (connectionJSON, error) {
	var j connectionJSON
	var err error
	if j.featureFlags, err = json.Marshal(conn.FeatureFlags); err != nil {
		return j, err
	}
	if conn.Capabilities != nil {
		if j.capabilities, err = json.Marshal(conn.Capabilities); err != nil {
			return j, err
		}
	}
	if conn.GroupPush != nil {
		if j.groupPush, err = json.Marshal(conn.GroupPush); err != nil {
			return j, err
		}
	}
	if j.mappings, err = json.Marshal(conn.AttributeMappings); err != nil {
		return j, err
	}
	if j.matching, err = json.Marshal(conn.Matching); err != nil {
		return j, err
	}
	if j.deprovision, err = json.Marshal(conn.DeprovisionPolicy); err != nil {
		return j, err
	}
	return j, nil
}

func mapConnection(c *domain.ProvisioningConnection, featureFlagsRaw, capabilitiesRaw, groupPushRaw, mappingsRaw, matchingRaw, deprovisionRaw []byte) error {
	if err := json.Unmarshal(featureFlagsRaw, &c.FeatureFlags); err != nil {
		return err
	}
	if len(capabilitiesRaw) > 0 {
		var caps domain.ProvisioningCapabilities
		if err := json.Unmarshal(capabilitiesRaw, &caps); err != nil {
			return err
		}
		c.Capabilities = &caps
	}
	if len(groupPushRaw) > 0 {
		var gp domain.GroupPushConfig
		if err := json.Unmarshal(groupPushRaw, &gp); err != nil {
			return err
		}
		c.GroupPush = &gp
	}
	if err := json.Unmarshal(mappingsRaw, &c.AttributeMappings); err != nil {
		return err
	}
	if err := json.Unmarshal(matchingRaw, &c.Matching); err != nil {
		return err
	}
	if err := json.Unmarshal(deprovisionRaw, &c.DeprovisionPolicy); err != nil {
		return err
	}
	return c.Validate()
}

func mapFindConnection(row *FindProvisioningConnectionRow) (*domain.ProvisioningConnection, error) {
	c := &domain.ProvisioningConnection{
		ApplicationID: row.ApplicationID,
		TenantID:      row.TenantID,
		Status:        domain.ProvisioningConnectionStatus(row.Status),
		BaseURL:       row.BaseUrl,
		Credential: domain.ProvisioningConnectionCredentialMetadata{
			CredentialID:   row.CredentialID,
			AuthMethod:     domain.ProvisioningAuthMethod(row.AuthMethod),
			OAuth2TokenURL: row.CredentialOauth2TokenUrl,
			OAuth2ClientID: row.CredentialOauth2ClientID,
			OAuth2Scope:    row.CredentialOauth2Scope,
			CreatedAt:      row.CredentialCreatedAt,
			RotatedAt:      fromPgTimestamptz(row.CredentialRotatedAt),
		},
		Scope:                             domain.ProvisioningScope(row.Scope),
		RateLimitPerMinute:                int(row.RateLimitPerMinute),
		MaxAttempts:                       int(row.MaxAttempts),
		NotificationEmail:                 fromPgText(row.NotificationEmail),
		QuarantineAfterConsecutiveFailure: int(row.QuarantineAfterConsecutiveFailures),
		Health:                            domain.ProvisioningHealth(row.Health),
		ConsecutiveFailureCount:           int(row.ConsecutiveFailureCount),
		LastFullSyncAt:                    fromPgTimestamptz(row.LastFullSyncAt),
		QuarantinedAt:                     fromPgTimestamptz(row.QuarantinedAt),
		QuarantineReason:                  fromPgText(row.QuarantineReason),
		CreatedAt:                         row.CreatedAt,
		UpdatedAt:                         row.UpdatedAt,
	}
	if err := mapConnection(c, row.FeatureFlags, row.Capabilities, row.GroupPush, row.AttributeMappings, row.Matching, row.DeprovisionPolicy); err != nil {
		return nil, err
	}
	return c, nil
}

func mapListConnection(row *ListProvisioningConnectionsByTenantRow) (*domain.ProvisioningConnection, error) {
	c := &domain.ProvisioningConnection{
		ApplicationID: row.ApplicationID,
		TenantID:      row.TenantID,
		Status:        domain.ProvisioningConnectionStatus(row.Status),
		BaseURL:       row.BaseUrl,
		Credential: domain.ProvisioningConnectionCredentialMetadata{
			CredentialID:   row.CredentialID,
			AuthMethod:     domain.ProvisioningAuthMethod(row.AuthMethod),
			OAuth2TokenURL: row.CredentialOauth2TokenUrl,
			OAuth2ClientID: row.CredentialOauth2ClientID,
			OAuth2Scope:    row.CredentialOauth2Scope,
			CreatedAt:      row.CredentialCreatedAt,
			RotatedAt:      fromPgTimestamptz(row.CredentialRotatedAt),
		},
		Scope:                             domain.ProvisioningScope(row.Scope),
		RateLimitPerMinute:                int(row.RateLimitPerMinute),
		MaxAttempts:                       int(row.MaxAttempts),
		NotificationEmail:                 fromPgText(row.NotificationEmail),
		QuarantineAfterConsecutiveFailure: int(row.QuarantineAfterConsecutiveFailures),
		Health:                            domain.ProvisioningHealth(row.Health),
		ConsecutiveFailureCount:           int(row.ConsecutiveFailureCount),
		LastFullSyncAt:                    fromPgTimestamptz(row.LastFullSyncAt),
		QuarantinedAt:                     fromPgTimestamptz(row.QuarantinedAt),
		QuarantineReason:                  fromPgText(row.QuarantineReason),
		CreatedAt:                         row.CreatedAt,
		UpdatedAt:                         row.UpdatedAt,
	}
	if err := mapConnection(c, row.FeatureFlags, row.Capabilities, row.GroupPush, row.AttributeMappings, row.Matching, row.DeprovisionPolicy); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ProvisioningConnectionRepository) Find(ctx context.Context, tenantID, applicationID string) (*domain.ProvisioningConnection, error) {
	row, err := New(r.Pool).FindProvisioningConnection(ctx, FindProvisioningConnectionParams{
		TenantID:      tenantID,
		ApplicationID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapFindConnection(row)
}

func (r *ProvisioningConnectionRepository) CredentialSecret(ctx context.Context, tenantID, applicationID string) (string, error) {
	secret, err := New(r.Pool).GetProvisioningConnectionSecret(ctx, GetProvisioningConnectionSecretParams{
		TenantID:      tenantID,
		ApplicationID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return secret, err
}

func (r *ProvisioningConnectionRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	return New(r.Pool).DeleteProvisioningConnection(ctx, DeleteProvisioningConnectionParams{
		TenantID:      tenantID,
		ApplicationID: applicationID,
	})
}

func (r *ProvisioningConnectionRepository) ListAll(ctx context.Context, tenantID string) ([]*domain.ProvisioningConnection, error) {
	rows, err := New(r.Pool).ListProvisioningConnectionsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ProvisioningConnection, 0, len(rows))
	for _, row := range rows {
		conn, err := mapListConnection(row)
		if err != nil {
			return nil, err
		}
		out = append(out, conn)
	}
	return out, nil
}

// RemoteResourceLinkRepository is the PostgreSQL ports.RemoteResourceLinkRepository.
type RemoteResourceLinkRepository struct{ Pool sharedpg.DB }

var _ ports.RemoteResourceLinkRepository = (*RemoteResourceLinkRepository)(nil)

func (r *RemoteResourceLinkRepository) Find(ctx context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) (*domain.RemoteResourceLink, error) {
	row, err := New(r.Pool).FindRemoteResourceLink(ctx, FindRemoteResourceLinkParams{
		ConnectionID: connectionID,
		SourceType:   string(sourceType),
		SourceID:     sourceID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.RemoteResourceLink{
		ConnectionID:      row.ConnectionID,
		TenantID:          row.TenantID,
		SourceType:        domain.ProvisioningSourceType(row.SourceType),
		SourceID:          row.SourceID,
		RemoteID:          row.RemoteID,
		ExternalID:        row.ExternalID,
		ETag:              fromPgText(row.Etag),
		LastSyncedVersion: row.LastSyncedVersion,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func (r *RemoteResourceLinkRepository) Upsert(ctx context.Context, link *domain.RemoteResourceLink) error {
	return New(r.Pool).UpsertRemoteResourceLink(ctx, UpsertRemoteResourceLinkParams{
		ConnectionID:      link.ConnectionID,
		TenantID:          link.TenantID,
		SourceType:        string(link.SourceType),
		SourceID:          link.SourceID,
		RemoteID:          link.RemoteID,
		ExternalID:        link.ExternalID,
		Etag:              pgText(link.ETag),
		LastSyncedVersion: link.LastSyncedVersion,
		UpdatedAt:         link.UpdatedAt,
	})
}

// ProvisioningDeliveryRepository is the PostgreSQL ports.ProvisioningDeliveryRepository.
type ProvisioningDeliveryRepository struct{ Pool sharedpg.DB }

var _ ports.ProvisioningDeliveryRepository = (*ProvisioningDeliveryRepository)(nil)

func mapDelivery(row *ProvisioningDelivery) (*domain.ProvisioningDelivery, error) {
	d := &domain.ProvisioningDelivery{
		ID:            row.ID,
		TenantID:      row.TenantID,
		ConnectionID:  row.ConnectionID,
		SourceType:    domain.ProvisioningSourceType(row.SourceType),
		SourceID:      row.SourceID,
		SourceVersion: row.SourceVersion,
		Operation:     domain.ProvisioningOperation(row.Operation),
		Status:        domain.ProvisioningDeliveryStatus(row.Status),
		JobID:         fromPgUUID(row.JobID),
		LastError:     fromPgText(row.LastError),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		CompletedAt:   fromPgTimestamptz(row.CompletedAt),
	}
	return d, d.Validate()
}

func (r *ProvisioningDeliveryRepository) Save(ctx context.Context, d *domain.ProvisioningDelivery) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, err
	}
	_, err := New(r.Pool).InsertProvisioningDelivery(ctx, InsertProvisioningDeliveryParams{
		ID:            d.ID,
		TenantID:      d.TenantID,
		ConnectionID:  d.ConnectionID,
		SourceType:    string(d.SourceType),
		SourceID:      d.SourceID,
		SourceVersion: d.SourceVersion,
		Operation:     string(d.Operation),
		Status:        string(d.Status),
		JobID:         pgUUID(d.JobID),
		LastError:     pgText(d.LastError),
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		CompletedAt:   pgTimestamptz(d.CompletedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ProvisioningDeliveryRepository) Find(ctx context.Context, tenantID, deliveryID string) (*domain.ProvisioningDelivery, error) {
	row, err := New(r.Pool).FindProvisioningDelivery(ctx, FindProvisioningDeliveryParams{
		TenantID: tenantID,
		ID:       deliveryID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapDelivery(row)
}

func (r *ProvisioningDeliveryRepository) ListByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, limit int) ([]*domain.ProvisioningDelivery, error) {
	var rows []*ProvisioningDelivery
	var err error
	if status != nil {
		rows, err = New(r.Pool).ListProvisioningDeliveriesByConnectionAndStatus(ctx, ListProvisioningDeliveriesByConnectionAndStatusParams{
			TenantID:     tenantID,
			ConnectionID: connectionID,
			Status:       string(*status),
			Limit:        int32(limit), //nolint:gosec // safe downcast
		})
	} else {
		rows, err = New(r.Pool).ListProvisioningDeliveriesByConnection(ctx, ListProvisioningDeliveriesByConnectionParams{
			TenantID:     tenantID,
			ConnectionID: connectionID,
			Limit:        int32(limit), //nolint:gosec // safe downcast
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ProvisioningDelivery, 0, len(rows))
	for _, row := range rows {
		d, err := mapDelivery(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// ListPageByConnection implements
// ports.ProvisioningDeliveryRepository.ListPageByConnection (wi-159)
// : keyset pagination ordered by (created_at, id) descending —
// matching ListByConnection's pre-existing "most recent first" order.
func (r *ProvisioningDeliveryRepository) ListPageByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, sourceType *domain.ProvisioningSourceType, afterCreatedAt time.Time, afterID string, limit int) ([]*domain.ProvisioningDelivery, error) {
	q := New(r.Pool)
	filterStatus, filterSourceType := "", ""
	if status != nil {
		filterStatus = string(*status)
	}
	if sourceType != nil {
		filterSourceType = string(*sourceType)
	}
	var rows []*ProvisioningDelivery
	var err error
	first := afterCreatedAt.IsZero() && afterID == ""
	if first {
		rows, err = q.ListProvisioningDeliveriesByConnectionPage(ctx, ListProvisioningDeliveriesByConnectionPageParams{
			TenantID: tenantID, ConnectionID: connectionID, FilterStatus: filterStatus, FilterSourceType: filterSourceType,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListProvisioningDeliveriesByConnectionPageAfter(ctx, ListProvisioningDeliveriesByConnectionPageAfterParams{
			TenantID: tenantID, ConnectionID: connectionID, FilterStatus: filterStatus, FilterSourceType: filterSourceType,
			AfterCreatedAt: afterCreatedAt, AfterID: afterID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ProvisioningDelivery, 0, len(rows))
	for _, row := range rows {
		d, err := mapDelivery(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *ProvisioningDeliveryRepository) ListPageBeforeByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, sourceType *domain.ProvisioningSourceType, beforeCreatedAt time.Time, beforeID string, limit int) ([]*domain.ProvisioningDelivery, error) {
	filterStatus, filterSourceType := "", ""
	if status != nil {
		filterStatus = string(*status)
	}
	if sourceType != nil {
		filterSourceType = string(*sourceType)
	}
	rows, err := New(r.Pool).ListProvisioningDeliveriesByConnectionPageBefore(ctx, ListProvisioningDeliveriesByConnectionPageBeforeParams{
		TenantID: tenantID, ConnectionID: connectionID, FilterStatus: filterStatus, FilterSourceType: filterSourceType,
		BeforeCreatedAt: beforeCreatedAt, BeforeID: beforeID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*domain.ProvisioningDelivery, 0, len(rows))
	for _, row := range rows {
		delivery, err := mapDelivery(row)
		if err != nil {
			return nil, err
		}
		out = append(out, delivery)
	}
	return out, nil
}

func (r *ProvisioningDeliveryRepository) ListUnenqueued(ctx context.Context, limit int) ([]*domain.ProvisioningDelivery, error) {
	rows, err := New(r.Pool).ListUnenqueuedProvisioningDeliveries(ctx, int32(limit)) //nolint:gosec // safe downcast
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ProvisioningDelivery, 0, len(rows))
	for _, row := range rows {
		d, err := mapDelivery(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *ProvisioningDeliveryRepository) AttachJob(ctx context.Context, tenantID, deliveryID, jobID string) (bool, error) {
	affected, err := New(r.Pool).AttachProvisioningDeliveryJob(ctx, AttachProvisioningDeliveryJobParams{
		TenantID: tenantID,
		ID:       deliveryID,
		JobID:    pgUUIDVal(jobID),
	})
	return affected == 1, err
}

func (r *ProvisioningDeliveryRepository) UpdateStatus(ctx context.Context, tenantID, deliveryID string, status domain.ProvisioningDeliveryStatus, lastError *string) error {
	return New(r.Pool).UpdateProvisioningDeliveryStatus(ctx, UpdateProvisioningDeliveryStatusParams{
		TenantID:  tenantID,
		ID:        deliveryID,
		Status:    string(status),
		LastError: pgText(lastError),
	})
}

func (r *ProvisioningDeliveryRepository) RetryDeadLetter(ctx context.Context, tenantID, deliveryID string) (bool, error) {
	affected, err := New(r.Pool).RetryDeadLetterProvisioningDelivery(ctx, RetryDeadLetterProvisioningDeliveryParams{
		TenantID: tenantID,
		ID:       deliveryID,
	})
	return affected == 1, err
}
