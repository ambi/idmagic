package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// ComputerFixture はコンピューターの保存先と、外部キーを満たす Connector を作る関数である。
type ComputerFixture struct {
	Computers     ports.ComputerRepository
	TenantID      string
	OtherTenantID string
	NewConnector  func(t *testing.T, tenantID string) string
}

// RunComputers は REQ-SOURCING-018 と REQ-SOURCING-020 が保存先に求める、置き換え、有効な記録の間の
// SID と主体名の一意性、大文字と小文字を区別しない主体名の照会、テナントの範囲、現れなかった記録の無効化を確かめる。
func RunComputers(t *testing.T, f ComputerFixture) {
	t.Helper()
	ctx := context.Background()
	at := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	connector := f.NewConnector(t, f.TenantID)
	sibling := f.NewConnector(t, f.TenantID)
	foreign := f.NewConnector(t, f.OtherTenantID)
	first, second := uuid.NewString(), uuid.NewString()
	computer := func(tenantID, connectorID, externalID, sam, sid string) domain.Computer {
		return domain.Computer{
			TenantID: tenantID, ConnectorID: connectorID, LastSeenRunID: first, UpdatedAt: at,
			ExternalID: externalID, Enabled: true, SAMAccountName: sam, ObjectSID: sid,
			Domain: "Corp.Example.Test", DNSHostName: "ws.corp.example.test",
		}
	}
	save := func(c domain.Computer) {
		t.Helper()
		if err := f.Computers.Save(ctx, c); err != nil {
			t.Fatalf("Save %s: %v", c.ExternalID, err)
		}
	}

	ws1 := computer(f.TenantID, connector, "g1", "WS1$", "S-1-5-21-1-2-3-1101")
	save(ws1)
	save(computer(f.TenantID, connector, "g2", "WS2$", "S-1-5-21-1-2-3-1102"))
	// 別のテナントは同じ値を持てる。
	save(computer(f.OtherTenantID, foreign, "g1", "WS1$", "S-1-5-21-1-2-3-1101"))

	if got, err := f.Computers.Find(ctx, f.TenantID, connector, "g1"); err != nil || got == nil || *got != ws1 {
		t.Fatalf("Find = %+v, %v; want %+v", got, err, ws1)
	}
	if got, err := f.Computers.Find(ctx, f.TenantID, sibling, "g1"); err != nil || got != nil {
		t.Fatalf("Find from another connector = %+v, %v", got, err)
	}
	if got, err := f.Computers.FindEnabledByPrincipal(ctx, f.TenantID, "ws1$", "CORP.EXAMPLE.TEST"); err != nil || got == nil || got.ExternalID != "g1" || got.ConnectorID != connector {
		t.Fatalf("FindEnabledByPrincipal = %+v, %v", got, err)
	}
	if got, err := f.Computers.FindEnabledBySID(ctx, f.TenantID, "S-1-5-21-1-2-3-1102"); err != nil || got == nil || got.ExternalID != "g2" {
		t.Fatalf("FindEnabledBySID = %+v, %v", got, err)
	}
	if got, err := f.Computers.FindEnabledBySID(ctx, f.TenantID, "s-1-5-21-1-2-3-1102"); err != nil || got != nil {
		t.Fatalf("FindEnabledBySID normalized the sid: %+v, %v", got, err)
	}
	holders, err := f.Computers.FindEnabledHolders(ctx, f.TenantID, "S-1-5-21-1-2-3-1102", domain.PrincipalKey("WS1$", "corp.example.test"))
	if err != nil || len(holders) != 2 || holders[0].ExternalID != "g1" || holders[1].ExternalID != "g2" {
		t.Fatalf("FindEnabledHolders = %+v, %v", holders, err)
	}

	for name, clash := range map[string]domain.Computer{
		"same sid":       computer(f.TenantID, sibling, "g9", "OTHER$", "S-1-5-21-1-2-3-1101"),
		"same principal": computer(f.TenantID, sibling, "g9", "ws1$", "S-1-5-21-1-2-3-1199"),
	} {
		if err := f.Computers.Save(ctx, clash); !errors.Is(err, domain.ErrComputerConflict) {
			t.Fatalf("Save with the %s = %v, want ErrComputerConflict", name, err)
		}
	}
	// 無効な記録は値を持たないので、同じ値の無効な記録を置ける。
	disabledClash := computer(f.TenantID, sibling, "g9", "WS1$", "S-1-5-21-1-2-3-1101")
	disabledClash.Enabled = false
	save(disabledClash)

	renamed := ws1
	renamed.SAMAccountName, renamed.UpdatedAt = "LAPTOP1$", at.Add(time.Hour)
	save(renamed)
	if got, _ := f.Computers.FindEnabledByPrincipal(ctx, f.TenantID, "WS1$", "corp.example.test"); got != nil {
		t.Fatalf("the old principal name still finds %+v", got)
	}
	if got, _ := f.Computers.Find(ctx, f.TenantID, connector, "g1"); got == nil || *got != renamed {
		t.Fatalf("Save did not replace the computer: %+v", got)
	}

	if err := f.Computers.MarkSeen(ctx, f.TenantID, connector, "g1", second); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	disabled, err := f.Computers.DisableNotSeenIn(ctx, f.TenantID, connector, second, at.Add(2*time.Hour))
	if err != nil || disabled != 1 {
		t.Fatalf("DisableNotSeenIn = %d, %v; want 1", disabled, err)
	}
	if got, _ := f.Computers.FindEnabledBySID(ctx, f.TenantID, "S-1-5-21-1-2-3-1102"); got != nil {
		t.Fatalf("an unseen computer is still enabled: %+v", got)
	}
	if got, _ := f.Computers.Find(ctx, f.TenantID, connector, "g2"); got == nil || got.Enabled || !got.UpdatedAt.Equal(at.Add(2*time.Hour)) {
		t.Fatalf("the unseen computer = %+v", got)
	}
	if got, _ := f.Computers.FindEnabledBySID(ctx, f.TenantID, "S-1-5-21-1-2-3-1101"); got == nil || got.ExternalID != "g1" {
		t.Fatalf("the seen computer = %+v", got)
	}
	if got, _ := f.Computers.FindEnabledBySID(ctx, f.OtherTenantID, "S-1-5-21-1-2-3-1101"); got == nil || got.ConnectorID != foreign {
		t.Fatalf("another tenant's computer = %+v", got)
	}
}
