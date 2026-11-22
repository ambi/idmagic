package usecases

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// seedDemoData は検証済みのマニフェストからデモを作り、既存値のドリフトは拒否する。
func SeedDemoData(
	ctx context.Context,
	clients oauthports.OAuth2ClientRepository,
	users userports.UserRepository,
	mfaFactors totpports.MfaFactorRepository,
	passwordHistory passwordports.PasswordHistoryRepository,
	groups groupports.GroupRepository,
	authzDetailTypes oauthports.AuthorizationDetailTypeRepository,
	hasher passwordports.PasswordHasher,
	seed seedingdomain.DevelopmentDemoSeed,
	clientSecret string,
	password string,
	totpSecret string,
	operations *resourceOperations,
	commands ResourceCommands,
) error {
	secretHash := oauthdomain.HashClientSecret(clientSecret)
	now := time.Now().UTC()
	demoClientName := "Demo Client"
	demoClient := &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: seed.ClientID, ClientName: &demoClientName,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: seed.ClientRedirectURIs,
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantClientCredentials, spec.GrantDeviceCode, spec.GrantCiba,
		},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		Scope:                   "openid profile email offline_access", IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now, UpdatedAt: now,
	}
	currentClient, err := clients.FindByID(ctx, demoClient.TenantID, demoClient.ClientID)
	if err != nil {
		return err
	}
	if err := operations.ensure("oauth2-client:"+seed.ClientID, currentClient != nil, currentClient != nil && sameDemoClient(currentClient, demoClient, clientSecret), func() error {
		return commands.CreateClient(ctx, demoClient)
	}); err != nil {
		return err
	}
	if result := passworddomain.ValidatePassword(password); !result.OK {
		return errors.New("seed user password violates password policy")
	}
	for index, configured := range seed.Users {
		email := configured.Email
		user := &userdomain.User{
			ID: configured.ID, TenantID: tenancydomain.DefaultTenantID,
			PreferredUsername: configured.PreferredUsername,
			Email:             &email, EmailVerified: true, MfaEnrolled: index == 0 && totpSecret != "",
			Roles: configured.Roles, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := ensureDemoUser(ctx, users, passwordHistory, hasher, user, password, now, operations, commands); err != nil {
			return err
		}
	}
	if err := seedDemoGroups(ctx, groups, seed.Groups, now, operations, commands); err != nil {
		return err
	}
	if err := seedDemoAuthorizationDetailTypes(ctx, authzDetailTypes, seed.AuthorizationDetailType, now, operations, commands); err != nil {
		return err
	}
	if totpSecret == "" {
		return nil
	}
	label := "Demo TOTP"
	desiredFactor := &totpdomain.MfaFactor{
		UserID: seed.Users[0].ID, Type: spec.MfaFactorTOTP, Secret: &totpSecret, Label: &label, CreatedAt: now,
	}
	existingFactor, err := mfaFactors.Find(ctx, seed.Users[0].ID, spec.MfaFactorTOTP)
	if err != nil {
		return err
	}
	return operations.ensure("mfa-factor:"+seed.Users[0].ID+":totp", existingFactor != nil, existingFactor != nil && sameMfaFactor(existingFactor, desiredFactor), func() error {
		return commands.CreateTOTP(ctx, desiredFactor)
	})
}

func sameDemoClient(actual, desired *oauthdomain.OAuth2Client, secret string) bool {
	if actual.ClientSecretHash == nil || !oauthdomain.VerifyClientSecret(secret, *actual.ClientSecretHash) {
		return false
	}
	left, right := *actual, *desired
	left.ClientSecretHash, right.ClientSecretHash = nil, nil
	return seedingdomain.SameFirstPartyClient(&left, &right)
}

func ensureDemoUser(ctx context.Context, users userports.UserRepository, history passwordports.PasswordHistoryRepository, hasher passwordports.PasswordHasher, desired *userdomain.User, password string, now time.Time, operations *resourceOperations, commands ResourceCommands) error {
	existing, err := users.FindBySub(ctx, desired.ID)
	if err != nil {
		return err
	}
	if err := operations.ensure("user:"+desired.ID, existing != nil, existing != nil && sameDemoUser(existing, desired, password, hasher), func() error {
		return commands.CreateUser(ctx, desired, password)
	}); err != nil {
		return err
	}
	recent, err := history.Recent(ctx, desired.ID, 24)
	if err != nil {
		return err
	}
	matchesHistory := false
	for _, entry := range recent {
		matches, verifyErr := hasher.Verify(password, entry.Encoded)
		if verifyErr == nil && matches {
			matchesHistory = true
		}
	}
	return operations.ensure("password-history:"+desired.ID, len(recent) != 0, matchesHistory, func() error {
		return commands.EnsurePasswordHistory(ctx, desired.ID, password, now)
	})
}

