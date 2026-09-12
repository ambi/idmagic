package handlers_http

// SCL interfaces: ListAdminAuditEvents / GetAdminAuditEvent / ExportAdminAuditEvents と、
// その制御面の双子 ListSystemAuditEvents / GetSystemAuditEvent / ExportSystemAuditEvents
// (bounded_context: Audit)。
// SCL permission: AdminAuditEventsRead — テナント管理経路は要求元の所属テナントへ閉じ、
// 全テナント横断はシステム経路の制御面主体だけが到達できる (REQ-AUDIT-001 / REQ-AUDIT-007)。
// 書き込み経路は定義しない。

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2domain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

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
		"AuthenticatorResetRequested",
		"AuthenticatorResetCompleted",
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
		"ClientSecretRotated",
		"ClientSecretIssued",
		"ClientSecretRevoked",
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

const (
	adminAuditEventExportMaxLimit    = 10000
	listAdminAuditEventsQuery        = "ListAdminAuditEvents"
	listAdminAuditEventsDefaultLimit = 100
	// listAdminAuditEventsMaxLimit is one less than the repository's own
	// max (1000, audit_event_store.go/audit_events.go auditMaxListLimit) so
	// requesting limit+1 rows to detect a next page never exceeds — and so
	// never gets silently re-clamped by — the repository's own limit.
	listAdminAuditEventsMaxLimit = 999
)

// auditScope は問い合わせが扱うテナントの範囲。経路が決めるものであり、リクエストの
// パラメーターからは決して作らない。範囲を切り替える入力を持たせると、その入力の
// 書き漏らし 1 つがテナント境界の穴になる (REQ-AUDIT-001 / REQ-AUDIT-007)。
type auditScope struct {
	tenantID   string
	allTenants bool
}

func tenantAuditScope(tenantID string) auditScope { return auditScope{tenantID: tenantID} }

func systemAuditScope() auditScope { return auditScope{allTenants: true} }

// cursorKey はカーソルの指紋へ混ぜる範囲の識別子。制御面主体はテナント管理経路でも
// 所属テナントが制御面テナントなので、これを混ぜないと両経路の指紋が一致し、テナント内で
// 発行したカーソルが横断検索の続きとして読めてしまう。
func (s auditScope) cursorKey() string {
	if s.allTenants {
		return "system"
	}
	return "tenant:" + s.tenantID
}

// auditEventQueryHash fingerprints the scope together with every filter/sort
// query param (everything except cursor/limit, which are pagination controls,
// not filter identity) so a cursor issued for one scope and filter combination
// is rejected if the caller changes either before following it (wi-159).
func auditEventQueryHash(c *echo.Context, scope auditScope) string {
	q := c.Request().URL.Query()
	q.Del("cursor")
	q.Del("limit")
	return scope.cursorKey() + "|" + q.Encode()
}

func (d Deps) handleListAdminAuditEvents(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.listAuditEvents(c, actor, tenantAuditScope(actor.TenantID))
}

func (d Deps) handleListSystemAuditEvents(c *echo.Context) error {
	actor, err := d.RequireControlPlaneUser(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.listAuditEvents(c, actor, systemAuditScope())
}

func (d Deps) listAuditEvents(c *echo.Context, actor *userdomain.User, scope auditScope) error {
	query, noMatch, err := d.parseAuditEventQuery(c, actor, scope)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	page, err := support.ParsePageRequest(c, d.PaginationCodec, actor.TenantID, auditEventQueryHash(c, scope), listAdminAuditEventsDefaultLimit, listAdminAuditEventsMaxLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if d.AuditEventRepo == nil || noMatch {
		metadata := support.CalculatePaginationMetadata(0, page)
		support.SetPaginationHeaders(c, metadata)
		return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": []AdminAuditEventResponse{}})
	}
	query.Limit = page.Limit + 1
	query.FromEnd = page.Anchor == support.PageAnchorEnd
	if page.AfterPrimary != "" {
		afterOccurredAt, err := time.Parse(time.RFC3339Nano, page.AfterPrimary)
		if err != nil {
			return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "cursor is invalid, expired, or does not match this tenant/query.")
		}
		if page.Direction == support.PageBackward {
			query.BeforeOccurredAt = afterOccurredAt
			query.BeforeID = page.AfterID
		} else {
			query.AfterOccurredAt = afterOccurredAt
			query.AfterID = page.AfterID
		}
	}
	ctx := c.Request().Context()
	var records []*auditports.AuditEventRecord
	var pageErr, countErr error
	var totalItems int64
	var wg sync.WaitGroup
	wg.Go(func() { records, pageErr = d.AuditEventRepo.List(ctx, query) })
	wg.Go(func() { totalItems, countErr = d.AuditEventRepo.Count(ctx, query) })
	wg.Wait()
	if pageErr != nil {
		return support.WriteServerError(c, pageErr)
	}
	if countErr != nil {
		return support.WriteServerError(c, countErr)
	}
	records, hasPrevious, hasNext := support.TrimPage(records, page)
	if page.Anchor == support.PageAnchorEnd {
		records = support.TrimEndPage(records, totalItems, page.Limit)
	}
	metadata := support.CalculatePaginationMetadata(totalItems, page)
	support.SetPaginationHeaders(c, metadata)
	response := make([]AdminAuditEventResponse, len(records))
	for i, rec := range records {
		response[i] = toAdminAuditEventResponse(rec)
	}
	var firstPrimary, firstID, lastPrimary, lastID string
	if len(records) > 0 {
		first := records[0]
		last := records[len(records)-1]
		firstPrimary, firstID = first.OccurredAt.UTC().Format(time.RFC3339Nano), first.ID
		lastPrimary, lastID = last.OccurredAt.UTC().Format(time.RFC3339Nano), last.ID
	}
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, actor.TenantID, auditEventQueryHash(c, scope), page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"events": response})
}

