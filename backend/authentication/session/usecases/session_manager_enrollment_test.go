package usecases

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	memory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
)

func TestSessionManagerEnrollmentUsesSameSessionAndCompletesMfa(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	store := memory.NewSessionStore()
	store.Clock = func() time.Time { return now }
	manager := NewSessionManager(store)
	created, err := manager.Create(ctx, "user", []string{"pwd"}, now)
	if err != nil {
		t.Fatal(err)
	}
	deadline := now.Add(10 * time.Minute)
	pending, err := manager.RequireEnrollment(ctx, created.SessionID, deadline, "bypass")
	if err != nil {
		t.Fatal(err)
	}
	if pending.SessionID != created.SessionID || pending.PendingPurpose != authdomain.LoginPendingEnrollment || !pending.AuthenticationPending {
		t.Fatalf("pending = %#v", pending)
	}
	completed, err := manager.CompleteFactor(ctx, created.SessionID, []string{"otp"})
	if err != nil {
		t.Fatal(err)
	}
	if completed.SessionID != created.SessionID || completed.AuthenticationPending || completed.ACR != authusecases.ACRMFA {
		t.Fatalf("completed = %#v", completed)
	}
}
