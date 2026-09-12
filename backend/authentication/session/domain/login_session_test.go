package domain_test

import (
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func mustUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	return id
}

func TestLoginSessionAndRequestValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validSession := domain.LoginSession{ID: mustUUID(t), UserID: "user_1", AMR: []string{"pwd"}, ACR: "1", ExpiresAt: now}
	badSession := validSession
	badSession.AMR = nil

	validLoginReq := domain.LoginRequest{RequestID: mustUUID(t), Username: "alice", Password: "pw"}
	badLoginReq := domain.LoginRequest{RequestID: "not-a-uuid", Username: "alice", Password: "pw"}

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"session ok", validSession, false},
		{"session bad", badSession, true},
		{"login req ok", validLoginReq, false},
		{"login req bad", badLoginReq, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}

// TestLoginSessionActive: scenario `ユーザーは自身の有効なセッションを管理する`
// (wi-253) — 認証解決は revoked / 期限切れの LoginSession を fail-closed で除外する。
func TestLoginSessionActive(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)

	tests := []struct {
		name string
		sess domain.LoginSession
		want bool
	}{
		{"active", domain.LoginSession{ExpiresAt: now.Add(time.Hour)}, true},
		{"expired", domain.LoginSession{ExpiresAt: now.Add(-time.Second)}, false},
		{"revoked", domain.LoginSession{ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}, false},
		{"revoked and expired", domain.LoginSession{ExpiresAt: now.Add(-time.Second), RevokedAt: &revokedAt}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sess.Active(now); got != tt.want {
				t.Fatalf("Active() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoginSessionRevokeIdempotent: 2 回目以降の失効要求は最初の revoked_at /
// revoke_reason を保持したまま成功として扱う (tombstone)。
func TestLoginSessionRevokeIdempotent(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	sess := domain.LoginSession{ExpiresAt: now.Add(time.Hour)}

	sess.Revoke(spec.SessionEndSelfRevoke, now)
	if sess.RevokedAt == nil || !sess.RevokedAt.Equal(now) {
		t.Fatalf("first revoke: RevokedAt = %v, want %v", sess.RevokedAt, now)
	}
	if sess.RevokeReason == nil || *sess.RevokeReason != spec.SessionEndSelfRevoke {
		t.Fatalf("first revoke: RevokeReason = %v, want %v", sess.RevokeReason, spec.SessionEndSelfRevoke)
	}
	if sess.Active(now) {
		t.Fatalf("revoked session must not be active")
	}

	later := now.Add(time.Minute)
	sess.Revoke(spec.SessionEndAdminRevoke, later)
	if !sess.RevokedAt.Equal(now) {
		t.Fatalf("second revoke overwrote RevokedAt: got %v, want %v (first revocation wins)", sess.RevokedAt, now)
	}
	if *sess.RevokeReason != spec.SessionEndSelfRevoke {
		t.Fatalf("second revoke overwrote RevokeReason: got %v, want %v", *sess.RevokeReason, spec.SessionEndSelfRevoke)
	}
}

// TestLoginSessionTouch: last_seen_at は LoginSessionTouchInterval 未満の再 touch では
// 更新しない粗粒度な値 (idle timeout 判定はこの粒度で近似する、wi-253)。
func TestLoginSessionTouch(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	sess := domain.LoginSession{ExpiresAt: now.Add(time.Hour)}

	if !sess.Touch(now) {
		t.Fatalf("first touch must update last_seen_at")
	}
	if !sess.LastSeenAt.Equal(now) {
		t.Fatalf("LastSeenAt = %v, want %v", sess.LastSeenAt, now)
	}

	soon := now.Add(domain.LoginSessionTouchInterval - time.Second)
	if sess.Touch(soon) {
		t.Fatalf("touch within interval must not update last_seen_at")
	}
	if !sess.LastSeenAt.Equal(now) {
		t.Fatalf("LastSeenAt changed within interval: got %v, want %v", sess.LastSeenAt, now)
	}

	later := now.Add(domain.LoginSessionTouchInterval + time.Second)
	if !sess.Touch(later) {
		t.Fatalf("touch past interval must update last_seen_at")
	}
	if !sess.LastSeenAt.Equal(later) {
		t.Fatalf("LastSeenAt = %v, want %v", sess.LastSeenAt, later)
	}
}

// 宣言された 8 語はいずれも通り、語彙の外の値はひとつでもあれば通らない。error の本文が
// 語彙の外の値を名指すことも観測する。何が拒否されたのかを言わない検査は、認証要素を
// 足した人にとって「amr が不正」以上の手がかりを持たない。
//
//spec:covers RFC8176-AMR-VOCABULARY: LoginSession の検証が amr の語彙を閉じていることを固定する。
func TestLoginSessionClosesTheAMRVocabulary(t *testing.T) {
	now := time.Now().UTC()
	session := func(amr []string) domain.LoginSession {
		return domain.LoginSession{ID: mustUUID(t), UserID: "user_1", AMR: amr, ACR: "1", ExpiresAt: now}
	}

	// 宣言された語彙は 1 語ずつ、そして全部まとめても通る。
	for _, value := range authdomain.AMRVocabulary() {
		if err := session([]string{value}).Validate(); err != nil {
			t.Fatalf("declared amr %q rejected: %v", value, err)
		}
	}
	if err := session(authdomain.AMRVocabulary()).Validate(); err != nil {
		t.Fatalf("the whole vocabulary rejected: %v", err)
	}

	// 語彙の外の値は、RFC 8176 の登録値であっても通らない。
	for _, unknown := range []string{"mfa", "pop", "sms", "user", "FEDERATED", "pwd "} {
		err := session([]string{"pwd", unknown}).Validate()
		if err == nil {
			t.Fatalf("amr %q outside the vocabulary was accepted", unknown)
		}
		if !strings.Contains(err.Error(), unknown) {
			t.Fatalf("error %q does not name the rejected value %q", err, unknown)
		}
	}
}
