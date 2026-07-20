// Package postgres implements the Provisioning bounded context's repositories
// on PostgreSQL using hand-written SQL via pgx (LifecycleWorkflowRunRepository
// precedent; sqlc is not required for every context, ADR-090). credential_secret
// is stored as plaintext for now (dev/test grade, see infra/schema/postgres.sql
// comment and wi-97 envelope-encryption-at-rest).
package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

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
	row := r.Pool.QueryRow(ctx, `
INSERT INTO provisioning_connections (
  application_id, tenant_id, status, base_url, credential_id, auth_method, credential_secret,
  credential_created_at, credential_rotated_at, capabilities, feature_flags, scope, group_push,
  attribute_mappings, matching, deprovision_policy, rate_limit_per_minute, max_attempts,
  notification_email, quarantine_after_consecutive_failures, health, consecutive_failure_count,
  last_full_sync_at, quarantined_at, quarantine_reason, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
ON CONFLICT (application_id) DO NOTHING
RETURNING application_id`,
		conn.ApplicationID, conn.TenantID, conn.Status, conn.BaseURL, conn.Credential.CredentialID, conn.Credential.AuthMethod, secret,
		conn.Credential.CreatedAt, conn.Credential.RotatedAt, j.capabilities, j.featureFlags, conn.Scope, j.groupPush,
		j.mappings, j.matching, j.deprovision, conn.RateLimitPerMinute, conn.MaxAttempts,
		conn.NotificationEmail, conn.QuarantineAfterConsecutiveFailure, conn.Health, conn.ConsecutiveFailureCount,
		conn.LastFullSyncAt, conn.QuarantinedAt, conn.QuarantineReason, conn.CreatedAt, conn.UpdatedAt)
	var id string
	if err := row.Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrConnectionAlreadyExists
	} else if err != nil {
		return err
	}
	return nil
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
		_, err = r.Pool.Exec(ctx, `
UPDATE provisioning_connections SET
  status=$3, base_url=$4, credential_id=$5, auth_method=$6, credential_secret=$7,
  credential_rotated_at=$8, capabilities=$9, feature_flags=$10, scope=$11, group_push=$12,
  attribute_mappings=$13, matching=$14, deprovision_policy=$15, rate_limit_per_minute=$16,
  max_attempts=$17, notification_email=$18, quarantine_after_consecutive_failures=$19, health=$20,
  consecutive_failure_count=$21, last_full_sync_at=$22, quarantined_at=$23, quarantine_reason=$24,
  updated_at=$25
WHERE tenant_id=$1 AND application_id=$2`,
			conn.TenantID, conn.ApplicationID, conn.Status, conn.BaseURL, conn.Credential.CredentialID, conn.Credential.AuthMethod, *secret,
			conn.Credential.RotatedAt, j.capabilities, j.featureFlags, conn.Scope, j.groupPush,
			j.mappings, j.matching, j.deprovision, conn.RateLimitPerMinute,
			conn.MaxAttempts, conn.NotificationEmail, conn.QuarantineAfterConsecutiveFailure, conn.Health,
			conn.ConsecutiveFailureCount, conn.LastFullSyncAt, conn.QuarantinedAt, conn.QuarantineReason,
			conn.UpdatedAt)
		return err
	}
	_, err = r.Pool.Exec(ctx, `
UPDATE provisioning_connections SET
  status=$3, base_url=$4, capabilities=$5, feature_flags=$6, scope=$7, group_push=$8,
  attribute_mappings=$9, matching=$10, deprovision_policy=$11, rate_limit_per_minute=$12,
  max_attempts=$13, notification_email=$14, quarantine_after_consecutive_failures=$15, health=$16,
  consecutive_failure_count=$17, last_full_sync_at=$18, quarantined_at=$19, quarantine_reason=$20,
  updated_at=$21
WHERE tenant_id=$1 AND application_id=$2`,
		conn.TenantID, conn.ApplicationID, conn.Status, conn.BaseURL, j.capabilities, j.featureFlags, conn.Scope, j.groupPush,
		j.mappings, j.matching, j.deprovision, conn.RateLimitPerMinute,
		conn.MaxAttempts, conn.NotificationEmail, conn.QuarantineAfterConsecutiveFailure, conn.Health,
		conn.ConsecutiveFailureCount, conn.LastFullSyncAt, conn.QuarantinedAt, conn.QuarantineReason,
		conn.UpdatedAt)
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

