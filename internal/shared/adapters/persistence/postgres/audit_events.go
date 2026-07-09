package postgres

// AuditEventRepository は AuditEventRepository (SCL OAuth2 bounded context) を PostgreSQL に
// 永続化する読み出しモデル。in-memory 実装 (memory.AuditEventStore) と同じ port 契約を
// 満たし、admin の時系列調査 / 本人サインイン履歴 / wi-44 の認証イベント検索が共有する。
// 付加属性 (ip_truncated / ip_hash / session_id 等) は payload JSONB に載るため、本テーブルは
// 構造化カラムを増やさず type / user_id / occurred_at の絞り込みだけを担う (ADR-041)。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/internal/oauth2/ports"
)

const (
	auditDefaultListLimit = 100
	auditMaxListLimit     = 1000
)

type AuditEventRepository struct{ Pool DB }

const auditEventSelect = `SELECT id,tenant_id,type,occurred_at,payload FROM audit_events`

func scanAuditEvent(row rowScanner) (*ports.AuditEventRecord, error) {
	var rec ports.AuditEventRecord
	err := row.Scan(&rec.ID, &rec.TenantID, &rec.Type, &rec.OccurredAt, &rec.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rec.Payload == nil {
		rec.Payload = map[string]any{}
	}
	return &rec, nil
}

func (r *AuditEventRepository) Append(ctx context.Context, rec *ports.AuditEventRecord) error {
	if rec == nil || rec.ID == "" || rec.Type == "" {
		return nil
	}
	var userID *string
	if s, ok := rec.Payload["userId"].(string); ok && s != "" {
		userID = &s
	}
	payload := rec.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (id,tenant_id,type,user_id,occurred_at,payload)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (id) DO NOTHING`,
		rec.ID, rec.TenantID, rec.Type, userID, rec.OccurredAt, payload); err != nil {
		return err
	}
	// wi-145: sidecar 検索属性を書く。attr_name は AuditSearchRegistry の Field。冪等 (ON CONFLICT)。
	for name, value := range rec.SearchAttributes {
		if value == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO audit_event_search_attributes (event_id,tenant_id,attr_name,attr_value,occurred_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (event_id,attr_name) DO NOTHING`,
			rec.ID, rec.TenantID, name, value, rec.OccurredAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *AuditEventRepository) List(ctx context.Context, q ports.AuditEventQuery) ([]*ports.AuditEventRecord, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = auditDefaultListLimit
	}
	if limit > auditMaxListLimit {
		limit = auditMaxListLimit
	}
	var conds []string
	var args []any
	add := func(expr string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(expr, len(args)))
	}
	// addExists は sidecar への EXISTS 副問合せを 1 条件足す (attr_name + 値の 2 引数)。
	addExists := func(condFmt, field string, val any) {
		args = append(args, field)
		fieldIdx := len(args)
		args = append(args, val)
		valIdx := len(args)
		conds = append(conds, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM audit_event_search_attributes a WHERE a.event_id = audit_events.id AND "+condFmt+")",
			fieldIdx, valIdx))
	}
	if !q.AllTenants && q.TenantID != "" {
		add("tenant_id = $%d", q.TenantID)
	}
	if q.Type != "" {
		add("type = $%d", q.Type)
	}
	if len(q.Types) > 0 {
		add("type = ANY($%d)", q.Types)
	}
	if q.UserID != "" {
		add("user_id = $%d", q.UserID)
	}
	if !q.After.IsZero() {
		add("occurred_at >= $%d", q.After)
	}
	if !q.Before.IsZero() {
		add("occurred_at <= $%d", q.Before)
	}
	// wi-145: registry allowlist の filter 式 (連言)。各式は sidecar への EXISTS 照合。
	for _, expr := range q.Filters {
		switch expr.Operator {
		case ports.OpEq:
			if len(expr.Values) == 1 {
				addExists("attr_name = $%d AND attr_value = $%d", expr.Field, expr.Values[0])
			}
		case ports.OpIn:
			if len(expr.Values) > 0 {
				addExists("attr_name = $%d AND attr_value = ANY($%d)", expr.Field, expr.Values)
			}
		case ports.OpContains:
			if len(expr.Values) == 1 {
				addExists("attr_name = $%d AND attr_value ILIKE $%d", expr.Field, "%"+expr.Values[0]+"%")
			}
		}
	}
	// wi-145: q は raw 保存された属性値への部分一致 (PII 列は対象外)。
	if q.Q != "" {
		rawNames := rawStorableAttrNames()
		args = append(args, rawNames)
		nameIdx := len(args)
		args = append(args, "%"+q.Q+"%")
		valIdx := len(args)
		conds = append(conds, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM audit_event_search_attributes a WHERE a.event_id = audit_events.id AND a.attr_name = ANY($%d) AND a.attr_value ILIKE $%d)",
			nameIdx, valIdx))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit)
	query := auditEventSelect + where + fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d", len(args))
	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ports.AuditEventRecord{}
	for rows.Next() {
		rec, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *AuditEventRepository) FindByID(ctx context.Context, id string) (*ports.AuditEventRecord, error) {
	return scanAuditEvent(r.Pool.QueryRow(ctx, auditEventSelect+" WHERE id=$1", id))
}

// rawStorableAttrNames は q フリーテキストの照合対象となる raw 保存属性の attr_name 一覧。
// PII 属性 (hash / ip_truncate) は含めない。
func rawStorableAttrNames() []string {
	names := make([]string, 0, len(ports.AuditSearchRegistry))
	for _, attr := range ports.AuditSearchRegistry {
		if attr.RawStorable {
			names = append(names, attr.Field)
		}
	}
	return names
}

// DeleteOlderThan は ADR-045 の保持期間 sweep。type 別 cutoff を個別 DELETE で消し、
// それ以外は Default cutoff で消す。Keep / ByType に挙げた type は Default 削除から除外する。
// (tenant_id, occurred_at) index が当たる。idempotent。
func (r *AuditEventRepository) DeleteOlderThan(ctx context.Context, cutoff ports.RetentionCutoff) (int64, error) {
	var deleted int64
	excluded := make([]string, 0, len(cutoff.ByType)+len(cutoff.Keep))
	for t := range cutoff.ByType {
		excluded = append(excluded, t)
	}
	excluded = append(excluded, cutoff.Keep...)
	for t, before := range cutoff.ByType {
		if before.IsZero() {
			continue
		}
		tag, err := r.Pool.Exec(ctx, "DELETE FROM audit_events WHERE type=$1 AND occurred_at < $2", t, before)
		if err != nil {
			return deleted, err
		}
		deleted += tag.RowsAffected()
	}
	if !cutoff.Default.IsZero() {
		tag, err := r.Pool.Exec(ctx,
			"DELETE FROM audit_events WHERE occurred_at < $1 AND type <> ALL($2)", cutoff.Default, excluded)
		if err != nil {
			return deleted, err
		}
		deleted += tag.RowsAffected()
	}
	return deleted, nil
}
