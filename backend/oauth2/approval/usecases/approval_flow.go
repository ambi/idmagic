package usecases

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/approval/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type StartApprovalInput struct {
	ClientID                string
	LoginHint               string
	IDTokenHint             string
	Scope                   string
	AuthorizationDetailsRaw string
	BindingMessage          string
	RequestedExpiry         *int
}

type StartApprovalDeps struct {
	ClientRepo          oauthports.OAuth2ClientRepository
	UserRepo            userports.UserRepository
	AgentRepo           agentports.AgentRepository
	Store               approvalports.ApprovalRequestStore
	HintVerifier        oauthports.IDTokenHintVerifier
	AuthzDetailTypeRepo oauthports.AuthorizationDetailTypeRepository
	Notifier            notificationports.Notifier
	ApprovalURL         string
	Emit                func(spec.DomainEvent)
}

type StartApprovalResult struct {
	AuthReqID string `json:"auth_req_id"`
	ExpiresIn int    `json:"expires_in"`
	Interval  int    `json:"interval"`
}

func ValidateStartApprovalInput(in StartApprovalInput) error {
	hints := 0
	if strings.TrimSpace(in.LoginHint) != "" {
		hints++
	}
	if strings.TrimSpace(in.IDTokenHint) != "" {
		hints++
	}
	if hints != 1 {
		return sharedusecases.NewOAuthError("invalid_request", "exactly one user hint is required")
	}
	scopes := strings.Fields(in.Scope)
	if len(scopes) == 0 || !slices.Contains(scopes, "openid") {
		return sharedusecases.NewOAuthError("invalid_scope", "scope must contain openid")
	}
	if utf8.RuneCountInString(in.BindingMessage) > approvaldomain.MaxBindingMessageLength ||
		strings.IndexFunc(in.BindingMessage, unicode.IsControl) >= 0 {
		return sharedusecases.NewOAuthError("invalid_binding_message", "binding_message is not suitable for display")
	}
	if in.RequestedExpiry != nil && (*in.RequestedExpiry <= 0 || *in.RequestedExpiry > int(approvaldomain.MaxTTL.Seconds())) {
		return sharedusecases.NewOAuthError("invalid_request", "requested_expiry must be between 1 and 600 seconds")
	}
	return nil
}

func StartApproval(ctx context.Context, deps StartApprovalDeps, in StartApprovalInput, now time.Time) (*StartApprovalResult, error) {
	if err := ValidateStartApprovalInput(in); err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, sharedusecases.NewOAuthError("invalid_client", "unknown client")
	}
	if client.ClientType != spec.ClientConfidential || !slices.Contains(client.GrantTypes, spec.GrantCiba) {
		return nil, sharedusecases.NewOAuthError("unauthorized_client", "ciba grant is not allowed")
	}
	scopes := strings.Fields(in.Scope)
	allowed := strings.Fields(client.Scope)
	for _, scope := range scopes {
		if !slices.Contains(allowed, scope) {
			return nil, sharedusecases.NewOAuthError("invalid_scope", "scope is not declared by the client")
		}
	}
	details, err := sharedusecases.ParseAuthorizationDetails(in.AuthorizationDetailsRaw)
	if err != nil {
		return nil, err
	}
	if err := sharedusecases.ValidateAuthorizationDetails(ctx, deps.AuthzDetailTypeRepo, details); err != nil {
		return nil, err
	}
	user, err := resolveApprovalUser(ctx, deps, in, client.ClientID, now)
	if err != nil {
		return nil, err
	}
	var agentID *string
	agentName := ""
	if deps.AgentRepo != nil {
		agent, findErr := deps.AgentRepo.FindByClientID(ctx, tenantID, client.ClientID)
		if findErr != nil {
			return nil, findErr
		}
		if agent != nil {
			if !agent.IsActive() {
				return nil, sharedusecases.NewOAuthError("invalid_client", "agent is disabled or killed")
			}
			agentID = &agent.ID
			agentName = agent.Name
		}
	}
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		return nil, err
	}
	authReqID, err := approvaldomain.GenerateAuthReqID()
	if err != nil {
		return nil, err
	}
	ttl := approvaldomain.ResolveTTL(in.RequestedExpiry)
	interval := spec.DefaultDeviceCodePolling().DefaultIntervalSeconds
	binding := optionalString(in.BindingMessage)
	rec := &approvaldomain.ApprovalRequest{
		ID: id, TenantID: tenantID, ClientID: client.ClientID, AgentID: agentID, UserID: user.ID,
		Scopes: slices.Clone(scopes), AuthorizationDetails: details, BindingMessage: binding,
		State: spec.ApprovalPending, AuthReqIDHash: approvaldomain.HashAuthReqID(authReqID),
		IntervalSeconds: interval, RequestedAt: now, ExpiresAt: now.Add(ttl),
	}
	if err := rec.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Store.Save(ctx, rec); err != nil {
		return nil, err
	}
	emit(deps.Emit, &oauthdomain.BackchannelAuthRequested{
		At: now, TenantID: tenantID, ApprovalRequestID: rec.ID, ClientID: rec.ClientID,
		AgentID: deref(agentID), UserID: rec.UserID, Scopes: slices.Clone(rec.Scopes),
	})
	notifyApproval(ctx, deps, client, user, agentName, in.BindingMessage, ttl)
	return &StartApprovalResult{AuthReqID: authReqID, ExpiresIn: int(ttl.Seconds()), Interval: interval}, nil
}

