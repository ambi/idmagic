package server_http_test

// Authentication が User の一部の値だけを記録するとき、その記録の直前に管理者が行った無効化を
// 書き戻さないことを、正式な組み立てを通して観測する。競合の窓は、保存先を包んで、
// Authentication の次の書き込みの直前に無効化を差し込むことで決定的に作る。

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
)

// disableBeforeNextWrite は、armed の間の最初の書き込みの直前に、別の書き手として User を
// 無効化する。無効化は包む前の保存先へ直接書くので、書き込みを行う側からは見えない。
type disableBeforeNextWrite struct {
	userports.UserRepository
	armed atomic.Bool
}

func (u *disableBeforeNextWrite) disableFirst(ctx context.Context, sub string) {
	if !u.armed.CompareAndSwap(true, false) {
		return
	}
	user, err := u.FindBySub(ctx, sub)
	if err != nil || user == nil {
		panic("concurrent disable: user not found")
	}
	user.Lifecycle.Status = idmdomain.UserStatusDisabled
	if err := u.UserRepository.Save(ctx, user); err != nil {
		panic(err)
	}
}

func (u *disableBeforeNextWrite) Save(ctx context.Context, user *userdomain.User) error {
	u.disableFirst(ctx, user.ID)
	return u.UserRepository.Save(ctx, user)
}

func (u *disableBeforeNextWrite) Update(ctx context.Context, sub string, change userports.UserChange) (*userdomain.User, error) {
	u.disableFirst(ctx, sub)
	return u.UserRepository.Update(ctx, sub, change)
}

func loadAlice(t *testing.T, users userports.UserRepository) *userdomain.User {
	t.Helper()
	stored, err := users.FindBySub(t.Context(), "user_alice")
	if err != nil || stored == nil {
		t.Fatalf("load user: %+v %v", stored, err)
	}
	return stored
}

//spec:covers REQ-IDMANAGEMENT-010: ログインが最終ログインを記録する直前に管理者が User を無効化しても、ログインの記録が無効化を書き戻さず、User が Disabled のまま残ること。
func TestLoginKeepsADisableMadeWhileItRecordsTheLogin(t *testing.T) {
	race := &disableBeforeNextWrite{}
	srv, users, _ := newPasswordExpiryServer(t, passwordExpiryOptions{
		wrapUsers: func(inner userports.UserRepository) userports.UserRepository {
			race.UserRepository = inner
			return race
		},
	})
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	race.armed.Store(true)
	postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": expiryTestUsername, "password": expiryTestPassword, "return_to": returnTo,
	})

	if race.armed.Load() {
		t.Fatal("前提が壊れている: ログインが User を書かなかった")
	}
	stored := loadAlice(t, users)
	if stored.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("status=%q, want disabled: the login reverted the concurrent disable", stored.Lifecycle.Status)
	}
	if stored.Lifecycle.LastLoginAt == nil {
		t.Fatal("last_login_at was not recorded")
	}
}

//spec:covers REQ-AUTHENTICATION-024: パスワードの変更が新しいハッシュを保存する直前に管理者が User を無効化しても、変更が無効化を書き戻さず、新しいパスワードと update_password の解除だけが残ること。
func TestPasswordChangeKeepsADisableMadeWhileItSavesTheHash(t *testing.T) {
	race := &disableBeforeNextWrite{}
	srv, users, _ := newPasswordExpiryServer(t, passwordExpiryOptions{
		maxAgeDays: 90, passwordChangedDaysAgo: 91,
		wrapUsers: func(inner userports.UserRepository) userports.UserRepository {
			race.UserRepository = inner
			return race
		},
	})
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": expiryTestUsername, "password": expiryTestPassword, "return_to": returnTo,
	})

	const replacement = "fresh-race-password-5524"
	account := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/account")
	race.armed.Store(true)
	changed := postAuthJSON(t, client, srv.URL+"/realms/default/api/auth/change-password",
		account.CSRFToken, map[string]string{"current_password": expiryTestPassword, "new_password": replacement})
	changedStatus := changed.StatusCode
	changed.Body.Close()
	if changedStatus != http.StatusNoContent {
		t.Fatalf("change_password status=%d, want 204", changedStatus)
	}

	if race.armed.Load() {
		t.Fatal("前提が壊れている: パスワードの変更が User を書かなかった")
	}
	stored := loadAlice(t, users)
	if stored.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("status=%q, want disabled: the password change reverted the concurrent disable", stored.Lifecycle.Status)
	}
	if ok, err := testing_passwords.NewHasher().Verify(replacement, stored.PasswordHash); err != nil || !ok {
		t.Fatalf("the new password was not stored: ok=%v err=%v", ok, err)
	}
	if len(stored.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("required actions=%v, want update_password cleared", stored.Lifecycle.RequiredActions)
	}
}
