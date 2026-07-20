package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestStepUpSatisfiedRecencyWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		ctx  *authdomain.AuthenticationContext
		want bool
	}{
		{"fresh auth", &authdomain.AuthenticationContext{AuthTime: now.Add(-time.Minute).Unix()}, true},
		{"stale auth", &authdomain.AuthenticationContext{AuthTime: now.Add(-10 * time.Minute).Unix()}, false},
		{
			"stale auth but recent step-up",
			&authdomain.AuthenticationContext{
				AuthTime: now.Add(-10 * time.Minute).Unix(), StepUpAt: now.Add(-2 * time.Minute).Unix(),
			},
			true,
		},
		{"boundary 300s", &authdomain.AuthenticationContext{AuthTime: now.Add(-300 * time.Second).Unix()}, true},
		{"just over 300s", &authdomain.AuthenticationContext{AuthTime: now.Add(-301 * time.Second).Unix()}, false},
		{"pending never", &authdomain.AuthenticationContext{AuthTime: now.Unix(), AuthenticationPending: true}, false},
		{"zero times", &authdomain.AuthenticationContext{}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := StepUpSatisfied(tc.ctx, now); got != tc.want {
			t.Errorf("%s: StepUpSatisfied = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAvailableStepUpMethods(t *testing.T) {
	t.Parallel()
	if got := AvailableStepUpMethods(&userdomain.User{}); len(got) != 1 || got[0] != StepUpMethodPassword {
		t.Fatalf("no MFA: got %v", got)
	}
	got := AvailableStepUpMethods(&userdomain.User{MfaEnrolled: true})
	if len(got) != 2 || got[1] != StepUpMethodTOTP {
		t.Fatalf("MFA enrolled: got %v", got)
	}
}

func newStepUpFixture(t *testing.T, now time.Time) (StepUpDeps, *sessionusecases.SessionManager, *[]spec.DomainEvent) {
	t.Helper()
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	hasher := passwords_argon2id.NewArgon2idPasswordHasher()
	hash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Save(ctx, &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: hash, MfaEnrolled: true,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	secret, err := totpusecases.GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	mfaRepo := totpmemory.NewMfaFactorRepository()
	if err := mfaRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: "user-1", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	// in-memory SessionStore は既定で実時計により期限切れ判定するため、固定 now の
	// テストでセッションが失効しないよう時計を now に固定する。
	sessionStore := sessionmemory.NewSessionStore()
	sessionStore.Clock = func() time.Time { return now }
	sm := sessionusecases.NewSessionManager(sessionStore)
	deps := StepUpDeps{
		UserRepo: userRepo, PasswordHasher: hasher, MfaFactorRepo: mfaRepo, SessionManager: sm,
		Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}
	return deps, sm, &events
}

func TestCompleteStepUpPasswordRecordsAndEmits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	deps, sm, events := newStepUpFixture(t, now)
	authn, err := sm.CreateWithPending(ctx, "user-1", []string{"pwd"}, now.Add(-30*time.Minute), false)
	if err != nil {
		t.Fatal(err)
	}

	if err := CompleteStepUp(ctx, deps, CompleteStepUpInput{
		Sub: "user-1", SessionID: authn.SessionID, Method: StepUpMethodPassword,
		Password: "demo-password-1234", Now: now,
	}); err != nil {
		t.Fatalf("complete step-up: %v", err)
	}
	sess, _ := sm.Store.Find(ctx, authn.SessionID)
	if sess.StepUpAt != now.Unix() {
		t.Fatalf("step_up_at=%d, want %d", sess.StepUpAt, now.Unix())
	}
	// 刻んだ後は recency 窓内なので gate を通過する。
	if !StepUpSatisfied(&authdomain.AuthenticationContext{AuthTime: authn.AuthTime, StepUpAt: sess.StepUpAt}, now) {
		t.Fatal("expected step-up to satisfy gate after completion")
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	completed, ok := (*events)[0].(*authdomain.StepUpCompleted)
	if !ok || completed.Method != "password" || completed.UserID != "user-1" {
		t.Fatalf("unexpected event %#v", (*events)[0])
	}
}

func TestCompleteStepUpWrongPasswordFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	deps, sm, _ := newStepUpFixture(t, now)
	authn, _ := sm.CreateWithPending(ctx, "user-1", []string{"pwd"}, now.Add(-30*time.Minute), false)

	err := CompleteStepUp(ctx, deps, CompleteStepUpInput{
		Sub: "user-1", SessionID: authn.SessionID, Method: StepUpMethodPassword,
		Password: "wrong-password", Now: now,
	})
	if !errors.Is(err, ErrStepUpFailed) {
		t.Fatalf("err=%v, want ErrStepUpFailed", err)
	}
	sess, _ := sm.Store.Find(ctx, authn.SessionID)
	if sess.StepUpAt != 0 {
		t.Fatal("step_up_at must stay unset on failure")
	}
}

func TestCompleteStepUpTOTPSucceeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	deps, sm, _ := newStepUpFixture(t, now)
	authn, _ := sm.CreateWithPending(ctx, "user-1", []string{"pwd"}, now.Add(-30*time.Minute), false)

	factor, err := deps.MfaFactorRepo.Find(ctx, "user-1", spec.MfaFactorTOTP)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totpusecases.GenerateTOTP(*factor.Secret, now.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if err := CompleteStepUp(ctx, deps, CompleteStepUpInput{
		Sub: "user-1", SessionID: authn.SessionID, Method: StepUpMethodTOTP, Code: code, Now: now,
	}); err != nil {
		t.Fatalf("complete TOTP step-up: %v", err)
	}
	sess, _ := sm.Store.Find(ctx, authn.SessionID)
	if sess.StepUpAt != now.Unix() {
		t.Fatal("expected step_up_at recorded for TOTP")
	}
}
