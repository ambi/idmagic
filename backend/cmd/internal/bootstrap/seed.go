package bootstrap

import (
	"context"
	"errors"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 固定 UUID の seed id (ADR-084)。id 列は UUID 型のため、デモ/first-party の値も
// UUID にする。再起動で重複しないよう固定し、UI (frontend/src/api/oidc.ts / authFlow.ts) の
// OIDC 設定と application binding もこの値を参照する。
const (
	seedUserAliceID = "00000000-0000-4000-8000-000000000001"
	seedUserRootID  = "00000000-0000-4000-8000-000000000002"

	seedGroupEngineeringID = "00000000-0000-4000-8000-000000000011"
	seedGroupSupportID     = "00000000-0000-4000-8000-000000000012"

	seedDemoClientID          = "00000000-0000-4000-8000-000000000021"
	seedAdminConsoleClientID  = "00000000-0000-4000-8000-000000000022"
	seedAccountPortalClientID = "00000000-0000-4000-8000-000000000023"
)

// seedDemoData は SKIP_DEMO_SEED が空のとき、デモ用クライアントとユーザーを 1 件投入する。
// 既存データを更新する想定で Save を直接呼ぶ。
func SeedDemoData(
	ctx context.Context,
	clients oauthports.OAuth2ClientRepository,
	users idmports.UserRepository,
	mfaFactors authnports.MfaFactorRepository,
	passwordHistory authnports.PasswordHistoryRepository,
	groups idmports.GroupRepository,
	authzDetailTypes oauthports.AuthorizationDetailTypeRepository,
	hasher authnports.PasswordHasher,
) error {
	secretHash := oauthdomain.HashClientSecret(EnvDefault("DEMO_CLIENT_SECRET", "demo-client-secret"))
	now := time.Now().UTC()
	if err := clients.Save(ctx, &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: seedDemoClientID,
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		RedirectURIs: []string{
			"http://localhost:3000/callback",
			"http://localhost:5173/callback",
			"http://localhost:8080/callback",
		},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantClientCredentials, spec.GrantDeviceCode,
		},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		Scope:                   "openid profile email offline_access", IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}
	if err := seedFirstPartyPortalClients(ctx, clients, now); err != nil {
		return err
	}
	password := EnvDefault("DEMO_USER_PASSWORD", "demo-password-1234")
	if result := authusecases.ValidatePassword(password); !result.OK {
		return errors.New("DEMO_USER_PASSWORD violates password policy")
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	email := "alice@example.com"
	totpSecret := EnvDefault("DEMO_TOTP_SECRET", "")
	if err := users.Save(ctx, &idmdomain.User{
		ID: seedUserAliceID, TenantID: tenancydomain.DefaultTenantID,
		PreferredUsername: "alice", PasswordHash: hash,
		Email: &email, EmailVerified: true, MfaEnrolled: totpSecret != "",
		Roles:     []string{"admin"},
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}
	if err := passwordHistory.Add(ctx, seedUserAliceID, hash, now); err != nil {
		return err
	}
	// root は super-admin デモユーザー。system_admin はテナント横断の管理操作
	// (例: /admin/keys/health の全テナント署名鍵ヘルス) 専用ロールで admin の
	// 上位集合ではないため、一般管理コンソール (RequireAdmin が要求する admin
	// ロール) とあわせて両方を付与し、全画面を試せるようにする。alice とは別に
	// 用意し、既定テナントに所属する。
	rootEmail := "root@example.com"
	if err := users.Save(ctx, &idmdomain.User{
		ID: seedUserRootID, TenantID: tenancydomain.DefaultTenantID,
		PreferredUsername: "root", PasswordHash: hash,
		Email: &rootEmail, EmailVerified: true,
		Roles:     []string{"admin", "system_admin"},
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}
	if err := passwordHistory.Add(ctx, seedUserRootID, hash, now); err != nil {
		return err
	}
	if err := seedDemoGroups(ctx, groups, now); err != nil {
		return err
	}
	if err := seedDemoAuthorizationDetailTypes(ctx, authzDetailTypes, now); err != nil {
		return err
	}
	if totpSecret == "" {
		return nil
	}
	label := "Demo TOTP"
	return mfaFactors.Save(ctx, &authdomain.MfaFactor{
		UserID: seedUserAliceID, Type: spec.MfaFactorTOTP, Secret: &totpSecret, Label: &label, CreatedAt: now,
	})
}

// seedFirstPartyPortalClients は管理コンソールとアカウントポータルを自分自身の IdP の
// OIDC RP として登録する (ADR-061 / [[wi-66-portals-as-oidc-rp]])。両者は public +
// authorization_code + PKCE のファーストパーティ SPA クライアントで、client secret を
// 持たない (token_endpoint_auth_method = none)。redirect_uri は SPA の `/callback`。
func seedFirstPartyPortalClients(ctx context.Context, clients oauthports.OAuth2ClientRepository, now time.Time) error {
	portals := []struct {
		clientID string
		name     string
		scope    string
	}{
		{seedAdminConsoleClientID, "IdMagic Admin Console", "openid profile idmagic.admin offline_access"},
		{seedAccountPortalClientID, "IdMagic Account Portal", "openid profile idmagic.account offline_access"},
	}
	for _, p := range portals {
		name := p.name
		if err := clients.Save(ctx, &oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: p.clientID,
			ClientName: &name, ClientType: spec.ClientPublic,
			RedirectURIs: []string{
				"http://localhost:3000/callback",
				"http://localhost:5173/callback",
				"http://localhost:8080/callback",
			},
			GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
			TokenEndpointAuthMethod: oauthdomain.AuthMethodNone,
			Scope:                   p.scope, IDTokenSignedResponseAlg: spec.SigAlgPS256,
			FapiProfile: oauthdomain.FapiNone, FirstParty: true, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// seedDemoApplications は既存の OIDC クライアント / WS-Fed RP を「アプリケーション」として
// カタログに登録する。管理コンソール・アカウントポータル・demo-client・demo WS-Fed RP を
// federated Application として binding 接続し、いずれも user_alice に割り当てる。これにより
// ポータルのアプリ一覧に並び、デモのログイン経路 (割当ゲート) も成立する (wi-69)。
// 管理コンソール / ポータルは first-party のため、割当がなくてもログイン自体は塞がない。
func SeedDemoApplications(
	ctx context.Context,
	apps appports.ApplicationRepository,
	assignments appports.AssignmentRepository,
	now time.Time,
) error {
	if apps == nil {
		return nil
	}
	seeds := []struct {
		id        string
		name      string
		launchURL string
		binding   appdomain.ProtocolBinding
	}{
		{"00000000-0000-4000-8000-000000000101", "IdMagic Admin Console", "/realms/default/admin", appdomain.ProtocolBinding{Type: appdomain.ProtocolBindingOIDC, ClientID: seedAdminConsoleClientID}},
		{"00000000-0000-4000-8000-000000000102", "IdMagic Account Portal", "/realms/default/account", appdomain.ProtocolBinding{Type: appdomain.ProtocolBindingOIDC, ClientID: seedAccountPortalClientID}},
		{"00000000-0000-4000-8000-000000000103", "Demo Client", "", appdomain.ProtocolBinding{Type: appdomain.ProtocolBindingOIDC, ClientID: seedDemoClientID}},
		{"00000000-0000-4000-8000-000000000104", "Demo WS-Federation RP", "https://rp.example/wsfed", appdomain.ProtocolBinding{Type: appdomain.ProtocolBindingWsFed, Wtrealm: "urn:idmagic:demo-rp"}},
	}
	for _, s := range seeds {
		if err := apps.Save(ctx, &appdomain.Application{
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: s.id, Name: s.name,
			Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
			LaunchURL: s.launchURL, Bindings: []appdomain.ProtocolBinding{s.binding},
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		if assignments == nil {
			continue
		}
		if err := assignments.Save(ctx, &appdomain.ApplicationAssignment{
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: s.id,
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: seedUserAliceID,
			Visibility: appdomain.AssignmentVisible, CreatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// seedDemoAuthorizationDetailTypes は RFC 9396 のサンプル type を 1 件投入する (ADR-050)。
// payment_initiation は actions を集合包含、creditorAccount を enum、instructedAmount を
// 上限 (単調減少) として扱い、エージェントに「口座 X へ最大 N まで」を束縛させる例。
func seedDemoAuthorizationDetailTypes(ctx context.Context, types oauthports.AuthorizationDetailTypeRepository, now time.Time) error {
	if types == nil {
		return nil
	}
	return types.Save(ctx, &oauthdomain.AuthorizationDetailType{
		TenantID:    tenancydomain.DefaultTenantID,
		Type:        "payment_initiation",
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
	})
}

// seedDemoGroups は固定 ID のデモ用グループ engineering / support を投入し、alice を
// engineering に所属させる。再起動時に重複しないよう ID は固定し、Save は id 上の
// upsert、AddMember は冪等 (no-op on conflict) を利用する。これにより demo.sh で
// グループ由来ロール (engineering → catalog:read) を確認できる。
func seedDemoGroups(ctx context.Context, groups idmports.GroupRepository, now time.Time) error {
	engineeringDesc := "プロダクト開発チーム"
	supportDesc := "カスタマーサポートチーム"
	demoGroups := []*idmdomain.Group{
		{
			ID: seedGroupEngineeringID, TenantID: tenancydomain.DefaultTenantID, Name: "engineering",
			Description: &engineeringDesc, Roles: []string{"catalog:read"}, CreatedAt: now,
		},
		{
			ID: seedGroupSupportID, TenantID: tenancydomain.DefaultTenantID, Name: "support",
			Description: &supportDesc, Roles: []string{"invoice:read"}, CreatedAt: now,
		},
	}
	for _, group := range demoGroups {
		if err := groups.Save(ctx, group); err != nil {
			return err
		}
	}
	if _, err := groups.AddMember(ctx, &idmdomain.GroupMember{
		GroupID: seedGroupEngineeringID, UserID: seedUserAliceID, CreatedAt: now,
	}); err != nil {
		return err
	}
	return nil
}