func resolveApprovalUser(ctx context.Context, deps StartApprovalDeps, in StartApprovalInput, clientID string, now time.Time) (*userdomain.User, error) {
	var user *userdomain.User
	var err error
	if in.IDTokenHint != "" {
		if deps.HintVerifier == nil {
			return nil, sharedusecases.NewOAuthError("unknown_user_id", "user could not be resolved")
		}
		claims, verifyErr := deps.HintVerifier.VerifyIDTokenHint(ctx, in.IDTokenHint)
		if verifyErr == nil && claims != nil && hintAudienceContains(claims, clientID) &&
			claims.Subject != "" && claims.ExpiresAt > now.Unix() {
			user, err = deps.UserRepo.FindBySub(ctx, claims.Subject)
		}
	} else {
		hint := strings.TrimSpace(in.LoginHint)
		user, err = deps.UserRepo.FindByUsername(ctx, tenancy.TenantID(ctx), hint)
		if err == nil && user == nil {
			user, err = deps.UserRepo.FindByEmail(ctx, tenancy.TenantID(ctx), hint)
		}
	}
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) || !user.IsActive() {
		return nil, sharedusecases.NewOAuthError("unknown_user_id", "user could not be resolved")
	}
	return user, nil
}

func hintAudienceContains(claims *oauthports.IDTokenHintClaims, clientID string) bool {
	if len(claims.Audiences) > 0 {
		return slices.Contains(claims.Audiences, clientID)
	}
	return claims.Audience == clientID
}

type ExchangeApprovalInput struct {
	ClientID     string
	AuthReqID    string
	ProofJKT     string
	ProofX5TS256 string
}

type ExchangeApprovalDeps struct {
	ClientRepo           oauthports.OAuth2ClientRepository
	UserRepo             userports.UserRepository
	AgentRepo            agentports.AgentRepository
	Store                approvalports.ApprovalRequestStore
	TokenIssuer          oauthports.TokenIssuer
	ResolveAttributeDefs func(context.Context, string) ([]userdomain.UserAttributeDef, error)
	Emit                 func(spec.DomainEvent)
}

type ExchangeApprovalResult struct {
	AccessToken string
	IDToken     string
	TokenType   string
	ExpiresIn   int
	Scope       string
}

