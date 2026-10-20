package usecases

import (
	"context"
	"slices"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnmemory "github.com/ambi/idmagic/backend/authentication/internal/db_memory"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// wi-20 スライス 3: 閾値超過 (throttle Locked) のログイン失敗は個別イベントへ
// 落とさず 5 分窓の bucket に集約する。recordLoginFailure の契約を直接検証する:
//   - bucket store がある構成では aggregated=true を返し、窓の最初の 1 件だけ
//     AuthenticationEventAggregated を emit する (後続は count に積むだけ)。
//   - bucket store が無い構成では aggregated=false を返し、呼び出し側が従来どおり
//     個別の AuthenticationFailed を残せる。

type fakeLockedThrottle struct{}

func (fakeLockedThrottle) TryAcquire(
	context.Context, sessionports.LoginThrottleKind, string, time.Time,
) (sessionports.LoginThrottleResult, error) {
	return sessionports.LoginThrottleResult{Allowed: true}, nil
}

func (fakeLockedThrottle) RecordFailure(
	context.Context, sessionports.LoginThrottleKind, string, time.Time,
) (sessionports.LoginThrottleResult, error) {
	return sessionports.LoginThrottleResult{Allowed: false, Locked: true, RetryAfterSeconds: 900}, nil
}

func (fakeLockedThrottle) RecordSuccess(context.Context, sessionports.LoginThrottleKind, string) error {
	return nil
}

func countEventType(events []spec.DomainEvent, typ string) int {
	n := 0
	for _, e := range events {
		if e.EventType() == typ {
			n++
		}
	}
	return n
}

// driveRecordLoginFailure は既定のテナントの文脈で recordLoginFailure を回数分呼び、
// 最後の aggregated 値を返す (account key だけ渡し IP は空にして 1 key に固定する)。
func driveRecordLoginFailure(t *testing.T, p PasswordAuthentication, username string, times int) (bool, []spec.DomainEvent) {
	t.Helper()
	var emitted []spec.DomainEvent
	p.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
	ctx := testing_tenant.Default(context.Background())
	lastAggregated := false
	for i := range times {
		agg, err := p.recordLoginFailure(ctx, username, "")
		if err != nil {
			t.Fatalf("call #%d: %v", i, err)
		}
		lastAggregated = agg
	}
	return lastAggregated, emitted
}

func TestRecordLoginFailureAggregatesWhenLocked(t *testing.T) {
	p := PasswordAuthentication{
		LoginAttemptThrottle: fakeLockedThrottle{},
		AuthEventBucketStore: authnmemory.NewAuthEventBucketStore(),
	}
	aggregated, emitted := driveRecordLoginFailure(t, p, "alice", 3)
	if !aggregated {
		t.Fatalf("expected aggregated=true when locked with bucket store")
	}
	// 同一窓なので集約イベントは 1 件だけ。LoginThrottled は毎回出る。
	if got := countEventType(emitted, "AuthenticationEventAggregated"); got != 1 {
		t.Fatalf("expected 1 AuthenticationEventAggregated, got %d", got)
	}
	if got := countEventType(emitted, "LoginThrottled"); got != 3 {
		t.Fatalf("expected 3 LoginThrottled, got %d", got)
	}
	// 集約に切り替わったので個別の AuthenticationFailed は出さない (呼び出し側で抑制)。
	if got := countEventType(emitted, "AuthenticationFailed"); got != 0 {
		t.Fatalf("expected 0 AuthenticationFailed, got %d", got)
	}
}

func TestRecordLoginFailureWithoutBucketStoreDoesNotAggregate(t *testing.T) {
	p := PasswordAuthentication{LoginAttemptThrottle: fakeLockedThrottle{}}
	aggregated, emitted := driveRecordLoginFailure(t, p, "alice", 2)
	if aggregated {
		t.Fatalf("expected aggregated=false without bucket store")
	}
	if got := countEventType(emitted, "AuthenticationEventAggregated"); got != 0 {
		t.Fatalf("expected 0 AuthenticationEventAggregated, got %d", got)
	}
}

type spyThrottleMetrics struct {
	throttlePolicy  string
	throttleOutcome string
}

func (s *spyThrottleMetrics) RecordLoginThrottle(policy, outcome string) {
	s.throttlePolicy, s.throttleOutcome = policy, outcome
}

type stubLoginThrottle struct {
	result sessionports.LoginThrottleResult
	err    error
}

func (s stubLoginThrottle) TryAcquire(
	context.Context, sessionports.LoginThrottleKind, string, time.Time,
) (sessionports.LoginThrottleResult, error) {
	return s.result, s.err
}

func (stubLoginThrottle) RecordFailure(
	context.Context, sessionports.LoginThrottleKind, string, time.Time,
) (sessionports.LoginThrottleResult, error) {
	return sessionports.LoginThrottleResult{}, nil
}

func (stubLoginThrottle) RecordSuccess(context.Context, sessionports.LoginThrottleKind, string) error {
	return nil
}

type stubTenantSaltStore struct {
	salt []byte
	err  error
}

func (s stubTenantSaltStore) GetSalt(context.Context) ([]byte, error) {
	return s.salt, s.err
}

func TestAcquireLoginThrottleRecordsMetricOutcome(t *testing.T) {
	cases := []struct {
		name              string
		throttle          stubLoginThrottle
		wantAllowed       bool
		wantMetricOutcome string
	}{
		{
			name:              "allowed",
			throttle:          stubLoginThrottle{result: sessionports.LoginThrottleResult{Allowed: true}},
			wantAllowed:       true,
			wantMetricOutcome: "allowed",
		},
		{
			name:              "throttled",
			throttle:          stubLoginThrottle{result: sessionports.LoginThrottleResult{Allowed: false, RetryAfterSeconds: 30}},
			wantAllowed:       false,
			wantMetricOutcome: "throttled",
		},
		{
			name:              "store unavailable",
			throttle:          stubLoginThrottle{err: context.DeadlineExceeded},
			wantAllowed:       false,
			wantMetricOutcome: "store_unavailable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &spyThrottleMetrics{}
			p := PasswordAuthentication{LoginAttemptThrottle: tc.throttle, Metrics: spy}
			result, _ := p.acquireLoginThrottle(context.Background(), sessionports.LoginThrottleAccount, "alice")
			if result.Allowed != tc.wantAllowed {
				t.Fatalf("Allowed=%v, want %v", result.Allowed, tc.wantAllowed)
			}
			if spy.throttlePolicy != string(sessionports.LoginThrottleAccount) {
				t.Fatalf("policy=%q, want %q", spy.throttlePolicy, sessionports.LoginThrottleAccount)
			}
			if spy.throttleOutcome != tc.wantMetricOutcome {
				t.Fatalf("outcome=%q, want %q", spy.throttleOutcome, tc.wantMetricOutcome)
			}
		})
	}
}

