package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	passwordusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 固定 UUID の seed id (ADR-084)。id 列は UUID 型のため、デモ/first-party の値も
// UUID にする。再起動で重複しないよう固定し、UI (frontend/src/api/oidc.ts / authFlow.ts) の
// OIDC 設定と application binding もこの値を参照する。
const (
	seedUserAliceID  = "00000000-0000-4000-8000-000000000001" // test assertion for the repository default manifest
	seedDemoClientID = "00000000-0000-4000-8000-000000000021" // test assertion for the repository default manifest
)

// seedDemoData は SKIP_DEMO_SEED が空のとき、デモ用クライアントとユーザーを 1 件投入する。
// 既存データを更新する想定で Save を直接呼ぶ。
func SeedDemoData(
	ctx context.Context,
	clients oauthports.OAuth2ClientRepository,
	users userports.UserRepository,
	mfaFactors totpports.MfaFactorRepository,
	passwordHistory passwordports.PasswordHistoryRepository,
	groups groupports.GroupRepository,
	authzDetailTypes oauthports.AuthorizationDetailTypeRepository,
	hasher passwordports.PasswordHasher,
	seed domain.DevelopmentDemoSeed,
	clientSecret string,
	password string,
	totpSecret string,
) error {
	secretHash := oauthdomain.HashClientSecret(clientSecret)
	now := time.Now().UTC()
	demoClient := &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: seed.ClientID,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: seed.ClientRedirectURIs,
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantClientCredentials, spec.GrantDeviceCode,
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
	if currentClient == nil {
		if err := clients.Save(ctx, demoClient); err != nil {
			return err
		}
	} else if !sameDemoClient(currentClient, demoClient, clientSecret) {
		return fmt.Errorf("seed drift at oauth2-client:%s", seed.ClientID)
	}
	if result := passwordusecases.ValidatePassword(password); !result.OK {
		return errors.New("seed user password violates password policy")
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	for index, configured := range seed.Users {
		email := configured.Email
		user := &userdomain.User{
			ID: configured.ID, TenantID: tenancydomain.DefaultTenantID,
			PreferredUsername: configured.PreferredUsername, PasswordHash: hash,
			Email: &email, EmailVerified: true, MfaEnrolled: index == 0 && totpSecret != "",
			Roles: configured.Roles, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := ensureDemoUser(ctx, users, passwordHistory, hasher, user, password, now); err != nil {
			return err
		}
	}
	if err := seedDemoGroups(ctx, groups, seed.Groups, now); err != nil {
		return err
	}
	if err := seedDemoAuthorizationDetailTypes(ctx, authzDetailTypes, seed.AuthorizationDetailType, now); err != nil {
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
	if existingFactor == nil {
		return mfaFactors.Save(ctx, desiredFactor)
	}
	if !sameMfaFactor(existingFactor, desiredFactor) {
		return fmt.Errorf("seed drift at mfa-factor:%s:totp", seed.Users[0].ID)
	}
	return nil
}

func sameDemoClient(actual, desired *oauthdomain.OAuth2Client, secret string) bool {
	if actual.ClientSecretHash == nil || !oauthdomain.VerifyClientSecret(secret, *actual.ClientSecretHash) {
		return false
	}
	left, right := *actual, *desired
	left.ClientSecretHash, right.ClientSecretHash = nil, nil
	return sameClient(&left, &right)
}

func ensureDemoUser(ctx context.Context, users userports.UserRepository, history passwordports.PasswordHistoryRepository, hasher passwordports.PasswordHasher, desired *userdomain.User, password string, now time.Time) error {
	existing, err := users.FindBySub(ctx, desired.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		if err := users.Save(ctx, desired); err != nil {
			return err
		}
		return history.Add(ctx, desired.ID, desired.PasswordHash, now)
	}
	if !sameDemoUser(existing, desired, password, hasher) {
		return fmt.Errorf("seed drift at user:%s", desired.ID)
	}
	recent, err := history.Recent(ctx, desired.ID, 24)
	if err != nil {
		return err
	}
	for _, entry := range recent {
		matches, verifyErr := hasher.Verify(password, entry.Encoded)
		if verifyErr == nil && matches {
			return nil
		}
	}
	if len(recent) != 0 {
		return fmt.Errorf("seed drift at password-history:%s", desired.ID)
	}
	return history.Add(ctx, desired.ID, desired.PasswordHash, now)
}

func sameDemoUser(actual, desired *userdomain.User, password string, hasher passwordports.PasswordHasher) bool {
	matches, err := hasher.Verify(password, actual.PasswordHash)
	if err != nil || !matches {
		return false
	}
	left, right := *actual, *desired
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
	seed domain.DevelopmentDemoSeed,
	now time.Time,
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
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: configured.ID, Name: configured.Name,
			Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
			LaunchURL: configured.LaunchURL, Protocol: &protocol,
			CreatedAt: now, UpdatedAt: now,
		}
		existing, err := apps.FindByID(ctx, desired.TenantID, desired.ApplicationID)
		if err != nil {
			return err
		}
		if existing == nil {
			if err := apps.Create(ctx, desired); err != nil {
				return err
			}
		} else if !sameApplication(existing, desired) {
			return fmt.Errorf("seed drift at application:%s", desired.ApplicationID)
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
		found := false
		for _, assignment := range currentAssignments {
			if assignment.SubjectType == desiredAssignment.SubjectType && assignment.SubjectID == desiredAssignment.SubjectID {
				found = true
				if assignment.Visibility != desiredAssignment.Visibility {
					return fmt.Errorf("seed drift at application-assignment:%s", desired.ApplicationID)
				}
			}
		}
		if !found {
			if err := assignments.Save(ctx, desiredAssignment); err != nil {
				return err
			}
		}
	}
	return nil
}

// seedDemoAuthorizationDetailTypes は RFC 9396 のサンプル type を 1 件投入する (ADR-050)。
// payment_initiation は actions を集合包含、creditorAccount を enum、instructedAmount を
// 上限 (単調減少) として扱い、エージェントに「口座 X へ最大 N まで」を束縛させる例。
func seedDemoAuthorizationDetailTypes(ctx context.Context, types oauthports.AuthorizationDetailTypeRepository, typeName string, now time.Time) error {
	if types == nil {
		return nil
	}
	desired := &oauthdomain.AuthorizationDetailType{
		TenantID:    tenancydomain.DefaultTenantID,
		Type:        typeName,
		Description: "口座から指定上限までの送金開始 (RFC 9396 例)",
		Schema: oauthdomain.AuthorizationDetailsSchema{
			Rules: []oauthdomain.AuthorizationDetailFieldRule{
				{Name: "actions", Semantics: oauthdomain.DetailFieldSet, Required: true, Allowed: []string{"initiate", "status", "cancel"}},
				{Name: "creditorAccount", Semantics: oauthdomain.DetailFieldEnum, Required: true},
				{Name: "instructedAmount", Semantics: oauthdomain.DetailFieldAtMost, Required: true},
			},
		},
		DisplayTemplate: "口座 {creditorAccount} に対して {actions} を、最大 {instructedAmount} まで",
		State:           oauthdomain.DetailTypeEnabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	existing, err := types.FindByType(ctx, desired.TenantID, desired.Type)
	if err != nil {
		return err
	}
	if existing == nil {
		return types.Save(ctx, desired)
	}
	if !sameAuthorizationDetailType(existing, desired) {
		return fmt.Errorf("seed drift at authorization-detail-type:%s", desired.Type)
	}
	return nil
}

// seedDemoGroups は固定 ID のデモ用グループ engineering / support を投入し、alice を
// engineering に所属させる。再起動時に重複しないよう ID は固定し、Save は id 上の
// upsert、AddMember は冪等 (no-op on conflict) を利用する。これにより demo.sh で
// グループ由来ロール (engineering → catalog:read) を確認できる。
func seedDemoGroups(ctx context.Context, groups groupports.GroupRepository, seeds []domain.DemoGroupSeed, now time.Time) error {
	for _, configured := range seeds {
		description := configured.Description
		group := &groupdomain.Group{ID: configured.ID, TenantID: tenancydomain.DefaultTenantID, Name: configured.Name, Description: &description, Roles: configured.Roles, CreatedAt: now}
		existing, err := groups.FindByID(ctx, group.TenantID, group.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			if err := groups.Save(ctx, group); err != nil {
				return err
			}
		} else if !sameGroup(existing, group) {
			return fmt.Errorf("seed drift at group:%s", group.ID)
		}
		for _, userID := range configured.Members {
			members, err := groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, configured.ID)
			if err != nil {
				return err
			}
			found := false
			for _, member := range members {
				if member.UserID == userID {
					found = true
					if member.Source.Effective() != groupdomain.MembershipSourceManual {
						return fmt.Errorf("seed drift at group-membership:%s:%s", configured.ID, userID)
					}
				}
			}
			if !found {
				if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: configured.ID, UserID: userID, CreatedAt: now}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sameGroup(actual, desired *groupdomain.Group) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func sameApplication(actual, desired *appdomain.Application) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func sameAuthorizationDetailType(actual, desired *oauthdomain.AuthorizationDetailType) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
