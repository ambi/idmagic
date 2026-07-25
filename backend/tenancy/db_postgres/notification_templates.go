package db_postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// NotificationTemplateRepository は通知テンプレートのテナント上書きを PostgreSQL に
// 保存する (wi-288, ADR-142)。行の有無がそのまま「上書き済み / 組込み既定」を表し、
// 削除が「既定へのリセット」になる。クエリは sqlc 生成。
type NotificationTemplateRepository struct{ Pool sharedpg.DB }

func (r *NotificationTemplateRepository) FindByKey(
	ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string,
) (*notificationports.TemplateOverride, error) {
	row, err := New(r.Pool).FindNotificationTemplate(ctx, FindNotificationTemplateParams{
		TenantID: tenantID, TemplateKey: string(key), Locale: locale,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return notificationTemplateFromRow(row), nil
}

func (r *NotificationTemplateRepository) ListByTenant(
	ctx context.Context, tenantID string,
) ([]*notificationports.TemplateOverride, error) {
	rows, err := New(r.Pool).ListNotificationTemplatesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*notificationports.TemplateOverride, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationTemplateFromRow(row))
	}
	return out, nil
}

func (r *NotificationTemplateRepository) Save(ctx context.Context, override *notificationports.TemplateOverride) error {
	return New(r.Pool).SaveNotificationTemplate(ctx, SaveNotificationTemplateParams{
		TenantID:        override.TenantID,
		TemplateKey:     string(override.Key),
		Locale:          override.Locale,
		Subject:         override.Subject,
		BodyText:        override.BodyText,
		BodyHtml:        override.BodyHTML,
		FromDisplayName: textOrNil(override.FromDisplayName),
		CreatedAt:       override.CreatedAt,
		UpdatedAt:       override.UpdatedAt,
	})
}

func (r *NotificationTemplateRepository) Delete(
	ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string,
) (bool, error) {
	affected, err := New(r.Pool).DeleteNotificationTemplate(ctx, DeleteNotificationTemplateParams{
		TenantID: tenantID, TemplateKey: string(key), Locale: locale,
	})
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func notificationTemplateFromRow(row *NotificationTemplate) *notificationports.TemplateOverride {
	return &notificationports.TemplateOverride{
		TenantID:        row.TenantID,
		Key:             notificationports.TemplateKey(row.TemplateKey),
		Locale:          row.Locale,
		Subject:         row.Subject,
		BodyText:        row.BodyText,
		BodyHTML:        row.BodyHtml,
		FromDisplayName: row.FromDisplayName.String,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
