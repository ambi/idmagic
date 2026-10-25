package handlers_http_test

import (
	"bytes"
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"
)

const (
	quotaContractID    = "00000000-0000-4000-8000-000000000471"
	quotaContractRealm = "quota-contract"
)

func newQuotaContractServer(t *testing.T) (*rulesServer, *observedQuotaRepository) {
	t.Helper()
	quotas := &observedQuotaRepository{QuotaRepository: memory.NewQuotaRepository()}
	users := 100
	for _, key := range []string{quotaContractID, quotaContractRealm} {
		if err := quotas.QuotaRepository.SetQuota(t.Context(), key, &domain.TenantQuota{Users: &users}); err != nil {
			t.Fatal(err)
		}
	}
	s := newRulesServer(t, systemAdmin(), rulesOptions{quotas: quotas})
	if err := s.tenants.Save(t.Context(), &domain.Tenant{
		ID: quotaContractID, Realm: quotaContractRealm, DisplayName: "Quota contract",
		Status: domain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	return s, quotas
}

func assertQuotaUsers(t *testing.T, quotas *observedQuotaRepository, key string, want int) {
	t.Helper()
	quota, err := quotas.GetQuota(context.Background(), key)
	if err != nil || quota == nil || quota.Users == nil || *quota.Users != want {
		t.Fatalf("quota[%s] = %+v, error = %v, want users %d", key, quota, err, want)
	}
}

//spec:covers REQ-TENANCY-012, REQ-TENANCY-046: realm と内部 ID が異なっても内部 ID に保存し、監査の対象と JSON 応答が一致し、no-store を伴う。
//verify:operation UpdateTenantQuota: realm を送る PUT が保存した users/groups の 200 JSON と no-store を返す。
func TestTenantQuotaResolvesRealmAndReturnsStoredQuota(t *testing.T) {
	s, quotas := newQuotaContractServer(t)
	rec := s.send(t, http.MethodPut, tenantsPath+"/"+quotaContractRealm+"/quota", map[string]int{"users": 120, "groups": 8})
	requireStatus(t, rec, http.StatusOK)
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	body := decodeJSON(t, rec)
	if body["users"] != float64(120) || body["groups"] != float64(8) {
		t.Errorf("quota response = %v, want users 120, groups 8", body)
	}
	assertQuotaUsers(t, quotas, quotaContractID, 120)
	assertQuotaUsers(t, quotas, quotaContractRealm, 100)
	stored, err := quotas.GetQuota(t.Context(), quotaContractID)
	if err != nil || stored.Groups == nil || *stored.Groups != 8 {
		t.Fatalf("stored quota = %+v, error = %v", stored, err)
	}
	if len(*s.events) != 1 {
		t.Fatalf("events = %v, want one quota update", *s.events)
	}
	event, ok := (*s.events)[0].(*domain.TenantQuotaUpdated)
	if !ok || event.TenantID != quotaContractID || !reflect.DeepEqual(event.ChangedFields, []string{"users", "groups"}) {
		t.Fatalf("event = %+v, want resolved tenant ID and changed fields", (*s.events)[0])
	}
}

//spec:covers REQ-TENANCY-047: TenantQuotaUpdateRequest の Soft の 0 と 2 GiB 超の上書きを TenantQuota に保存し、その値の後も TenantUsage を加算できる。
//verify:operation UpdateTenantQuota: Soft の上書きの 200 JSON と保存した値を照合する。
func TestTenantQuotaKeepsSoftOverridesWithoutRefusingUsage(t *testing.T) {
	s, quotas := newQuotaContractServer(t)
	const large = 3 << 30
	rec := s.send(t, http.MethodPut, tenantsPath+"/"+quotaContractRealm+"/quota", map[string]int{"audit_events_retained": 0, "export_artifacts_bytes": large})
	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON(t, rec)
	if body["audit_events_retained"] != float64(0) || body["export_artifacts_bytes"] != float64(large) {
		t.Fatalf("soft overrides=%v", body)
	}
	stored, err := quotas.GetQuota(t.Context(), quotaContractID)
	if err != nil || stored.EffectiveLimit(domain.ResourceAuditEventsRetained) != 0 || stored.EffectiveLimit(domain.ResourceExportArtifactsBytes) != large {
		t.Fatalf("quota=%+v err=%v", stored, err)
	}
	if err := quotas.CheckAndIncrement(t.Context(), quotaContractID, domain.ResourceAuditEventsRetained, 1); err != nil {
		t.Fatal(err)
	}
	usage, err := quotas.GetUsage(t.Context(), quotaContractID)
	if err != nil || usage.AuditEventsRetained != 1 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
}

//spec:covers REQ-TENANCY-046: 不在の realm と内部 UUID は本文の妥当性によらず 404 となり、保存とイベントを行わない。
//verify:operation UpdateTenantQuota: 未知 realm と内部 UUID に対する PUT は 404 tenant_not_found の Problem Details を返す。
func TestTenantQuotaRefusesMissingRealm(t *testing.T) {
	for _, target := range []string{"missing", quotaContractID} {
		for _, body := range []string{`{"users":120}`, `{"users":`} {
			t.Run(target+body, func(t *testing.T) {
				s, quotas := newQuotaContractServer(t)
				rec := s.sendRaw(t, http.MethodPut, tenantsPath+"/"+target+"/quota", "application/json", bytes.NewReader([]byte(body)))
				assertTenancyProblem(t, rec, http.StatusNotFound, "tenant_not_found")
				assertQuotaUsers(t, quotas, quotaContractID, 100)
				assertQuotaUsers(t, quotas, quotaContractRealm, 100)
				if quotas.setQuotaCalls.Load() != 0 || len(*s.events) != 0 {
					t.Fatal("missing realm caused a save or an event")
				}
			})
		}
	}
}

//spec:covers REQ-PLATFORM-005, REQ-TENANCY-012: 不正な JSON と 64 KiB を超える本文は保存とイベント発行より前に拒否する。
//verify:operation UpdateTenantQuota: 不正な JSON と過大な本文の PUT は 400 invalid_request の Problem Details を返す。
func TestTenantQuotaRefusesInvalidBody(t *testing.T) {
	for name, body := range map[string]string{"malformed": `{"users":`, "oversized": `{"users":120}` + strings.Repeat(" ", 64<<10)} {
		t.Run(name, func(t *testing.T) {
			s, quotas := newQuotaContractServer(t)
			rec := s.sendRaw(t, http.MethodPut, tenantsPath+"/"+quotaContractRealm+"/quota", "application/json", bytes.NewReader([]byte(body)))
			assertTenancyProblem(t, rec, http.StatusBadRequest, "invalid_request")
			assertQuotaUsers(t, quotas, quotaContractID, 100)
			if quotas.setQuotaCalls.Load() != 0 || len(*s.events) != 0 {
				t.Fatal("invalid body caused a save or an event")
			}
		})
	}
}

//spec:covers REQ-SYSTEM-023, REQ-TENANCY-012: 保存失敗の内部エラーは応答へ出さず、監査イベントを発行しない。
//verify:operation UpdateTenantQuota: 保存失敗は内部情報を含まない 500 internal_server_error の Problem Details を返す。
func TestTenantQuotaHidesSaveFailure(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{quotas: quotaSaveFailingRepository{QuotaRepository: memory.NewQuotaRepository()}})
	rec := s.send(t, http.MethodPut, tenantsPath+"/acme/quota", map[string]int{"users": 120})
	assertTenancyProblem(t, rec, http.StatusInternalServerError, "internal_server_error")
	if strings.Contains(rec.Body.String(), "quota store unavailable") {
		t.Fatal("save failure disclosed the store error")
	}
	if len(*s.events) != 0 {
		t.Fatal("unsaved quota emitted an event")
	}
}

//spec:covers REQ-PLATFORM-005, REQ-TENANCY-046: 未知フィールドは無視し、既知フィールドだけを解決したテナントへ保存する。
//verify:operation UpdateTenantQuota: 未知フィールドを含む PUT は 200 の既知クォータだけを返し、内部 ID の保存へ反映する。
func TestTenantQuotaIgnoresUnknownFields(t *testing.T) {
	s, quotas := newQuotaContractServer(t)
	rec := s.send(t, http.MethodPut, tenantsPath+"/"+quotaContractRealm+"/quota", map[string]int{"users": 120, "future_resource": 7})
	requireStatus(t, rec, http.StatusOK)
	if body := decodeJSON(t, rec); !reflect.DeepEqual(body, map[string]any{"users": float64(120)}) {
		t.Fatalf("quota response = %v, want only users 120", body)
	}
	assertQuotaUsers(t, quotas, quotaContractID, 120)
}

//spec:covers REQ-TENANCY-012: 通常の管理者と制御面外の system_admin は対象や本文によらず拒否され、保存済みの上限とイベントを変えない。
//verify:operation UpdateTenantQuota: 通常の管理者は 403 access_denied、制御面で認証できない別所属の主体は 401 authentication_required の Problem Details で返す。
func TestTenantQuotaRefusesNonSystemAdmin(t *testing.T) {
	for _, tc := range []struct {
		actor  *userdomain.User
		status int
		code   string
	}{
		{settingsActor("tenant-admin", domain.DefaultTenantID, []string{"admin"}), http.StatusForbidden, "access_denied"},
		{settingsActor("foreign-system-admin", "acme", []string{"system_admin"}), http.StatusUnauthorized, "authentication_required"},
	} {
		t.Run(tc.actor.ID, func(t *testing.T) {
			quotas := &observedQuotaRepository{QuotaRepository: memory.NewQuotaRepository()}
			users := 100
			if err := quotas.QuotaRepository.SetQuota(t.Context(), "acme", &domain.TenantQuota{Users: &users}); err != nil {
				t.Fatal(err)
			}
			s := newRulesServer(t, tc.actor, rulesOptions{quotas: quotas})
			for _, target := range []string{"acme", "missing"} {
				rec := s.sendRaw(t, http.MethodPut, tenantsPath+"/"+target+"/quota", "application/json", bytes.NewReader([]byte(`{"users":`)))
				assertTenancyProblem(t, rec, tc.status, tc.code)
			}
			assertQuotaUsers(t, quotas, "acme", 100)
			if quotas.setQuotaCalls.Load() != 0 || len(*s.events) != 0 {
				t.Fatal("unauthorized request caused a save or an event")
			}
		})
	}
}
