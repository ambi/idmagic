-- name: FindNotificationTemplate :one
SELECT tenant_id, template_key, locale, subject, body_text, body_html, from_display_name,
       created_at, updated_at
FROM notification_templates
WHERE tenant_id = $1 AND template_key = $2 AND locale = $3;

-- name: ListNotificationTemplatesByTenant :many
SELECT tenant_id, template_key, locale, subject, body_text, body_html, from_display_name,
       created_at, updated_at
FROM notification_templates
WHERE tenant_id = $1
ORDER BY template_key, locale;

-- name: SaveNotificationTemplate :exec
INSERT INTO notification_templates (
    tenant_id, template_key, locale, subject, body_text, body_html, from_display_name,
    created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (tenant_id, template_key, locale) DO UPDATE SET
    subject = EXCLUDED.subject,
    body_text = EXCLUDED.body_text,
    body_html = EXCLUDED.body_html,
    from_display_name = EXCLUDED.from_display_name,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteNotificationTemplate :execrows
DELETE FROM notification_templates
WHERE tenant_id = $1 AND template_key = $2 AND locale = $3;
