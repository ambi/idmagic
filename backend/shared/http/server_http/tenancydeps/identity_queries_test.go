package tenancydeps_test

import (
	"context"
	"errors"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/http/server_http/tenancydeps"
)

func TestNotificationActorsExposeOnlyTheVerifiedAddress(t *testing.T) {
	for _, tc := range []struct {
		name     string
		email    *string
		verified bool
		want     string
	}{
		{name: "verified", email: new("operator@example.test"), verified: true, want: "operator@example.test"},
		{name: "unverified", email: new("operator@example.test")},
		{name: "missing", verified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := tenancydeps.TestNotificationActors(userLookup{user: &userdomain.User{
				ID: "operator", Name: new("Operator"), Email: tc.email, EmailVerified: tc.verified,
			}})
			actor, err := reader(context.Background(), "operator")
			if err != nil || actor == nil || actor.Email != tc.want || actor.DisplayName != "Operator" {
				t.Fatalf("unexpected operator recipient: %#v, error %v", actor, err)
			}
		})
	}
}

func TestIdentityQueriesPreserveMissingUsersAndReadErrors(t *testing.T) {
	ctx := context.Background()
	missing, err := tenancydeps.TestNotificationActors(userLookup{})(ctx, "missing")
	if err != nil || missing != nil {
		t.Fatalf("missing operator became a recipient: %#v, %v", missing, err)
	}
	want := errors.New("read unavailable")
	actor, err := tenancydeps.TestNotificationActors(userLookup{err: want})(ctx, "operator")
	if !errors.Is(err, want) || actor != nil {
		t.Fatalf("operator read error changed: %#v, %v", actor, err)
	}
}

type userLookup struct {
	userports.UserRepository
	user *userdomain.User
	err  error
}

func (s userLookup) FindBySub(_ context.Context, sub string) (*userdomain.User, error) {
	if sub != "operator" && sub != "missing" {
		return nil, errors.New("wrong operator")
	}
	return s.user, s.err
}