func sameDemoUser(actual, desired *userdomain.User, password string, hasher passwordports.PasswordHasher) bool {
	matches, err := hasher.Verify(password, actual.PasswordHash)
	if err != nil || !matches {
		return false
	}
	left, right := *actual, *desired
	left.Roles, right.Roles = normalizedSeedRoles(left.Roles), normalizedSeedRoles(right.Roles)
	left.PreferredUsername, right.PreferredUsername = strings.TrimSpace(left.PreferredUsername), strings.TrimSpace(right.PreferredUsername)
	left.Email, right.Email = normalizedSeedOptionalText(left.Email), normalizedSeedOptionalText(right.Email)
	left.PasswordHash, right.PasswordHash = "", ""
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func sameMfaFactor(actual, desired *totpdomain.MfaFactor) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.LastUsedAt = time.Time{}, nil
	right.CreatedAt, right.LastUsedAt = time.Time{}, nil
	return reflect.DeepEqual(left, right)
}

// seedDemoApplications は既存の OIDC クライアント / WS-Fed RP を「アプリケーション」として
// カタログに登録する。管理コンソール・アカウントポータル・demo-client・demo WS-Fed RP を
// federated Application として単一 protocol relation を作り、いずれも user_alice に割り当てる。これにより
// ポータルのアプリ一覧に並び、デモのログイン経路 (割当ゲート) も成立する (wi-69)。
// 管理コンソール / ポータルは first-party のため、割当がなくてもログイン自体は塞がない。
func SeedDemoApplications(
	ctx context.Context,
	apps appports.ApplicationRepository,
	assignments appports.AssignmentRepository,
	seed seedingdomain.DevelopmentDemoSeed,
	now time.Time,
	operations *resourceOperations,
	commands ResourceCommands,
) error {
	if apps == nil {
		return nil
	}
	for _, configured := range seed.Applications {
		protocol := appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: configured.BindingValue}
		if configured.BindingType == "wsfed" {
			protocol = appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolWsFed, Wtrealm: configured.BindingValue}
		}
		desired := &appdomain.Application{
			TenantID: tenancydomain.DefaultTenantID, ID: configured.ID, Name: configured.Name,
			Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
			LaunchURL: configured.LaunchURL, Protocol: &protocol,
			CreatedAt: now, UpdatedAt: now,
		}
		existing, err := apps.FindByID(ctx, desired.TenantID, desired.ID)
		if err != nil {
			return err
		}
		if err := operations.ensure("application:"+desired.ID, existing != nil, existing != nil && sameApplication(existing, desired), func() error {
			return commands.CreateApplication(ctx, desired)
		}); err != nil {
			return err
		}
		if assignments == nil {
			continue
		}
		desiredAssignment := &appdomain.ApplicationAssignment{
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: configured.ID,
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: configured.AssignedUser,
			Visibility: appdomain.AssignmentVisible, CreatedAt: now,
		}
		currentAssignments, err := assignments.ListByApplication(ctx, desiredAssignment.TenantID, desiredAssignment.ApplicationID)
		if err != nil {
			return err
		}
		found, same := false, false
		for _, assignment := range currentAssignments {
			if assignment.SubjectType == desiredAssignment.SubjectType && assignment.SubjectID == desiredAssignment.SubjectID {
				found = true
				same = assignment.Visibility == desiredAssignment.Visibility
			}
		}
		if err := operations.ensure("application-assignment:"+desired.ID+":"+configured.AssignedUser, found, same, func() error {
			return commands.AssignApplication(ctx, desiredAssignment)
		}); err != nil {
			return err
		}
	}
	return nil
}

