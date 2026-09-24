package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-002、REQ-PROVISIONING-014。

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// eventRecorder は発行ポートに渡したイベントを発行順に残す。
type eventRecorder struct{ events []spec.DomainEvent }

func (r *eventRecorder) emit(event spec.DomainEvent) { r.events = append(r.events, event) }

func (r *eventRecorder) types() []string {
	types := make([]string, 0, len(r.events))
	for _, event := range r.events {
		types = append(types, event.EventType())
	}
	return types
}

//spec:covers EX-PROVISIONING-014-01: 資格情報を渡した更新だけが、保存した新しい credential_id を持つ ProvisioningCredentialRotated を発行する。
func TestAdminConnectionOperationsEmitTheirEventAfterSaving(t *testing.T) {
	ctx := context.Background()
	deps, connRepo, _ := newAdminDeps()
	recorder := &eventRecorder{}
	deps.Emit = recorder.emit
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	register := usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	}

	if _, err := usecases.RegisterConnection(ctx, deps, register); err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}
	registered, ok := onlyEvent[*domain.ProvisioningConnectionRegistered](t, recorder)
	if !ok || registered.TenantID != "tenant-a" || registered.ApplicationID != "app-1" || !registered.At.Equal(now) {
		t.Fatalf("registration event = %+v, want tenant-a/app-1 at %v", registered, now)
	}

	// 保存が拒否された登録は何も発行しない。
	if _, err := usecases.RegisterConnection(ctx, deps, register); err == nil {
		t.Fatal("RegisterConnection() duplicate: want error")
	}
	// 資格情報を渡さない更新はローテーションではない。
	maxAttempts := 5
	if _, err := usecases.UpdateConnection(ctx, deps, usecases.UpdateConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", MaxAttempts: &maxAttempts, Now: now,
	}); err != nil {
		t.Fatalf("UpdateConnection() without credential error = %v", err)
	}
	if got := recorder.types(); len(got) != 1 {
		t.Fatalf("events after a refused registration and a settings update = %v, want only the first registration", got)
	}

	recorder.events = nil
	rotatedAt := now.Add(time.Hour)
	if _, err := usecases.UpdateConnection(ctx, deps, usecases.UpdateConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1",
		Credential: &domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok-2"},
		Now:        rotatedAt,
	}); err != nil {
		t.Fatalf("UpdateConnection() with credential error = %v", err)
	}
	saved, err := connRepo.Find(ctx, "tenant-a", "app-1")
	if err != nil || saved == nil {
		t.Fatalf("Find() = %v, %v", saved, err)
	}
	rotated, ok := onlyEvent[*domain.ProvisioningCredentialRotated](t, recorder)
	if !ok || rotated.CredentialID != saved.Credential.CredentialID || rotated.ApplicationID != "app-1" || !rotated.At.Equal(rotatedAt) {
		t.Fatalf("rotation event = %+v, want credential %q saved at %v", rotated, saved.Credential.CredentialID, rotatedAt)
	}
}

func TestResumeConnectionEmitsQuarantineClearedOnlyWhenItLiftsAQuarantine(t *testing.T) {
	ctx := context.Background()
	deps, connRepo, _ := newAdminDeps()
	recorder := &eventRecorder{}
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	if _, err := usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	}); err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}
	deps.Emit = recorder.emit

	if _, err := usecases.ResumeConnection(ctx, deps, "tenant-a", "app-1", now); err == nil {
		t.Fatal("ResumeConnection() on a healthy connection: want error")
	}
	if len(recorder.events) != 0 {
		t.Fatalf("events after a refused resume = %v, want none", recorder.types())
	}

	conn, _ := connRepo.Find(ctx, "tenant-a", "app-1")
	if err := conn.Quarantine("too many failures", now); err != nil {
		t.Fatalf("Quarantine() error = %v", err)
	}
	if err := connRepo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	resumedAt := now.Add(time.Hour)
	if _, err := usecases.ResumeConnection(ctx, deps, "tenant-a", "app-1", resumedAt); err != nil {
		t.Fatalf("ResumeConnection() error = %v", err)
	}
	saved, _ := connRepo.Find(ctx, "tenant-a", "app-1")
	cleared, ok := onlyEvent[*domain.ProvisioningConnectionQuarantineCleared](t, recorder)
	if !ok || saved.Health != domain.HealthOK || cleared.ApplicationID != "app-1" || !cleared.At.Equal(resumedAt) {
		t.Fatalf("resume: health = %v, event = %+v; want ok and one QuarantineCleared for app-1 at %v", saved.Health, cleared, resumedAt)
	}
}

// onlyEvent は記録がちょうど 1 件で、その型が E であることを確かめて返す。
func onlyEvent[E spec.DomainEvent](t *testing.T, recorder *eventRecorder) (E, bool) {
	t.Helper()
	var zero E
	if len(recorder.events) != 1 {
		t.Errorf("events = %v, want exactly one", recorder.types())
		return zero, false
	}
	event, ok := recorder.events[0].(E)
	if !ok {
		t.Errorf("event = %T, want %T", recorder.events[0], zero)
	}
	return event, ok
}