func ExchangeApproval(ctx context.Context, deps ExchangeApprovalDeps, in ExchangeApprovalInput, now time.Time) (*ExchangeApprovalResult, error) {
	if strings.TrimSpace(in.AuthReqID) == "" {
		return nil, sharedusecases.NewOAuthError("invalid_request", "auth_req_id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	hash := approvaldomain.HashAuthReqID(in.AuthReqID)
	rec, err := deps.Store.FindByAuthReqIDHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.ClientID != in.ClientID {
		return nil, sharedusecases.NewOAuthError("invalid_grant", "unknown auth_req_id")
	}
	if approvaldomain.IsExpired(rec, now) {
		if expired, expireErr := deps.Store.Expire(ctx, hash, now); expireErr != nil {
			return nil, expireErr
		} else if expired != nil {
			emit(deps.Emit, &oauthdomain.BackchannelAuthExpired{At: now, TenantID: rec.TenantID, ApprovalRequestID: rec.ID, ClientID: rec.ClientID, UserID: rec.UserID})
		}
		return nil, sharedusecases.NewOAuthError("expired_token", "auth_req_id has expired")
	}
	switch rec.State {
	case spec.ApprovalPending:
		polled, tooFast, pollErr := deps.Store.RecordPoll(ctx, hash, now)
		if pollErr != nil {
			return nil, pollErr
		}
		if polled == nil {
			return nil, sharedusecases.NewOAuthError("invalid_grant", "unknown auth_req_id")
		}
		if polled.State != spec.ApprovalPending {
			rec = polled
			break
		}
		if tooFast {
			return nil, sharedusecases.NewOAuthError("slow_down", "polling interval is too short")
		}
		return nil, sharedusecases.NewOAuthError("authorization_pending", "authorization is pending")
	case spec.ApprovalDenied:
		return nil, sharedusecases.NewOAuthError("access_denied", "the user denied the request")
	case spec.ApprovalConsumed:
		return nil, sharedusecases.NewOAuthError("invalid_grant", "auth_req_id has already been consumed")
	case spec.ApprovalExpired:
		return nil, sharedusecases.NewOAuthError("expired_token", "auth_req_id has expired")
	}
	if rec.State != spec.ApprovalApproved {
		switch rec.State {
		case spec.ApprovalDenied:
			return nil, sharedusecases.NewOAuthError("access_denied", "the user denied the request")
		case spec.ApprovalConsumed:
			return nil, sharedusecases.NewOAuthError("invalid_grant", "auth_req_id has already been consumed")
		case spec.ApprovalExpired:
			return nil, sharedusecases.NewOAuthError("expired_token", "auth_req_id has expired")
		default:
			return nil, sharedusecases.NewOAuthError("invalid_grant", "approval request is not approved")
		}
	}
	client, err := deps.ClientRepo.FindByID(ctx, tenancy.TenantID(ctx), in.ClientID)
	if err != nil || client == nil {
		if err != nil {
			return nil, err
		}
		return nil, sharedusecases.NewOAuthError("invalid_grant", "client is unavailable")
	}
	user, err := deps.UserRepo.FindBySub(ctx, rec.UserID)
	if err != nil || user == nil || !user.IsActive() || user.TenantID != tenancy.TenantID(ctx) {
		if err != nil {
			return nil, err
		}
		return nil, sharedusecases.NewOAuthError("invalid_grant", "user is unavailable")
	}
	if rec.AgentID != nil {
		if deps.AgentRepo == nil {
			return nil, sharedusecases.NewOAuthError("invalid_grant", "agent status cannot be verified")
		}
		agent, findErr := deps.AgentRepo.FindByID(ctx, tenancy.TenantID(ctx), *rec.AgentID)
		if findErr != nil {
			return nil, findErr
		}
		if agent == nil || !agent.IsActive() {
			return nil, sharedusecases.NewOAuthError("invalid_grant", "agent is disabled or killed")
		}
	}
	consumed, err := deps.Store.Consume(ctx, hash, now)
	if err != nil {
		return nil, err
	}
	if consumed == nil {
		return nil, sharedusecases.NewOAuthError("invalid_grant", "auth_req_id was consumed concurrently")
	}
	var constraint *oauthdomain.SenderConstraint
	if in.ProofJKT != "" {
		constraint = &oauthdomain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: in.ProofJKT}
	} else if in.ProofX5TS256 != "" {
		constraint = &oauthdomain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: in.ProofX5TS256}
	}
	authTime := now.Unix()
	if consumed.DecidedAt != nil {
		authTime = consumed.DecidedAt.Unix()
	}
	agentID := deref(consumed.AgentID)
	access, jti, err := deps.TokenIssuer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: client, Sub: user.ID, Scopes: consumed.Scopes, SenderConstraint: constraint,
		AuthTime: authTime, AgentID: agentID, AuthorizationDetails: consumed.AuthorizationDetails,
	})
	if err != nil {
		return nil, err
	}
	idToken, err := deps.TokenIssuer.SignIDToken(ctx, oauthports.IDTokenInput{
		Client: client, User: user, Scopes: consumed.Scopes, AuthTime: authTime,
		AtHashFor: access, ResolveAttributeDefs: deps.ResolveAttributeDefs,
	})
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &oauthdomain.AccessTokenIssued{At: now, TenantID: consumed.TenantID, JTI: jti, ClientID: client.ClientID, UserID: user.ID, Scopes: slices.Clone(consumed.Scopes), SenderConstraint: senderConstraintTag(constraint)})
	tokenType := "Bearer"
	if constraint != nil && constraint.Type == spec.SenderConstraintDPoP {
		tokenType = "DPoP"
	}
	return &ExchangeApprovalResult{AccessToken: access, IDToken: idToken, TokenType: tokenType, ExpiresIn: deps.TokenIssuer.AccessTokenTTLSeconds(), Scope: strings.Join(consumed.Scopes, " ")}, nil
}