// seedDemoAuthorizationDetailTypes は RFC 9396 のサンプル type を 1 件投入する。
// payment_initiation は actions を集合包含、creditorAccount を enum、instructedAmount を
// 上限 (単調減少) として扱い、エージェントに「口座 X へ最大 N まで」を束縛させる例。
func seedDemoAuthorizationDetailTypes(ctx context.Context, types oauthports.AuthorizationDetailTypeRepository, typeName string, now time.Time, operations *resourceOperations, commands ResourceCommands) error {
	if types == nil {
		return nil
	}
	desired := &oauthdomain.AuthorizationDetailType{
		TenantID:    tenancydomain.DefaultTenantID,
		Type:        typeName,
		Description: "Initiate a transfer from an account up to a specified limit (RFC 9396 example)",
		Schema: oauthdomain.AuthorizationDetailsSchema{
			Rules: []oauthdomain.AuthorizationDetailFieldRule{
				{Name: "actions", Semantics: oauthdomain.DetailFieldSet, Required: true, Allowed: []string{"initiate", "status", "cancel"}},
				{Name: "creditorAccount", Semantics: oauthdomain.DetailFieldEnum, Required: true},
				{Name: "instructedAmount", Semantics: oauthdomain.DetailFieldAtMost, Required: true},
			},
		},
		DisplayTemplate: "For account {creditorAccount}: {actions}, up to {instructedAmount}",
		State:           oauthdomain.DetailTypeEnabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	existing, err := types.FindByType(ctx, desired.TenantID, desired.Type)
	if err != nil {
		return err
	}
	return operations.ensure("authorization-detail-type:"+desired.Type, existing != nil, existing != nil && sameAuthorizationDetailType(existing, desired), func() error {
		return commands.CreateAuthorizationDetailType(ctx, desired)
	})
}

// seedDemoGroups は所属も独立した項目として比較し、途中失敗後は欠けた所属だけを補う。
func seedDemoGroups(ctx context.Context, groups groupports.GroupRepository, seeds []seedingdomain.DemoGroupSeed, now time.Time, operations *resourceOperations, commands ResourceCommands) error {
	for _, configured := range seeds {
		description := configured.Description
		group := &groupdomain.Group{ID: configured.ID, TenantID: tenancydomain.DefaultTenantID, Name: configured.Name, Description: &description, Roles: configured.Roles, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now}
		existing, err := groups.FindByID(ctx, group.TenantID, group.ID)
		if err != nil {
			return err
		}
		if err := operations.ensure("group:"+group.ID, existing != nil, existing != nil && sameGroup(existing, group), func() error {
			return commands.CreateGroup(ctx, group)
		}); err != nil {
			return err
		}
		for _, userID := range configured.Members {
			members, err := groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, configured.ID)
			if err != nil {
				return err
			}
			found, same := false, false
			for _, member := range members {
				if member.UserID == userID {
					found = true
					same = member.Source.Effective() == groupdomain.MembershipSourceManual
				}
			}
			if err := operations.ensure("group-membership:"+configured.ID+":"+userID, found, same, func() error {
				return commands.AddGroupMember(ctx, &groupdomain.GroupMember{GroupID: configured.ID, UserID: userID, CreatedAt: now})
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func sameGroup(actual, desired *groupdomain.Group) bool {
	left, right := *actual, *desired
	if len(left.Attributes) == 0 {
		left.Attributes = nil
	}
	if len(right.Attributes) == 0 {
		right.Attributes = nil
	}
	left.MembershipType, right.MembershipType = left.MembershipType.Effective(), right.MembershipType.Effective()
	left.Name, right.Name = strings.TrimSpace(left.Name), strings.TrimSpace(right.Name)
	left.Description, right.Description = normalizedSeedOptionalText(left.Description), normalizedSeedOptionalText(right.Description)
	left.Roles, right.Roles = normalizedSeedRoles(left.Roles), normalizedSeedRoles(right.Roles)
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func sameApplication(actual, desired *appdomain.Application) bool {
	left, right := *actual, *desired
	left.Name, right.Name = strings.TrimSpace(left.Name), strings.TrimSpace(right.Name)
	left.LaunchURL, right.LaunchURL = strings.TrimSpace(left.LaunchURL), strings.TrimSpace(right.LaunchURL)
	if len(left.CategoryIDs) == 0 {
		left.CategoryIDs = nil
	}
	if len(right.CategoryIDs) == 0 {
		right.CategoryIDs = nil
	}
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func normalizedSeedRoles(roles []string) []string {
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized = append(normalized, strings.TrimSpace(role))
	}
	slices.Sort(normalized)
	return slices.Compact(normalized)
}

func normalizedSeedOptionalText(description *string) *string {
	if description == nil {
		return nil
	}
	value := strings.TrimSpace(*description)
	if value == "" {
		return nil
	}
	return &value
}

func sameAuthorizationDetailType(actual, desired *oauthdomain.AuthorizationDetailType) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