func (d Deps) handleGetAdminAuditEvent(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.getAuditEvent(c, tenantAuditScope(actor.TenantID))
}

func (d Deps) handleGetSystemAuditEvent(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.getAuditEvent(c, systemAuditScope())
}

func (d Deps) getAuditEvent(c *echo.Context, scope auditScope) error {
	if d.AuditEventRepo == nil {
		return writeAuditEventNotFound(c)
	}
	rec, err := d.AuditEventRepo.FindByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return support.WriteServerError(c, err)
	}
	if rec == nil {
		return writeAuditEventNotFound(c)
	}
	if !scope.includes(rec) {
		// 別テナントのイベントは存在を隠す。
		return writeAuditEventNotFound(c)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminAuditEventResponse(rec))
}

func writeAuditEventNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "event_not_found", "The audit event does not exist.")
}

func (d Deps) handleExportAdminAuditEvents(c *echo.Context) error {
	actor, err := d.RequireAuditReader(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.exportAuditEvents(c, actor, tenantAuditScope(actor.TenantID))
}

func (d Deps) handleExportSystemAuditEvents(c *echo.Context) error {
	actor, err := d.RequireControlPlaneUser(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.exportAuditEvents(c, actor, systemAuditScope())
}

func (d Deps) exportAuditEvents(c *echo.Context, actor *userdomain.User, scope auditScope) error {
	query, noMatch, err := d.parseAuditEventQuery(c, actor, scope)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	query.Limit = adminAuditEventExportMaxLimit
	var records []*auditports.AuditEventRecord
	if d.AuditEventRepo != nil && !noMatch {
		records, err = d.AuditEventRepo.List(c.Request().Context(), query)
		if err != nil {
			return support.WriteServerError(c, err)
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
	EventTypes      []string `json:"event_types"`
	Outcomes        []string `json:"outcomes"`
	ActorTypes      []string `json:"actor_types"`
	DelegationModes []string `json:"delegation_modes"`
}

// auditEventOutcomeChoices は outcome フィルタの選択肢 (eventOutcome() の分類先と一致させる)。
var auditEventOutcomeChoices = []string{"success", "failure"}

// auditActorTypeChoices は actor.type フィルタの選択肢。
var auditActorTypeChoices = []string{auditports.ActorTypeUser, auditports.ActorTypeAgent}

// auditDelegationModeChoices は delegation.mode フィルタの選択肢。REQ-OAUTH2-049 が定める
// 列挙をそのまま使い、監査側で別の一覧を持たない。
var auditDelegationModeChoices = []string{
	string(oauth2domain.DelegationModeDirect),
	string(oauth2domain.DelegationModeAutonomous),
	string(oauth2domain.DelegationModeOnBehalfOf),
}

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
		EventTypes:      eventTypes,
		Outcomes:        auditEventOutcomeChoices,
		ActorTypes:      auditActorTypeChoices,
		DelegationModes: auditDelegationModeChoices,
	})
}

// parseAuditEventQuery は query string を AuditEventQuery へ変換する。テナントの範囲は
// 引数の scope が決め、query string は絞り込みにしか使わない。第 2 戻り値 noMatch が
// true の場合、username が実アカウントに解決できなかったことを示し、呼び出し側は
// AuditEventRepo.List を呼ばず空の結果を返す (フィルタ無視で全件返すという誤動作を避ける)。
func (d Deps) parseAuditEventQuery(c *echo.Context, actor *userdomain.User, scope auditScope) (auditports.AuditEventQuery, bool, error) {
	q := auditports.AuditEventQuery{
		TenantID:   scope.tenantID,
		AllTenants: scope.allTenants,
	}
	if t := c.QueryParam("type"); t != "" {
		q.Type = t
	}
	// category はイベントカテゴリ絞り込み (wi-44 統合: 認証サブ分類 + 管理操作カテゴリ)。
	if category := c.QueryParam("category"); category != "" {
		types, ok := auditEventCategoryTypes[category]
		if !ok {
			return auditports.AuditEventQuery{}, false, errors.New("category is invalid")
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
			return auditports.AuditEventQuery{}, false, errors.New("after must be in RFC 3339 format")
		}
		q.After = t
	}
	if before := c.QueryParam("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return auditports.AuditEventQuery{}, false, errors.New("before must be in RFC 3339 format")
		}
		q.Before = t
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil || limit < 0 {
			return auditports.AuditEventQuery{}, false, errors.New("limit must be a non-negative integer")
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
// よう先頭 2 個の ":" で切る。により PII 属性の hash/truncate transform はしない (平文一致)。
func (d Deps) parseAuditFilters(c *echo.Context) ([]auditports.AuditFilterExpression, error) {
	raw := c.Request().URL.Query()["filter"]
	if len(raw) == 0 {
		return nil, nil
	}
	parsed := make([]auditusecases.RawFilter, 0, len(raw))
	for _, token := range raw {
		parts := strings.SplitN(token, ":", 3)
		if len(parts) != 3 {
			return nil, errors.New("filter must use the field:operator:value format")
		}
		parsed = append(parsed, auditusecases.RawFilter{
			Field:    parts[0],
			Operator: parts[1],
			Values:   strings.Split(parts[2], ","),
		})
	}
	return auditusecases.ParseAuditFilter(parsed)
}

// includes は 1 件参照でイベントがこの範囲から見えるかを返す。範囲外は存在を隠す。
func (s auditScope) includes(rec *auditports.AuditEventRecord) bool {
	return s.allTenants || (rec != nil && rec.TenantID == s.tenantID)
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
