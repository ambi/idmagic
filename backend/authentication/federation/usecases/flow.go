package usecases

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	"github.com/ambi/idmagic/backend/tenancy"
)

type ProtocolDriver interface {
	Start(
		federationdomain.IdentityProviderConnection,
		federationdomain.FederatedLoginAttempt,
		string,
		time.Time,
	) (string, error)
	Complete(
		context.Context,
		federationdomain.IdentityProviderConnection,
		federationdomain.FederatedLoginAttempt,
		string,
		string,
		time.Time,
	) (federationdomain.NormalizedClaims, error)
}

type LoginStart struct {
	State      string
	RedirectTo string
}

func StartLogin(
	ctx context.Context,
	deps BrokerDeps,
	providerID, returnTo, linkUserID, callbackURL string,
	now time.Time,
) (*LoginStart, error) {
	tenantID := tenancy.TenantID(ctx)
	connection, err := deps.Connections.Find(ctx, tenantID, providerID)
	if err != nil || connection == nil || !connection.Active() {
		return nil, ErrLinkingDenied
	}
	driver := deps.Drivers[connection.Protocol]
	if driver == nil {
		return nil, errors.New("federation protocol driver unavailable")
	}
	state, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	createdAt := normalizedNow(now)
	attempt := &federationdomain.FederatedLoginAttempt{
		State: state, TenantID: tenantID, ProviderID: connection.ID,
		Protocol: connection.Protocol, ReturnTo: returnTo, LinkUserID: linkUserID,
		CreatedAt: createdAt, ExpiresAt: createdAt.Add(10 * time.Minute),
	}
	switch connection.Protocol {
	case federationdomain.ProtocolOIDC:
		attempt.Nonce, err = randomToken(32)
		if err == nil {
			attempt.PKCEVerifier, err = randomToken(48)
		}
	case federationdomain.ProtocolSAML:
		var id string
		id, err = randomToken(24)
		attempt.RequestID = "_" + id
	default:
		err = errors.New("unsupported federation protocol")
	}
	if err != nil {
		return nil, err
	}
	redirectTo, err := driver.Start(*connection, *attempt, callbackURL, now)
	if err != nil {
		return nil, err
	}
	if err := deps.Attempts.Save(ctx, attempt); err != nil {
		return nil, err
	}
	return &LoginStart{State: state, RedirectTo: redirectTo}, nil
}

func CompleteLogin(
	ctx context.Context,
	deps BrokerDeps,
	state, response, callbackURL string,
	now time.Time,
) (*Completion, error) {
	tenantID := tenancy.TenantID(ctx)
	attempt, err := deps.Attempts.Consume(ctx, tenantID, state, normalizedNow(now))
	if err != nil {
		return nil, err
	}
	connection, err := deps.Connections.Find(ctx, tenantID, attempt.ProviderID)
	if err != nil || connection == nil || !connection.Active() {
		return nil, ErrLinkingDenied
	}
	driver := deps.Drivers[connection.Protocol]
	if driver == nil {
		return nil, errors.New("federation protocol driver unavailable")
	}
	claims, err := driver.Complete(ctx, *connection, *attempt, response, callbackURL, now)
	if err != nil {
		emit(deps.Emit, &federationdomain.FederatedLoginRejected{
			At: normalizedNow(now), TenantID: tenantID, ProviderID: connection.ID,
			Reason: "protocol_validation_failed",
		})
		return nil, err
	}
	completion, err := CompleteIdentity(ctx, deps, *connection, *attempt, claims, now)
	if err != nil {
		return nil, err
	}
	completion.ReturnTo = attempt.ReturnTo
	return completion, nil
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