func TestAcquireLoginThrottleWithoutStoreAllowsAndSkipsMetric(t *testing.T) {
	spy := &spyThrottleMetrics{}
	p := PasswordAuthentication{Metrics: spy}
	result, err := p.acquireLoginThrottle(context.Background(), sessionports.LoginThrottleIP, "203.0.113.10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected Allowed=true when no throttle store is configured")
	}
	if spy.throttleOutcome != "" {
		t.Fatalf("expected no metric recorded without a throttle store, got outcome=%q", spy.throttleOutcome)
	}
}

func TestCorrelationHashUsesTenantSaltWhenAvailable(t *testing.T) {
	p := PasswordAuthentication{TenantSaltStore: stubTenantSaltStore{salt: []byte("tenant-salt")}}
	salted := p.correlationHash(context.Background(), "alice")
	if salted == "" || salted == hashThrottleKey("alice") {
		t.Fatalf("expected salted hash distinct from the unsalted fallback, got %q", salted)
	}
}

func TestCorrelationHashFallsBackToUnsaltedHashWithoutSaltStore(t *testing.T) {
	p := PasswordAuthentication{}
	got := p.correlationHash(context.Background(), "alice")
	if want := hashThrottleKey("alice"); got != want {
		t.Fatalf("correlationHash=%q, want fallback %q", got, want)
	}
}

