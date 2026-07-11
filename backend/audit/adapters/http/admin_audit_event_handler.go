package http

// SCL interfaces: ListAdminAuditEvents / GetAdminAuditEvent (bounded_context: Audit)。
// SCL permission: AdminAuditEventsRead — admin は所属テナント内、system_admin は
// default tenant 経路から全テナント横断で参照できる。書き込み経路は定義しない。

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type AdminAuditEventResponse struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Type       string         `json:"type"`
	OccurredAt time.Time      `json:"occurred_at"`
	Payload    map[string]any `json:"payload"`
}

// 監査ログのイベントカテゴリ → 監査 type 群 (wi-44)。admin が分かりやすく絞り込めるよう、
// 認証系は成功 / 失敗 / 集約のサブ分類を持ち (authentication はその和集合)、管理操作系も
// 大分類でまとめる。type 完全一致 (query.type) は機械向けの低レベルフィルタとして別に残す。
// 各値は SCL events の EventType 文字列 (owns_events と一致)。
var auditEventCategoryTypes = map[string][]string{
	"success": {
		(&spec.UserAuthenticated{}).EventType(),
		(&spec.AuthenticationStepCompleted{}).EventType(),
		(&spec.MfaChallengeIssued{}).EventType(),
		(&spec.MfaChallengeSucceeded{}).EventType(),
		(&spec.BackupCodeConsumed{}).EventType(),
		(&spec.SessionStarted{}).EventType(),
		(&spec.SessionRefreshed{}).EventType(),
		(&spec.SessionEnded{}).EventType(),
		(&spec.FederatedAuthenticated{}).EventType(),
		(&spec.FederationLinked{}).EventType(),
		(&spec.FederationUnlinked{}).EventType(),
		(&spec.SessionImpersonationStarted{}).EventType(),
		(&spec.SessionImpersonationEnded{}).EventType(),
	},
	"fail": {
		(&spec.AuthenticationFailed{}).EventType(),
		(&spec.AuthenticationStepFailed{}).EventType(),
		(&spec.MfaChallengeFailed{}).EventType(),
	},
	"aggregated": {
		(&spec.AuthenticationEventAggregated{}).EventType(),
		(&spec.LoginThrottled{}).EventType(),
	},
	"user": {
		(&spec.UserCreated{}).EventType(),
		(&spec.UserUpdated{}).EventType(),
		(&spec.UserDisabled{}).EventType(),
		(&spec.UserEnabled{}).EventType(),
		(&spec.UserSoftDeleted{}).EventType(),
		(&spec.UserRestored{}).EventType(),
		(&spec.UserDeleted{}).EventType(),
		(&spec.UserRequiredActionSet{}).EventType(),
		(&spec.UserRequiredActionCleared{}).EventType(),
		(&spec.PasswordChanged{}).EventType(),
		(&spec.PasswordResetRequested{}).EventType(),
		(&spec.EmailChangeRequested{}).EventType(),
		(&spec.EmailChanged{}).EventType(),
		(&spec.MfaFactorEnrolled{}).EventType(),
		(&spec.MfaFactorRemoved{}).EventType(),
	},
	"group": {
		(&spec.GroupCreated{}).EventType(),
		(&spec.GroupUpdated{}).EventType(),
		(&spec.GroupDeleted{}).EventType(),
		(&spec.GroupMemberAdded{}).EventType(),
		(&spec.GroupMemberRemoved{}).EventType(),
	},
	"client": {
		(&oauthdomain.ClientRegistered{}).EventType(),
		(&oauthdomain.AdminOAuth2ClientCreated{}).EventType(),
		(&oauthdomain.AdminOAuth2ClientUpdated{}).EventType(),
		(&oauthdomain.AdminOAuth2ClientDeleted{}).EventType(),
	},
	"consent": {
		(&oauthdomain.ConsentGrantedEvent{}).EventType(),
		(&oauthdomain.ConsentRevokedEvent{}).EventType(),
	},
	"token": {
		(&spec.AuthorizationCodeIssued{}).EventType(),
		(&spec.AuthorizationCodeRedeemed{}).EventType(),
		(&spec.AccessTokenIssued{}).EventType(),
		(&spec.RefreshTokenIssued{}).EventType(),
		(&spec.RefreshTokenRotated{}).EventType(),
		(&spec.TokenRevoked{}).EventType(),
		(&spec.TokenIntrospected{}).EventType(),
		(&spec.RefreshTokenReuseDetected{}).EventType(),
		(&spec.PARStored{}).EventType(),
		(&spec.DeviceAuthorizationRequested{}).EventType(),
		(&spec.DeviceAuthorizationApproved{}).EventType(),
		(&spec.DeviceAuthorizationDenied{}).EventType(),
	},
	"tenant": {
		(&spec.TenantCreated{}).EventType(),
		(&spec.TenantUpdated{}).EventType(),
		(&spec.TenantDisabled{}).EventType(),
		(&spec.TenantEnabled{}).EventType(),
		(&spec.TenantUserAttributeSchemaUpdated{}).EventType(),
	},
	"key": {
		(&spec.SigningKeyRotated{}).EventType(),
	},
}

func init() {
	authn := []string{}
	for _, k := range []string{"success", "fail", "aggregated"} {
		authn = append(authn, auditEventCategoryTypes[k]...)
	}
	auditEventCategoryTypes["authentication"] = authn
}

const adminAuditEventExportMaxLimit = 10000

