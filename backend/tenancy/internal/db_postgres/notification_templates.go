package db_postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// NotificationTemplateRepository は通知テンプレートのテナント上書きを PostgreSQL に
// 保存する (wi-288)。行の有無がそのまま「上書き済み / 組込み既定」を表し、
// 削除が「既定へのリセット」になる。クエリは sqlc 生成。
type NotificationTemplateRepository struct{ Pool sharedpg.DB }

func (r *NotificationTemplateRepository) Save(ctx context.Context, override *notificationports.TemplateOverride) error {
	_, err := r.Update(ctx, override.TenantID, override.Key, override.Locale, func(_ *notificationports.TemplateOverride) (*notificationports.TemplateOverride, error) {
		return override, nil
	})
	return err
}

func (r *NotificationTemplateRepository) Update(ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string, change func(*notificationports.TemplateOverride) (*notificationports.TemplateOverride, error)) (*notificationports.TemplateOverride, error) {
	return sharedpg.InTransaction(ctx, r.Pool, func(tx pgx.Tx) (*notificationports.TemplateOverride, error) {
		if _, err := New(tx).LockTenantForUpdate(ctx, tenantID); err != nil {
			return nil, err
		}
		bound := &NotificationTemplateRepository{Pool: sharedpg.BindTransaction(tx)}
		current, err := bound.FindByKey(ctx, tenantID, key, locale)
		if err != nil {
			return nil, err
		}
		updated, err := change(current)
		if err != nil {
			return nil, err
		}
		if updated == nil {
			_, err := New(tx).DeleteNotificationTemplate(ctx, DeleteNotificationTemplateParams{TenantID: tenantID, TemplateKey: string(key), Locale: locale})
			return nil, err
		}
		return updated, bound.save(ctx, updated)
	})
}

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

func (r *NotificationTemplateRepository) ListAll(
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

func (r *NotificationTemplateRepository) save(ctx context.Context, override *notificationports.TemplateOverride) error {
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
	removed := false
	_, err := r.Update(ctx, tenantID, key, locale, func(current *notificationports.TemplateOverride) (*notificationports.TemplateOverride, error) {
		removed = current != nil
		return nil, nil
	})
	return removed, err
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