const connectionColumns = `application_id, tenant_id, status, base_url, credential_id, auth_method,
credential_created_at, credential_rotated_at, capabilities, feature_flags, scope, group_push,
attribute_mappings, matching, deprovision_policy, rate_limit_per_minute, max_attempts,
notification_email, quarantine_after_consecutive_failures, health, consecutive_failure_count,
last_full_sync_at, quarantined_at, quarantine_reason, created_at, updated_at`

func scanConnection(row sharedpg.RowScanner) (*domain.ProvisioningConnection, error) {
	c := &domain.ProvisioningConnection{Credential: domain.ProvisioningConnectionCredentialMetadata{}}
	var capabilitiesRaw, featureFlagsRaw, groupPushRaw, mappingsRaw, matchingRaw, deprovisionRaw []byte
	err := row.Scan(
		&c.ApplicationID, &c.TenantID, &c.Status, &c.BaseURL, &c.Credential.CredentialID, &c.Credential.AuthMethod,
		&c.Credential.CreatedAt, &c.Credential.RotatedAt, &capabilitiesRaw, &featureFlagsRaw, &c.Scope, &groupPushRaw,
		&mappingsRaw, &matchingRaw, &deprovisionRaw, &c.RateLimitPerMinute, &c.MaxAttempts,
		&c.NotificationEmail, &c.QuarantineAfterConsecutiveFailure, &c.Health, &c.ConsecutiveFailureCount,
		&c.LastFullSyncAt, &c.QuarantinedAt, &c.QuarantineReason, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(featureFlagsRaw, &c.FeatureFlags); err != nil {
		return nil, err
	}
	if len(capabilitiesRaw) > 0 {
		var caps domain.ProvisioningCapabilities
		if err := json.Unmarshal(capabilitiesRaw, &caps); err != nil {
			return nil, err
		}
		c.Capabilities = &caps
	}
	if len(groupPushRaw) > 0 {
		var gp domain.GroupPushConfig
		if err := json.Unmarshal(groupPushRaw, &gp); err != nil {
			return nil, err
		}
		c.GroupPush = &gp
	}
	if err := json.Unmarshal(mappingsRaw, &c.AttributeMappings); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(matchingRaw, &c.Matching); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(deprovisionRaw, &c.DeprovisionPolicy); err != nil {
		return nil, err
	}
	return c, c.Validate()
}

func (r *ProvisioningConnectionRepository) Find(ctx context.Context, tenantID, applicationID string) (*domain.ProvisioningConnection, error) {
	conn, err := scanConnection(r.Pool.QueryRow(ctx, `SELECT `+connectionColumns+` FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2`, tenantID, applicationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return conn, err
}

func (r *ProvisioningConnectionRepository) CredentialSecret(ctx context.Context, tenantID, applicationID string) (string, error) {
	var secret string
	err := r.Pool.QueryRow(ctx, `SELECT credential_secret FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2`, tenantID, applicationID).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return secret, err
}

func (r *ProvisioningConnectionRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2`, tenantID, applicationID)
	return err
}

func (r *ProvisioningConnectionRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ProvisioningConnection, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+connectionColumns+` FROM provisioning_connections WHERE tenant_id=$1 ORDER BY application_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ProvisioningConnection{}
	for rows.Next() {
		conn, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, conn)
	}
	return out, rows.Err()
}

// RemoteResourceLinkRepository is the PostgreSQL ports.RemoteResourceLinkRepository.
type RemoteResourceLinkRepository struct{ Pool sharedpg.DB }

var _ ports.RemoteResourceLinkRepository = (*RemoteResourceLinkRepository)(nil)

func (r *RemoteResourceLinkRepository) Find(ctx context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) (*domain.RemoteResourceLink, error) {
	link := &domain.RemoteResourceLink{}
	var etag *string
	err := r.Pool.QueryRow(ctx, `SELECT connection_id,tenant_id,source_type,source_id,remote_id,external_id,etag,last_synced_version,updated_at
FROM provisioning_remote_links WHERE connection_id=$1 AND source_type=$2 AND source_id=$3`, connectionID, sourceType, sourceID).
		Scan(&link.ConnectionID, &link.TenantID, &link.SourceType, &link.SourceID, &link.RemoteID, &link.ExternalID, &etag, &link.LastSyncedVersion, &link.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	link.ETag = etag
	return link, nil
}

func (r *RemoteResourceLinkRepository) Upsert(ctx context.Context, link *domain.RemoteResourceLink) error {
	_, err := r.Pool.Exec(ctx, `
INSERT INTO provisioning_remote_links (connection_id, tenant_id, source_type, source_id, remote_id, external_id, etag, last_synced_version, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (connection_id, source_type, source_id) DO UPDATE SET
  remote_id=EXCLUDED.remote_id, external_id=EXCLUDED.external_id, etag=EXCLUDED.etag,
  last_synced_version=EXCLUDED.last_synced_version, updated_at=EXCLUDED.updated_at`,
		link.ConnectionID, link.TenantID, link.SourceType, link.SourceID, link.RemoteID, link.ExternalID, link.ETag, link.LastSyncedVersion, link.UpdatedAt)
	return err
}

// ProvisioningDeliveryRepository is the PostgreSQL ports.ProvisioningDeliveryRepository.
type ProvisioningDeliveryRepository struct{ Pool sharedpg.DB }

var _ ports.ProvisioningDeliveryRepository = (*ProvisioningDeliveryRepository)(nil)

func (r *ProvisioningDeliveryRepository) Save(ctx context.Context, d *domain.ProvisioningDelivery) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, err
	}
	row := r.Pool.QueryRow(ctx, `
INSERT INTO provisioning_deliveries (id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (tenant_id, connection_id, source_type, source_id, source_version) DO NOTHING
RETURNING id`,
		d.ID, d.TenantID, d.ConnectionID, d.SourceType, d.SourceID, d.SourceVersion, d.Operation, d.Status, d.JobID, d.LastError, d.CreatedAt, d.UpdatedAt, d.CompletedAt)
	var id string
	if err := row.Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

const deliveryColumns = `id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at`

func scanDelivery(row sharedpg.RowScanner) (*domain.ProvisioningDelivery, error) {
	d := &domain.ProvisioningDelivery{}
	err := row.Scan(&d.ID, &d.TenantID, &d.ConnectionID, &d.SourceType, &d.SourceID, &d.SourceVersion, &d.Operation, &d.Status, &d.JobID, &d.LastError, &d.CreatedAt, &d.UpdatedAt, &d.CompletedAt)
	if err != nil {
		return nil, err
	}
	return d, d.Validate()
}

func (r *ProvisioningDeliveryRepository) Find(ctx context.Context, tenantID, deliveryID string) (*domain.ProvisioningDelivery, error) {
	d, err := scanDelivery(r.Pool.QueryRow(ctx, `SELECT `+deliveryColumns+` FROM provisioning_deliveries WHERE tenant_id=$1 AND id=$2`, tenantID, deliveryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

func (r *ProvisioningDeliveryRepository) ListByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, limit int) ([]*domain.ProvisioningDelivery, error) {
	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = r.Pool.Query(ctx, `SELECT `+deliveryColumns+` FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2 AND status=$3 ORDER BY created_at DESC LIMIT $4`, tenantID, connectionID, *status, limit)
	} else {
		rows, err = r.Pool.Query(ctx, `SELECT `+deliveryColumns+` FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2 ORDER BY created_at DESC LIMIT $3`, tenantID, connectionID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ProvisioningDelivery{}
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *ProvisioningDeliveryRepository) ListUnenqueued(ctx context.Context, limit int) ([]*domain.ProvisioningDelivery, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+deliveryColumns+` FROM provisioning_deliveries WHERE status='pending' AND job_id IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ProvisioningDelivery{}
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *ProvisioningDeliveryRepository) AttachJob(ctx context.Context, tenantID, deliveryID, jobID string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `UPDATE provisioning_deliveries SET job_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND job_id IS NULL`, tenantID, deliveryID, jobID)
	return tag.RowsAffected() == 1, err
}

func (r *ProvisioningDeliveryRepository) UpdateStatus(ctx context.Context, tenantID, deliveryID string, status domain.ProvisioningDeliveryStatus, lastError *string) error {
	_, err := r.Pool.Exec(ctx, `
UPDATE provisioning_deliveries SET status=$3, last_error=$4, updated_at=now(),
  completed_at = CASE WHEN $3 IN ('succeeded','dead_letter') THEN now() ELSE completed_at END
WHERE tenant_id=$1 AND id=$2`, tenantID, deliveryID, status, lastError)
	return err
}

func (r *ProvisioningDeliveryRepository) RetryDeadLetter(ctx context.Context, tenantID, deliveryID string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `UPDATE provisioning_deliveries SET status='pending', job_id=NULL, last_error=NULL, updated_at=now()
WHERE tenant_id=$1 AND id=$2 AND status='dead_letter'`, tenantID, deliveryID)
	return tag.RowsAffected() == 1, err
}
