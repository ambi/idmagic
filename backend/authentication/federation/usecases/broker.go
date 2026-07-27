package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	mfausecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	ErrLinkingDenied = errors.New("federated identity cannot be linked by policy")
	ErrUserInactive  = errors.New("federated identity user is not active")
)

const (
	LinkingMethodExisting      = "existing"
	LinkingMethodExplicit      = "explicit"
	LinkingMethodVerifiedEmail = "verified_email"
	LinkingMethodJIT           = "jit"
)

type BrokerDeps struct {
	Connections   federationports.ConnectionRepository
	Identities    federationports.IdentityRepository
	Attempts      federationports.AttemptStore
	Users         userports.UserRepository
	Sessions      *sessionusecases.SessionManager
	Drivers       map[federationdomain.Protocol]ProtocolDriver
	ProvisionUser func(context.Context, federationdomain.NormalizedClaims, time.Time) (*userdomain.User, error)
	Emit          func(spec.DomainEvent)
}

type Completion struct {
	User           *userdomain.User
	Authentication *authdomain.AuthenticationContext
	LinkingMethod  string
	ReturnTo       string
}

func CompleteIdentity(
	ctx context.Context,
	deps BrokerDeps,
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	claims federationdomain.NormalizedClaims,
	now time.Time,
) (*Completion, error) {
	if !connection.Active() || claims.Subject == "" || claims.Username == "" {
		return nil, ErrLinkingDenied
	}
	now = normalizedNow(now)
	link, err := deps.Identities.FindBySubject(
		ctx, connection.TenantID, connection.ID, claims.Subject,
	)
	if err != nil {
		return nil, err
	}
	var user *userdomain.User
	method := LinkingMethodExisting
	if link != nil {
		if attempt.LinkUserID != "" {
			if link.LocalUserID != attempt.LinkUserID {
				return nil, ErrLinkingDenied
			}
			method = LinkingMethodExplicit
		}
		user, err = deps.Users.FindBySub(ctx, link.LocalUserID)
		if err != nil {
			return nil, err
		}
	} else {
		user, method, err = resolveUnlinkedIdentity(ctx, deps, connection, attempt, claims, now)
		if err != nil {
			return nil, err
		}
		link = &federationdomain.FederatedIdentity{
			TenantID: connection.TenantID, ProviderID: connection.ID,
			ExternalSubject: claims.Subject, LocalUserID: user.ID, LinkedAt: now,
		}
		if err := deps.Identities.Create(ctx, link); err != nil {
			return nil, err
		}
		emit(deps.Emit, &federationdomain.FederatedIdentityLinked{
			At: now, TenantID: connection.TenantID, UserID: user.ID,
			ProviderID: connection.ID, LinkingMethod: method,
		})
	}
	if user == nil || !user.IsActive() {
		return nil, ErrUserInactive
	}
	authn, err := deps.Sessions.Create(ctx, user.ID, []string{"federated"}, now)
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &federationdomain.FederatedAuthenticated{
		At: now, TenantID: connection.TenantID, UserID: user.ID,
		ProviderID: connection.ID, SessionID: authn.SessionID,
	})
	return &Completion{User: user, Authentication: authn, LinkingMethod: method}, nil
}

func UnlinkIdentity(
	ctx context.Context,
	deps BrokerDeps,
	authn *authdomain.AuthenticationContext,
	providerID string,
	now time.Time,
) error {
	now = normalizedNow(now)
	if authn == nil || !mfausecases.StepUpSatisfied(authn, now) {
		return mfausecases.ErrStepUpRequired
	}
	user, err := deps.Users.FindBySub(ctx, authn.UserID)
	if err != nil || user == nil || !user.IsActive() {
		return ErrUserInactive
	}
	link, err := deps.Identities.FindByUserProvider(ctx, user.TenantID, providerID, user.ID)
	if err != nil {
		return err
	}
	if link == nil {
		return nil
	}
	links, err := deps.Identities.ListByUser(ctx, user.TenantID, user.ID)
	if err != nil {
		return err
	}
	if user.PasswordHash == "" && len(links) <= 1 {
		return ErrLinkingDenied
	}
	if err := deps.Identities.Delete(ctx, user.TenantID, providerID, user.ID); err != nil {
		return err
	}
	emit(deps.Emit, &federationdomain.FederatedIdentityUnlinked{
		At: now, TenantID: user.TenantID, UserID: user.ID, ProviderID: providerID,
	})
	return nil
}

func resolveUnlinkedIdentity(
	ctx context.Context,
	deps BrokerDeps,
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	claims federationdomain.NormalizedClaims,
	now time.Time,
) (*userdomain.User, string, error) {
	if attempt.LinkUserID != "" {
		user, err := deps.Users.FindBySub(ctx, attempt.LinkUserID)
		if err != nil || user == nil || !user.IsActive() {
			return nil, "", ErrLinkingDenied
		}
		return user, LinkingMethodExplicit, nil
	}
	if connection.LinkingPolicy == federationdomain.LinkingVerifiedEmail &&
		claims.EmailVerified && claims.Email != "" {
		user, err := uniqueVerifiedEmailUser(ctx, deps.Users, connection.TenantID, claims.Email)
		if err != nil {
			return nil, "", err
		}
		if user != nil {
			return user, LinkingMethodVerifiedEmail, nil
		}
	}
	if connection.JITProvisioning && deps.ProvisionUser != nil &&
		emailDomainAllowed(claims.Email, connection.AllowedEmailDomains) {
		user, err := deps.ProvisionUser(ctx, claims, now)
		if err != nil {
			return nil, "", err
		}
		return user, LinkingMethodJIT, nil
	}
	return nil, "", ErrLinkingDenied
}

func uniqueVerifiedEmailUser(
	ctx context.Context,
	users userports.UserRepository,
	tenantID, email string,
) (*userdomain.User, error) {
	all, err := users.FindAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var match *userdomain.User
	for _, user := range all {
		if user.Email == nil || !user.EmailVerified || !strings.EqualFold(*user.Email, email) {
			continue
		}
		if match != nil {
			return nil, ErrLinkingDenied
		}
		match = user
	}
	return match, nil
}

func emailDomainAllowed(email string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
	for _, candidate := range allowed {
		if domain == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func emit(sink func(spec.DomainEvent), event spec.DomainEvent) {
	if sink != nil {
		sink(event)
	}
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
