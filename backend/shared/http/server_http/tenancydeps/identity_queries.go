package tenancydeps

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func GroupAttributeReferences(groups groupports.GroupRepository) tenantports.DynamicGroupAttributeReferences {
	if groups == nil {
		return nil
	}
	return func(ctx context.Context, tenantID string) ([][]string, error) {
		rules, err := groups.ListDynamicRules(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		references := make([][]string, len(rules))
		for i, rule := range rules {
			references[i] = rule.ReferencedAttributes
		}
		return references, nil
	}
}

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
