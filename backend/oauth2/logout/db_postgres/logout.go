package db_postgres

import (
	"context"
	"errors"
	"time"

	oauth2pg "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/logout/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ClientSessionStore struct{ Pool sharedpg.DB }

func (s *ClientSessionStore) Upsert(ctx context.Context, session *domain.ClientSession) error {
	return oauth2pg.New(s.Pool).UpsertClientSession(ctx, oauth2pg.UpsertClientSessionParams{
		TenantID: session.TenantID, Sid: session.Sid, ClientID: session.ClientID,
		FirstIssuedAt: session.FirstIssuedAt, LastIssuedAt: session.LastIssuedAt,
	})
}

func (s *ClientSessionStore) ListBySid(ctx context.Context, tenantID, sid string) ([]*domain.ClientSession, error) {
	rows, err := oauth2pg.New(s.Pool).ListClientSessionsBySid(ctx, oauth2pg.ListClientSessionsBySidParams{TenantID: tenantID, Sid: sid})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ClientSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domain.ClientSession{TenantID: row.TenantID, Sid: row.Sid, ClientID: row.ClientID, FirstIssuedAt: row.FirstIssuedAt, LastIssuedAt: row.LastIssuedAt})
	}
	return result, nil
}

type NotificationStore struct{ Pool sharedpg.DB }

func (s *NotificationStore) Save(ctx context.Context, notification *domain.LogoutNotification) error {
	jobID, err := nullableUUID(notification.JobID)
	if err != nil {
		return err
	}
	return oauth2pg.New(s.Pool).SaveLogoutNotification(ctx, oauth2pg.SaveLogoutNotificationParams{
		ID: notification.ID, TenantID: notification.TenantID, Sid: notification.Sid,
		ClientID: notification.ClientID, LogoutTokenJti: notification.LogoutTokenJTI,
		TargetUri: notification.TargetURI, State: string(notification.State), Attempts: notification.Attempts,
		LastError: nullableText(notification.LastError), JobID: jobID, CreatedAt: notification.CreatedAt,
		DeliveredAt: nullableTime(notification.DeliveredAt),
	})
}

func (s *NotificationStore) FindByID(ctx context.Context, tenantID, id string) (*domain.LogoutNotification, error) {
	row, err := oauth2pg.New(s.Pool).FindLogoutNotificationByID(ctx, oauth2pg.FindLogoutNotificationByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.LogoutNotification{
		ID: row.ID, TenantID: row.TenantID, Sid: row.Sid, ClientID: row.ClientID,
		LogoutTokenJTI: row.LogoutTokenJti, TargetURI: row.TargetUri,
		State: domain.LogoutNotificationState(row.State), Attempts: row.Attempts,
		LastError: textPointer(row.LastError), JobID: uuidPointer(row.JobID),
		CreatedAt: row.CreatedAt, DeliveredAt: timePointer(row.DeliveredAt),
	}, nil
}

func nullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func nullableUUID(value *string) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}
	var result pgtype.UUID
	if err := result.Scan(*value); err != nil {
		return pgtype.UUID{}, err
	}
	return result, nil
}

func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func uuidPointer(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	converted := sharedpg.UUIDString(value)
	return &converted
}

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
