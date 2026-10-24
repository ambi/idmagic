package tenancydeps

import (
	"context"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestNotificationActors(users userports.UserRepository) tenantports.TestNotificationActorReader {
	if users == nil {
		return nil
	}
	return func(ctx context.Context, sub string) (*tenantports.TestNotificationActor, error) {
		user, err := users.FindBySub(ctx, sub)
		if err != nil || user == nil {
			return nil, err
		}
		actor := &tenantports.TestNotificationActor{DisplayName: user.DisplayName()}
		if user.Email != nil && user.EmailVerified {
			actor.Email = *user.Email
		}
		return actor, nil
	}
}
