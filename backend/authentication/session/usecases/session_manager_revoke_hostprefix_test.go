package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/session/db_memory"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/authentication/session/usecases"
)

// TestRevokeHonorsHostPrefixedCookie は REQ-AUTHENTICATION-035 を、返り値ではなく効果で検査する。
//
// サブドメイン形式のテナントでは support_http.TenantCookieName がセッション Cookie を
// "__Host-idmagic_session" として発行する。Revoke がこの接頭辞を見落とすと、サインアウトは
// 成功を返したままサーバー側のセッションが生き続け、Cookie を復元すれば再び認証できてしまう。
func TestRevokeHonorsHostPrefixedCookie(t *testing.T) {
	for name, cookieName := range map[string]string{
		"subdomain tenant": "__Host-" + usecases.SessionCookie,
		"path tenant":      usecases.SessionCookie,
	} {
		t.Run(name, func(t *testing.T) {
			store := db_memory.NewSessionStore()
			ctx := context.Background()
			now := time.Now().UTC()

			session := &domain.LoginSession{
				ID:        "sid-under-test",
				TenantID:  "tenant-a",
				UserID:    "user-1",
				AuthTime:  now.Unix(),
				ExpiresAt: now.Add(time.Hour),
			}
			if err := store.Save(ctx, session); err != nil {
				t.Fatalf("save session: %v", err)
			}

			manager := usecases.NewSessionManager(store)
			if err := manager.Revoke(ctx, cookieName+"="+session.ID); err != nil {
				t.Fatalf("revoke: %v", err)
			}

			// 失効が起きたことは、応答ではなく状態を読み戻して確かめる。
			found, err := store.Find(ctx, session.ID)
			if err != nil {
				t.Fatalf("find after revoke: %v", err)
			}
			if found != nil {
				t.Fatalf("sign-out left the session active: %+v", found)
			}
		})
	}
}

// TestSessionEntryPointsResolveTheSameCookie は、Cookie 名の選択が 1 か所に集約されていることを
// 呼び出し側から検査する。同じヘッダに対して 3 つの入口が同じセッションを指さなければならない。
func TestSessionEntryPointsResolveTheSameCookie(t *testing.T) {
	headers := map[string]string{
		"host prefixed only": "__Host-" + usecases.SessionCookie + "=sid-1",
		"plain only":         usecases.SessionCookie + "=sid-1",
		"both present":       "__Host-" + usecases.SessionCookie + "=sid-1; " + usecases.SessionCookie + "=sid-2",
		"with noise":         "other=x; __Host-" + usecases.SessionCookie + "=sid-1; foo=bar",
	}
	for name, header := range headers {
		t.Run(name, func(t *testing.T) {
			store := db_memory.NewSessionStore()
			ctx := context.Background()
			now := time.Now().UTC()

			manager := usecases.NewSessionManager(store)
			resolved := manager.SessionIDFromCookie(header)
			if resolved != "sid-1" {
				t.Fatalf("SessionIDFromCookie=%q, want sid-1", resolved)
			}

			session := &domain.LoginSession{
				ID: resolved, TenantID: "tenant-a", UserID: "user-1",
				AuthTime: now.Unix(), ExpiresAt: now.Add(time.Hour),
			}
			if err := store.Save(ctx, session); err != nil {
				t.Fatalf("save session: %v", err)
			}
			if err := manager.Revoke(ctx, header); err != nil {
				t.Fatalf("revoke: %v", err)
			}
			found, err := store.Find(ctx, resolved)
			if err != nil {
				t.Fatalf("find after revoke: %v", err)
			}
			if found != nil {
				t.Fatalf("Revoke resolved a different cookie than SessionIDFromCookie: %+v", found)
			}
		})
	}
}