func (d Deps) handleListAdminAuditEvents(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.AuditEventRepo == nil {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": []AdminAuditEventResponse{}})
	}
	query, err := d.parseAuditEventQuery(c, actor)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	records, err := d.AuditEventRepo.List(c.Request().Context(), query)
	if err != nil {
		return err
	}
	response := make([]AdminAuditEventResponse, len(records))
	for i, rec := range records {
		response[i] = toAdminAuditEventResponse(rec)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": response})
}

func (d Deps) handleGetAdminAuditEvent(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.AuditEventRepo == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "event_not_found", "監査イベントが存在しません")
	}
	rec, err := d.AuditEventRepo.FindByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	if rec == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "event_not_found", "監査イベントが存在しません")
	}
	if !auditEventVisibleTo(rec, actor) {
		// 別テナントのイベントは存在を隠す。
		return support.WriteBrowserError(c, http.StatusNotFound, "event_not_found", "監査イベントが存在しません")
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminAuditEventResponse(rec))
}

func (d Deps) handleExportAdminAuditEvents(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	query, err := d.parseAuditEventQuery(c, actor)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	query.Limit = adminAuditEventExportMaxLimit
	var records []*auditports.AuditEventRecord
	if d.AuditEventRepo != nil {
		records, err = d.AuditEventRepo.List(c.Request().Context(), query)
		if err != nil {
			return err
		}
	}
	response := make([]AdminAuditEventResponse, len(records))
	for i, rec := range records {
		response[i] = toAdminAuditEventResponse(rec)
	}
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"audit_events.json\"")
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": response})
}

// requireAuditReader は AdminAuditEventsRead パーミッションを満たすユーザーを返す。
// admin / system_admin のどちらでも通る。所属テナントの拘束は問わない (実際の
// テナント絞り込みは List のクエリ生成時に行う)。

func (d Deps) parseAuditEventQuery(c *echo.Context, actor *idmdomain.User) (auditports.AuditEventQuery, error) {
	q := auditports.AuditEventQuery{
		TenantID:   actor.TenantID,
		AllTenants: false,
	}
	// system_admin が default tenant 経路で全テナント横断する場合のみ all_tenants を許可する。
	if slices.Contains(actor.Roles, "system_admin") &&
		actor.TenantID == tenancydomain.DefaultTenantID &&
		c.QueryParam("all_tenants") == "true" {
		q.AllTenants = true
		q.TenantID = ""
	}
	if t := c.QueryParam("type"); t != "" {
		q.Type = t
	}
	// category はイベントカテゴリ絞り込み (wi-44 統合: 認証サブ分類 + 管理操作カテゴリ)。
	if category := c.QueryParam("category"); category != "" {
		types, ok := auditEventCategoryTypes[category]
		if !ok {
			return auditports.AuditEventQuery{}, errors.New("category が不正です")
		}
		q.Types = types
	}
	if userID := c.QueryParam("user_id"); userID != "" {
		q.UserID = userID
	}
	if after := c.QueryParam("after"); after != "" {
		t, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return auditports.AuditEventQuery{}, errors.New("after は RFC3339 形式を指定してください")
		}
		q.After = t
	}
	if before := c.QueryParam("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return auditports.AuditEventQuery{}, errors.New("before は RFC3339 形式を指定してください")
		}
		q.Before = t
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil || limit < 0 {
			return auditports.AuditEventQuery{}, errors.New("limit は 0 以上の整数を指定してください")
		}
		q.Limit = limit
	}
	// wi-145: q フリーテキストと filter 式 (registry allowlist)。
	if freeText := c.QueryParam("q"); freeText != "" {
		q.Q = freeText
	}
	filters, err := d.parseAuditFilters(c)
	if err != nil {
		return auditports.AuditEventQuery{}, err
	}
	q.Filters = filters
	return q, nil
}

// parseAuditFilters は繰り返し filter=field:op:value[,value2] クエリを parse / validate し、
// PII 属性の平文値を tenant salt でサーバ側 transform する (wi-145)。field/operator は
// registry allowlist のみ許可。IPv6 の値に含まれる ":" を壊さないよう先頭 2 個の ":" で切る。
func (d Deps) parseAuditFilters(c *echo.Context) ([]auditports.AuditFilterExpression, error) {
	raw := c.Request().URL.Query()["filter"]
	if len(raw) == 0 {
		return nil, nil
	}
	parsed := make([]auditusecases.RawFilter, 0, len(raw))
	for _, token := range raw {
		parts := strings.SplitN(token, ":", 3)
		if len(parts) != 3 {
			return nil, errors.New("filter は field:operator:value 形式で指定してください")
		}
		parsed = append(parsed, auditusecases.RawFilter{
			Field:    parts[0],
			Operator: parts[1],
			Values:   strings.Split(parts[2], ","),
		})
	}
	exprs, err := auditusecases.ParseAuditFilter(parsed)
	if err != nil {
		return nil, err
	}
	var salt []byte
	if d.TenantSaltStore != nil {
		if s, serr := d.TenantSaltStore.GetSalt(c.Request().Context()); serr == nil {
			salt = s
		}
	}
	return auditusecases.TransformFilterValues(exprs, salt)
}

// auditEventVisibleTo は GetAdminAuditEvent で別テナントイベントを隠すための判定。
// system_admin で default テナント在籍なら全件 OK、それ以外は所属テナントのみ。
func auditEventVisibleTo(rec *auditports.AuditEventRecord, actor *idmdomain.User) bool {
	if slices.Contains(actor.Roles, "system_admin") && actor.TenantID == tenancydomain.DefaultTenantID {
		return true
	}
	return rec.TenantID == actor.TenantID
}

func toAdminAuditEventResponse(rec *auditports.AuditEventRecord) AdminAuditEventResponse {
	return AdminAuditEventResponse{
		ID:         rec.ID,
		TenantID:   rec.TenantID,
		Type:       rec.Type,
		OccurredAt: rec.OccurredAt,
		Payload:    rec.Payload,
	}
}
