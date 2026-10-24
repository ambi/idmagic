package usecases

import (
	"context"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
)

//spec:covers REQ-DIRECTORYCONNECTOR-004, REQ-DIRECTORYCONNECTOR-006: 起点を並べた順に検索の範囲の全件を読み、写せた項目の数と送らないエントリを返し、IdMagic を呼ばず、位置を読み書きしない。
func TestCheckCountsTheItemsWithoutSending(t *testing.T) {
	h := newHarness(t, nil)
	incomplete := domain.Entry{DN: "uid=nouuid,ou=staff", Attributes: []domain.Attribute{{Name: "uid", Values: [][]byte{[]byte("nouuid")}}}}
	h.dir.byBase = map[string][]domain.Entry{"ou=staff": append(people(3), incomplete), "ou=contractors": people(2)}
	scope := domain.Scope{BaseDNs: []string{"ou=staff", "ou=contractors"}, Filter: domain.DefaultUserFilter(domain.OpenLDAP)}

	result, err := Check(context.Background(), h.dir, domain.OpenLDAP, scope, h.syncer.Mapping)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items != 5 || len(result.Skipped) != 1 || result.Skipped[0].DN != incomplete.DN || result.Skipped[0].Attribute != "entryUUID" {
		t.Fatalf("result = %+v", result)
	}
	if !slices.Equal(h.dir.bases, scope.BaseDNs) || !slices.Equal(h.dir.filters, []string{scope.Filter, scope.Filter}) {
		t.Fatalf("searched bases=%v filters=%v, want every base with the full filter", h.dir.bases, h.dir.filters)
	}
	if len(h.api.calls) != 0 || h.cursors.cursor != nil {
		t.Fatalf("calls=%v cursor=%+v, want nothing sent or saved", h.api.calls, h.cursors.cursor)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-004: 同期は複数の起点を並べた順に読み、一つの実行として送る。
func TestSyncReadsEveryBaseDNInOneRun(t *testing.T) {
	h := newHarness(t, nil)
	h.dir.byBase = map[string][]domain.Entry{"ou=staff": people(60), "ou=contractors": peopleFrom(1000, 10)}
	h.syncer.Scope.BaseDNs = []string{"ou=staff", "ou=contractors"}
	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"heartbeat", "start full", "batch run1-1", "complete run1 70"}; !slices.Equal(h.api.calls, want) {
		t.Fatalf("calls = %v, want %v", h.api.calls, want)
	}
	if got := h.api.batches["run1-1"]; got[60].ExternalID != "uuid-1000" {
		t.Fatalf("second base starts at %s, want the contractors after the staff", got[60].ExternalID)
	}
}
