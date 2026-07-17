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

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

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
		"UserAuthenticated",
		"AuthenticationStepCompleted",
		"MfaChallengeIssued",
		"MfaChallengeSucceeded",
		"MfaEnrollmentRequired",
		"MfaEnrollmentCompleted",
		"MfaEnrollmentBypassConsumed",
		"BackupCodeConsumed",
		"SessionStarted",
		"SessionRefreshed",
		"SessionEnded",
		"FederatedAuthenticated",
		"FederationLinked",
		"FederationUnlinked",
		"SessionImpersonationStarted",
		"SessionImpersonationEnded",
	},
	"fail": {
		"AuthenticationFailed",
		"AuthenticationStepFailed",
		"MfaChallengeFailed",
	},
	"aggregated": {
		"AuthenticationEventAggregated",
		"LoginThrottled",
	},
	"user": {
		"UserCreated",
		"UserUpdated",
		"UserDisabled",
		"UserEnabled",
		"UserSoftDeleted",
		"UserRestored",
		"UserDeleted",
		"UserRequiredActionSet",
		"UserRequiredActionCleared",
		"PasswordChanged",
		"PasswordResetRequested",
		"EmailChangeRequested",
		"EmailChanged",
		"MfaFactorEnrolled",
		"MfaFactorRemoved",
		"MfaEnrollmentBypassIssued",
		"MfaEnrollmentBypassRevoked",
		"MfaEnrollmentBypassExpired",
	},
	"group": {
		"GroupCreated",
		"GroupUpdated",
		"GroupDeleted",
		"GroupMemberAdded",
		"GroupMemberRemoved",
	},
	"client": {
		"ClientRegistered",
		"AdminOAuth2ClientCreated",
		"AdminOAuth2ClientUpdated",
		"AdminOAuth2ClientDeleted",
	},
	"consent": {
		"ConsentGranted",
		"ConsentRevoked",
	},
	"token": {
		"AuthorizationCodeIssued",
		"AuthorizationCodeRedeemed",
		"AccessTokenIssued",
		"RefreshTokenIssued",
		"RefreshTokenRotated",
		"TokenRevoked",
		"TokenIntrospected",
		"RefreshTokenReuseDetected",
		"PARStored",
		"DeviceAuthorizationRequested",
		"DeviceAuthorizationApproved",
		"DeviceAuthorizationDenied",
	},
	"tenant": {
		"TenantCreated",
		"TenantUpdated",
		"TenantDisabled",
		"TenantEnabled",
		"TenantUserAttributeSchemaUpdated",
	},
	"key": {
		"SigningKeyRotated",
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
	query, noMatch, err := d.parseAuditEventQuery(c, actor)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if noMatch {
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": []AdminAuditEventResponse{}})
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
	query, noMatch, err := d.parseAuditEventQuery(c, actor)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	query.Limit = adminAuditEventExportMaxLimit
	var records []*auditports.AuditEventRecord
	if d.AuditEventRepo != nil && !noMatch {
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

// AuditEventSearchOptionsResponse は SCL AuditEventSearchOptionsResponse の双子。
type AuditEventSearchOptionsResponse struct {
	EventTypes []string `json:"event_types"`
	Outcomes   []string `json:"outcomes"`
}

// auditEventOutcomeChoices は outcome フィルタの選択肢 (eventOutcome() の分類先と一致させる)。
var auditEventOutcomeChoices = []string{"success", "failure"}

// handleAdminAuditEventSearchOptions は event.type / outcome を選択式にするための選択肢一覧を
// 返す (wi-147)。event_types は auditEventCategoryTypes (category 絞り込みと同じ単一の正) の
// 和集合から重複除去・ソートして導出し、UI 側の手書きリストとの drift を防ぐ。
func (d Deps) handleAdminAuditEventSearchOptions(c *echo.Context) error {
	if _, err := d.RequireAuditReader(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	seen := map[string]bool{}
	eventTypes := make([]string, 0, len(auditEventCategoryTypes))
	for _, types := range auditEventCategoryTypes {
		for _, t := range types {
			if seen[t] {
				continue
			}
			seen[t] = true
			eventTypes = append(eventTypes, t)
		}
	}
	slices.Sort(eventTypes)
	return support.NoStoreJSON(c, http.StatusOK, AuditEventSearchOptionsResponse{
		EventTypes: eventTypes,
		Outcomes:   auditEventOutcomeChoices,
	})
}

// requireAuditReader は AdminAuditEventsRead パーミッションを満たすユーザーを返す。
// admin / system_admin のどちらでも通る。所属テナントの拘束は問わない (実際の
// テナント絞り込みは List のクエリ生成時に行う)。

// parseAuditEventQuery は query string を AuditEventQuery へ変換する。第 2 戻り値 noMatch が
// true の場合、username が実アカウントに解決できなかったことを示し、呼び出し側は
// AuditEventRepo.List を呼ばず空の結果を返す (フィルタ無視で全件返すという誤動作を避ける)。
func (d Deps) parseAuditEventQuery(c *echo.Context, actor *idmdomain.User) (auditports.AuditEventQuery, bool, error) {
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
			return auditports.AuditEventQuery{}, false, errors.New("category が不正です")
		}
		q.Types = types
	}
	if userID := c.QueryParam("user_id"); userID != "" {
		q.UserID = userID
	}
	// username (wi-147): 実アカウントが常に確定するイベントの検索用。payload に username/hash を
	// 持たせず、検索時に UserRepo で user_id へ解決してから既存の user_id フィルタで絞り込む。
	// 該当ユーザーが存在しない場合は noMatch=true (0 件を返す。全件返すフォールバックはしない)。
	if username := c.QueryParam("username"); username != "" {
		if d.UserRepo == nil {
			return auditports.AuditEventQuery{}, true, nil
		}
		resolveTenant := q.TenantID
		if q.AllTenants {
			resolveTenant = actor.TenantID
		}
		user, err := d.UserRepo.FindByUsername(c.Request().Context(), resolveTenant, username)
		if err != nil {
			return auditports.AuditEventQuery{}, false, err
		}
		if user == nil {
			return auditports.AuditEventQuery{}, true, nil
		}
		q.UserID = user.ID
	}
	if after := c.QueryParam("after"); after != "" {
		t, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return auditports.AuditEventQuery{}, false, errors.New("after は RFC3339 形式を指定してください")
		}
		q.After = t
	}
	if before := c.QueryParam("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return auditports.AuditEventQuery{}, false, errors.New("before は RFC3339 形式を指定してください")
		}
		q.Before = t
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil || limit < 0 {
			return auditports.AuditEventQuery{}, false, errors.New("limit は 0 以上の整数を指定してください")
		}
		q.Limit = limit
	}
	// wi-145: q フリーテキストと filter 式 (registry allowlist)。
	if freeText := c.QueryParam("q"); freeText != "" {
		q.Q = freeText
	}
	filters, err := d.parseAuditFilters(c)
	if err != nil {
		return auditports.AuditEventQuery{}, false, err
	}
	q.Filters = filters
	return q, false, nil
}

// parseAuditFilters は繰り返し filter=field:op:value[,value2] クエリを parse / validate する
// (wi-145)。field/operator は registry allowlist のみ許可。IPv6 の値に含まれる ":" を壊さない
// よう先頭 2 個の ":" で切る。ADR-104 により PII 属性の hash/truncate transform はしない (平文一致)。
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
	return auditusecases.ParseAuditFilter(parsed)
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
