package valkey

import (
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
)

func newThrottle(t *testing.T) *LoginAttemptThrottle {
	t.Helper()
	return &LoginAttemptThrottle{
		Client: testClient(t),
		Configs: authnports.LoginThrottleConfigs{
			Account: authnports.LoginThrottleConfig{MaxFailures: 3, WindowSeconds: 60, LockoutSeconds: 120},
			IP:      authnports.LoginThrottleConfig{MaxFailures: 5, WindowSeconds: 60, LockoutSeconds: 60},
		},
	}
}

func TestValkeyLoginThrottleLocksAtThreshold(t *testing.T) {
	throttle := newThrottle(t)
	now := time.Now().UTC()
	for i := range 2 {
		result, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now)
		if err != nil || result.Locked {
			t.Fatalf("failure %d: result=%+v err=%v", i+1, result, err)
		}
	}
	result, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now)
	if err != nil || !result.Locked || result.RetryAfterSeconds != 120 {
		t.Fatalf("lock result=%+v err=%v", result, err)
	}
	acquire, err := throttle.TryAcquire(t.Context(), authnports.LoginThrottleAccount, "alice", now)
	if err != nil || acquire.Allowed || acquire.RetryAfterSeconds <= 0 || acquire.RetryAfterSeconds > 120 {
		t.Fatalf("acquire result=%+v err=%v", acquire, err)
	}
}

// throttle が cluster-wide であること: 同一識別子への失敗を別々のクライアント接続
// (別レプリカに相当) から積んでも、カウンタは共有ストア上で合算されて閾値でロックされる。
func TestValkeyLoginThrottleCountsAcrossReplicas(t *testing.T) {
	client := testClient(t)
	configs := authnports.LoginThrottleConfigs{
		Account: authnports.LoginThrottleConfig{MaxFailures: 3, WindowSeconds: 60, LockoutSeconds: 120},
	}
	replicaA := &LoginAttemptThrottle{Client: client, Configs: configs}
	replicaB := &LoginAttemptThrottle{Client: client, Configs: configs}
	now := time.Now().UTC()

	if r, err := replicaA.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now); err != nil || r.Locked {
		t.Fatalf("replicaA first: result=%+v err=%v", r, err)
	}
	if r, err := replicaB.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now); err != nil || r.Locked {
		t.Fatalf("replicaB second: result=%+v err=%v", r, err)
	}
	// 3 回目の失敗は別レプリカから来ても合算閾値でロックされる (per-replica なら緩む箇所)。
	r, err := replicaA.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now)
	if err != nil || !r.Locked {
		t.Fatalf("replicaA third should lock: result=%+v err=%v", r, err)
	}
	// どのレプリカから見てもロックが見える。
	if acquire, err := replicaB.TryAcquire(t.Context(), authnports.LoginThrottleAccount, "alice", now); err != nil || acquire.Allowed {
		t.Fatalf("replicaB should see lock: result=%+v err=%v", acquire, err)
	}
}

func TestValkeyLoginThrottleSuccessClearsAccountOnly(t *testing.T) {
	throttle := newThrottle(t)
	now := time.Now().UTC()
	if _, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now); err != nil {
		t.Fatal(err)
	}
	if _, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleIP, "203.0.113.1", now); err != nil {
		t.Fatal(err)
	}
	if err := throttle.RecordSuccess(t.Context(), authnports.LoginThrottleAccount, "alice"); err != nil {
		t.Fatal(err)
	}
	// account counter は消える。
	if r, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now); err != nil || r.Locked {
		t.Fatalf("account counter should have reset: result=%+v err=%v", r, err)
	}
	// per-IP counter は success で消えない (ADR-029): 直後の失敗でカウントが継続する。
	for i := range 3 {
		if _, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleIP, "203.0.113.1", now); err != nil {
			t.Fatalf("ip failure %d: %v", i, err)
		}
	}
	// IP は max 5、success 前 1 + ここで 3 = 4、まだ未ロック。もう 1 回でロック。
	r, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleIP, "203.0.113.1", now)
	if err != nil || !r.Locked {
		t.Fatalf("ip should lock without clearing on account success: result=%+v err=%v", r, err)
	}
}

// 到達不能な共有ストアではエラーを返し、呼び出し側で fail-closed に倒れる (ADR-077)。
func TestValkeyLoginThrottleFailsClosedOnStoreError(t *testing.T) {
	throttle := newThrottle(t)
	throttle.Client.Close()
	now := time.Now().UTC()
	if _, err := throttle.TryAcquire(t.Context(), authnports.LoginThrottleAccount, "alice", now); err == nil {
		t.Fatal("TryAcquire should error when store is unreachable")
	}
	if _, err := throttle.RecordFailure(t.Context(), authnports.LoginThrottleAccount, "alice", now); err == nil {
		t.Fatal("RecordFailure should error when store is unreachable")
	}
}
