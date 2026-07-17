package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"

	"github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newRecoveryDeps(t *testing.T) (usecases.RecoveryCodesDeps, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := idmmemory.NewUserRepository()
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&idmdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	var events []spec.DomainEvent
	deps := usecases.RecoveryCodesDeps{
		UserRepo:         userRepo,
		RecoveryCodeRepo: authnmemory.NewRecoveryCodeRepository(),
		Emit:             func(e spec.DomainEvent) { events = append(events, e) },
	}
	return deps, &events
}

func TestGenerateRecoveryCodesReturnsPlaintextOnce(t *testing.T) {
	ctx := context.Background()
	deps, events := newRecoveryDeps(t)
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	result, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Codes) != 10 {
		t.Fatalf("code count=%d, want 10", len(result.Codes))
	}
	seen := map[string]bool{}
	for _, c := range result.Codes {
		if len(c) != 10 {
			t.Fatalf("code length=%d, want 10 (%q)", len(c), c)
		}
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
	}
	status, err := usecases.RecoveryCodeStatusFor(ctx, deps.RecoveryCodeRepo, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if status.Total != 10 || status.Remaining != 10 {
		t.Fatalf("status=%#v, want total/remaining=10", status)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "RecoveryCodesGenerated" {
		t.Fatalf("unexpected events: %#v", *events)
	}
}

func TestConsumeRecoveryCodeSingleUse(t *testing.T) {
	ctx := context.Background()
	deps, events := newRecoveryDeps(t)
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	result, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now)
	if err != nil {
		t.Fatal(err)
	}
	code := result.Codes[0]

	remaining, err := usecases.ConsumeRecoveryCode(ctx, deps, "user-alice", code, now)
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 9 {
		t.Fatalf("remaining=%d, want 9", remaining)
	}
	// 再利用は拒否される。
	if _, err := usecases.ConsumeRecoveryCode(ctx, deps, "user-alice", code, now); !errors.Is(err, usecases.ErrRecoveryCodeInvalid) {
		t.Fatalf("reuse error=%v, want ErrRecoveryCodeInvalid", err)
	}
	// 未知コードも拒否される。
	if _, err := usecases.ConsumeRecoveryCode(ctx, deps, "user-alice", "zzzzzzzzzz", now); !errors.Is(err, usecases.ErrRecoveryCodeInvalid) {
		t.Fatalf("unknown error=%v, want ErrRecoveryCodeInvalid", err)
	}
	last := (*events)[len(*events)-1]
	if last.EventType() != "BackupCodeConsumed" {
		t.Fatalf("last event=%s, want BackupCodeConsumed", last.EventType())
	}
}

func TestConsumeRecoveryCodeIgnoresFormattingAndCase(t *testing.T) {
	ctx := context.Background()
	deps, _ := newRecoveryDeps(t)
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	result, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now)
	if err != nil {
		t.Fatal(err)
	}
	// 表示上のハイフン / 大文字 / 空白を含めても正規化して一致する。
	decorated := " " + strings.ToUpper(result.Codes[0][:5]) + "-" + strings.ToUpper(result.Codes[0][5:]) + " "
	if _, err := usecases.ConsumeRecoveryCode(ctx, deps, "user-alice", decorated, now); err != nil {
		t.Fatalf("normalized consume failed: %v", err)
	}
}

func TestRegenerateReplacesExistingSet(t *testing.T) {
	ctx := context.Background()
	deps, _ := newRecoveryDeps(t)
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	first, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now); err != nil {
		t.Fatal(err)
	}
	// 旧 set のコードはもう使えない。
	if _, err := usecases.ConsumeRecoveryCode(ctx, deps, "user-alice", first.Codes[0], now); !errors.Is(err, usecases.ErrRecoveryCodeInvalid) {
		t.Fatalf("old code error=%v, want ErrRecoveryCodeInvalid", err)
	}
	status, _ := usecases.RecoveryCodeStatusFor(ctx, deps.RecoveryCodeRepo, "user-alice")
	if status.Total != 10 || status.Remaining != 10 {
		t.Fatalf("status after regenerate=%#v", status)
	}
}

func TestRevokeRecoveryCodes(t *testing.T) {
	ctx := context.Background()
	deps, events := newRecoveryDeps(t)
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	if _, err := usecases.GenerateRecoveryCodes(ctx, deps, "user-alice", now); err != nil {
		t.Fatal(err)
	}
	if err := usecases.RevokeRecoveryCodes(ctx, deps, "user-alice", now); err != nil {
		t.Fatal(err)
	}
	status, _ := usecases.RecoveryCodeStatusFor(ctx, deps.RecoveryCodeRepo, "user-alice")
	if status.Total != 0 || status.Remaining != 0 {
		t.Fatalf("status after revoke=%#v", status)
	}
	last := (*events)[len(*events)-1]
	if last.EventType() != "RecoveryCodesRevoked" {
		t.Fatalf("last event=%s, want RecoveryCodesRevoked", last.EventType())
	}
}
