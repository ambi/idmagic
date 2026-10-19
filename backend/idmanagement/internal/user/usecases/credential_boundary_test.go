package usecases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type initialPasswordRecordingHasher struct{ calls *[]string }

func (h initialPasswordRecordingHasher) Hash(string) (string, error) {
	*h.calls = append(*h.calls, "hash")
	return "encoded", nil
}

func TestCreateUserValidatesPasswordBeforeRolesAndHashesAfterRoles(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	for _, invalidPassword := range []bool{true, false} {
		t.Run(map[bool]string{true: "password", false: "role"}[invalidPassword], func(t *testing.T) {
			f := newUserRulesFixture(t)
			var calls []string
			policyError := errors.New("rejected password")
			f.deps.ValidateInitialPassword = func(context.Context, string) error {
				calls = append(calls, "validate")
				if invalidPassword {
					return policyError
				}
				return nil
			}
			f.deps.PasswordHasher = initialPasswordRecordingHasher{calls: &calls}
			user, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
				PreferredUsername: "alice", Password: "initial-password-9182", Roles: []string{""},
			})
			if user != nil || err == nil || invalidPassword && !errors.Is(err, policyError) {
				t.Fatalf("user=%+v error=%v, want refusal before hashing", user, err)
			}
			if !reflect.DeepEqual(calls, []string{"validate"}) || len(*f.events) != 0 {
				t.Fatalf("calls=%v events=%v, want validation only", calls, *f.events)
			}
			stored, findErr := f.users.FindByUsername(ctx, tenantports.TenantID(ctx), "alice")
			if findErr != nil || stored != nil {
				t.Fatalf("refused user=%+v error=%v, want no persisted user", stored, findErr)
			}
		})
	}
}

func TestDeleteUserRejectsMissingCredentialPurgerBeforeAnonymizing(t *testing.T) {
	f := newUserRulesFixture(t)
	user := f.seed("missing-purger", nil)
	f.deps.UserCredentials = nil
	err := userusecases.DeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.DeleteUserInput{
		ActorUserID: "operator", Sub: user.ID, Now: time.Now().UTC(),
	})
	stored, findErr := f.users.FindBySubIncludingDeleted(context.Background(), user.ID)
	if err == nil || findErr != nil || !reflect.DeepEqual(stored, user) || len(*f.events) != 0 {
		t.Fatalf("error=%v stored=%+v findError=%v events=%v, want missing dependency without mutation", err, stored, findErr, *f.events)
	}
}

type failingUserCredentials struct {
	err   error
	calls []string
}

func (f *failingUserCredentials) PurgeUserCredentials(_ context.Context, sub string) error {
	f.calls = append(f.calls, sub)
	return f.err
}

//spec:covers REQ-IDMANAGEMENT-050: 資格情報の削除に失敗した完全削除は使用量とイベントへ進まず、同じ Tombstone から再実行できる。
func TestDeleteUserPropagatesCredentialFailureAndResumes(t *testing.T) {
	f := newUserRulesFixture(t)
	user := f.seed("credential-failure", nil)
	ctx := testing_tenant.Default(context.Background())
	if err := f.quota.CheckAndIncrement(ctx, tenancydomain.DefaultTenantID, tenancydomain.ResourceUsers, 1); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("credential storage failed")
	credentials := &failingUserCredentials{err: failure}
	f.deps.UserCredentials = credentials
	input := userusecases.DeleteUserInput{ActorUserID: "operator", Sub: user.ID, Reason: "first request", Now: userRulesNow}
	if err := userusecases.DeleteUser(ctx, f.deps, input); !errors.Is(err, failure) {
		t.Fatalf("error=%v, want credential failure", err)
	}
	stored := f.stored(t, user.ID)
	usage, err := f.quota.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil || usage.Users != 1 || !stored.IsDeleted() || stored.Lifecycle.PendingPurge == nil || len(*f.events) != 0 {
		t.Fatalf("stored=%+v usage=%+v error=%v events=%v, want pending purge before quota and event", stored, usage, err, *f.events)
	}
	credentials.err = nil
	if err := userusecases.DeleteUser(ctx, f.deps, input); err != nil {
		t.Fatal(err)
	}
	stored = f.stored(t, user.ID)
	usage, err = f.quota.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil || usage.Users != 0 || stored.Lifecycle.PendingPurge != nil || len(*f.events) != 1 || !reflect.DeepEqual(credentials.calls, []string{user.ID, user.ID}) {
		t.Fatalf("stored=%+v usage=%+v error=%v events=%v calls=%v, want completed retry", stored, usage, err, *f.events, credentials.calls)
	}
}