func ListPendingApprovals(ctx context.Context, store approvalports.ApprovalRequestStore, userID string) ([]*approvaldomain.ApprovalRequest, error) {
	return store.ListPendingForUser(ctx, userID)
}

func DecideApproval(ctx context.Context, store approvalports.ApprovalRequestStore, emitFn func(spec.DomainEvent), userID, id string, approve bool, now time.Time) error {
	rec, err := store.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil || rec.UserID != userID {
		return sharedusecases.NewOAuthError("access_denied", "approval request is not owned by the user")
	}
	if approvaldomain.IsExpired(rec, now) || rec.State != spec.ApprovalPending {
		return sharedusecases.NewOAuthError("invalid_request", "approval request is no longer pending")
	}
	event := spec.ApprovalEventDeny
	if approve {
		event = spec.ApprovalEventApprove
	}
	decided, err := store.Decide(ctx, id, userID, event, now)
	if err != nil {
		return err
	}
	if decided == nil {
		return sharedusecases.NewOAuthError("invalid_request", "approval request was already decided")
	}
	if approve {
		emit(emitFn, &oauthdomain.BackchannelAuthApproved{At: now, TenantID: decided.TenantID, ApprovalRequestID: decided.ID, ClientID: decided.ClientID, UserID: userID})
	} else {
		emit(emitFn, &oauthdomain.BackchannelAuthDenied{At: now, TenantID: decided.TenantID, ApprovalRequestID: decided.ID, ClientID: decided.ClientID, UserID: userID})
	}
	return nil
}

func notifyApproval(ctx context.Context, deps StartApprovalDeps, client *oauthdomain.OAuth2Client, user *userdomain.User, agentName, binding string, ttl time.Duration) {
	if deps.Notifier == nil || user.Email == nil || strings.TrimSpace(*user.Email) == "" {
		return
	}
	clientName := client.ClientID
	if client.ClientName != nil && strings.TrimSpace(*client.ClientName) != "" {
		clientName = *client.ClientName
	}
	agentSuffix := ""
	if agentName != "" {
		agentSuffix = fmt.Sprintf(" (%s)", agentName)
	}
	deps.Notifier.Notify(ctx, notificationports.Notification{
		TenantID: user.TenantID, To: *user.Email, Key: notificationports.TemplateKeyAgentActionApprovalRequest,
		RecipientLocale: user.LocaleAttribute(), Vars: map[string]string{
			"approval_url": deps.ApprovalURL, "client_name": clientName, "agent_name": agentSuffix,
			"binding_message": binding, "expires_in_minutes": fmt.Sprintf("%d", int(ttl.Minutes())),
			"user_display_name": user.DisplayName(),
		},
	})
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func emit(fn func(spec.DomainEvent), event spec.DomainEvent) {
	if fn != nil {
		fn(event)
	}
}

func senderConstraintTag(sc *oauthdomain.SenderConstraint) string {
	if sc == nil {
		return "none"
	}
	return string(sc.Type)
}
