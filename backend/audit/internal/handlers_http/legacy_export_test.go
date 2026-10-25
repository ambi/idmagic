package handlers_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

type legacyExportEvents struct {
	ports.AuditEventRepository
	record *ports.AuditEventRecord
	calls  int
}

func (r *legacyExportEvents) List(context.Context, ports.AuditEventQuery) ([]*ports.AuditEventRecord, error) {
	r.calls++
	return []*ports.AuditEventRecord{r.record}, nil
}

//spec:covers REQ-AUDIT-012: 二つの同期枠が使用中なら読まずに拒否し、バイト超過でも部分 JSON を送らない。
func TestLegacyAuditExportConcurrencyAndByteBounds(t *testing.T) {
	repo := &legacyExportEvents{record: &ports.AuditEventRecord{ID: "event", TenantID: "a", Type: "UserAuthenticated", OccurredAt: time.Now(), Payload: map[string]any{"data": strings.Repeat("x", 17<<20)}}}
	deps := Deps{AuditEventRepo: repo, Exports: usecases.ExportDeps{Events: repo, Codec: support.NewCursorCodec([]byte("test-export"))}, ExportPolicy: domain.DefaultExportPolicy()}
	actor := &support.Principal{ID: "owner", TenantID: "a"}
	for range 2 {
		legacyExportSlots <- struct{}{}
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest("GET", "/api/admin/v1/audit-events/export", http.NoBody), rec)
	err := deps.exportAuditEvents(c, actor, tenantAuditScope("a"))
	for range 2 {
		<-legacyExportSlots
	}
	if err != nil || rec.Code != 429 || repo.calls != 0 {
		t.Fatalf("busy status=%d calls=%d err=%v", rec.Code, repo.calls, err)
	}
	rec = httptest.NewRecorder()
	c = echo.New().NewContext(httptest.NewRequest("GET", "/api/admin/v1/audit-events/export", http.NoBody), rec)
	if err := deps.exportAuditEvents(c, actor, tenantAuditScope("a")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 422 || strings.Contains(rec.Body.String(), `"events"`) {
		t.Fatalf("byte limit status=%d body=%s", rec.Code, rec.Body.String())
	}
}