func TestCorrelationHashFallsBackWhenSaltStoreErrors(t *testing.T) {
	p := PasswordAuthentication{TenantSaltStore: stubTenantSaltStore{err: context.DeadlineExceeded}}
	got := p.correlationHash(context.Background(), "alice")
	if want := hashThrottleKey("alice"); got != want {
		t.Fatalf("correlationHash=%q, want fallback %q", got, want)
	}
}

// throttledKinds は捕捉した LoginThrottled の種類を発行順に返す。
func throttledKinds(events []spec.DomainEvent) []string {
	var kinds []string
	for _, e := range events {
		if throttled, ok := e.(*authdomain.LoginThrottled); ok {
			kinds = append(kinds, throttled.Kind)
		}
	}
	return kinds
}

// 失敗はアカウントと発信元の IP の両方に記録し、IP が分からないときはアカウントだけに記録する。
// 鍵の種類まで読まないと、どちらか片方の鍵を取り違えた記録を見分けられない。
func TestRecordLoginFailureRecordsBothKeysAndSkipsAnUnknownIP(t *testing.T) {
	for _, tc := range []struct {
		name     string
		clientIP string
		want     []string
	}{
		{name: "with IP", clientIP: "203.0.113.9", want: []string{string(sessionports.LoginThrottleAccount), string(sessionports.LoginThrottleIP)}},
		{name: "without IP", clientIP: "", want: []string{string(sessionports.LoginThrottleAccount)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var emitted []spec.DomainEvent
			p := PasswordAuthentication{LoginAttemptThrottle: fakeLockedThrottle{}, Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) }}
			if _, err := p.recordLoginFailure(testing_tenant.Default(context.Background()), "alice", tc.clientIP); err != nil {
				t.Fatal(err)
			}
			if got := throttledKinds(emitted); !slices.Equal(got, tc.want) {
				t.Fatalf("LoginThrottled kinds = %v, want %v", got, tc.want)
			}
		})
	}
}

// ipOnlyThrottle は IP 単位の取得だけを拒否する。
type ipOnlyThrottle struct{ stubLoginThrottle }

func (ipOnlyThrottle) TryAcquire(
	_ context.Context, kind sessionports.LoginThrottleKind, _ string, _ time.Time,
) (sessionports.LoginThrottleResult, error) {
	if kind == sessionports.LoginThrottleIP {
		return sessionports.LoginThrottleResult{Allowed: false, RetryAfterSeconds: 60}, nil
	}
	return sessionports.LoginThrottleResult{Allowed: true}, nil
}

// IP の制限は発信元が分かるときだけ判定する。分からない試みまで IP の鍵で止めると、空の鍵を
// 共有するすべての試みが一つの発信元として締め出される。
func TestAuthenticatePasswordChecksTheIPThrottleOnlyForAKnownSource(t *testing.T) {
	p := PasswordAuthentication{
		LoginAttemptThrottle: ipOnlyThrottle{}, UserRepo: idmanagement.NewMemoryModule().UserRepo,
		PasswordHasher: testing_passwords.NewHasher(),
	}
	ctx := testing_tenant.Default(context.Background())
	outcome, err := p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: "x", ClientIP: "203.0.113.9"})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Refusal != signinports.PasswordRefusalThrottled || outcome.RetryAfterSeconds != 60 {
		t.Fatalf("outcome=%+v, want throttled with Retry-After 60", outcome)
	}

	t.Run("control: an unknown source reaches the password check", func(t *testing.T) {
		outcome, err := p.AuthenticatePassword(ctx, signinports.PasswordAttempt{Username: "alice", Password: "x"})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Refusal != signinports.PasswordRefusalInvalidCredentials {
			t.Fatalf("outcome=%+v, want invalid_credentials", outcome)
		}
	})
}
