package sourcingdeps

import (
	"context"
	"strings"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/sourcing"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
)

// PasswordDelegation は提供側の詳細な LDAP の結果を、認証側の必要な三つの事実へ写す。
func PasswordDelegation(source sourcing.Module, users userports.UserRepository) passwordports.PasswordDelegation {
	service := source.PasswordDelegation(users)
	if service == nil {
		return nil
	}
	return passwordDelegation{
		verify: func(ctx context.Context, userID, password string) (passwordports.PasswordVerification, error) {
			got, err := service.Verify(ctx, userID, password)
			return passwordports.PasswordVerification{Delegated: got.Delegated, Accepted: got.Result == "accepted", Unavailable: got.Result == "connector_unavailable" || got.Result == "directory_unavailable"}, err
		},
		write: func(ctx context.Context, userID string, apply func() error) error {
			return service.WithPasswordConfigurationLock(ctx, func() error {
				allowed, err := service.AllowsLocalPassword(ctx, userID)
				if err != nil {
					return err
				}
				if !allowed {
					return passwordports.ErrPasswordDelegated
				}
				return apply()
			})
		},
	}
}

type passwordDelegation struct {
	verify func(context.Context, string, string) (passwordports.PasswordVerification, error)
	write  func(context.Context, string, func() error) error
}

func (d passwordDelegation) VerifyPassword(ctx context.Context, userID, password string) (passwordports.PasswordVerification, error) {
	return d.verify(ctx, userID, password)
}

func (d passwordDelegation) WithLocalPasswordWrite(ctx context.Context, userID string, apply func() error) error {
	return d.write(ctx, userID, apply)
}

type passwordDiagnosticGate struct {
	throttle sessionports.LoginAttemptThrottle
}

func PasswordDiagnosticGate(throttle sessionports.LoginAttemptThrottle) directoryports.PasswordDiagnosticGate {
	if throttle == nil {
		return nil
	}
	return passwordDiagnosticGate{throttle: throttle}
}

func (g passwordDiagnosticGate) Check(ctx context.Context, username, ip string) (int, error) {
	for _, key := range []struct {
		kind  sessionports.LoginThrottleKind
		value string
	}{{sessionports.LoginThrottleAccount, strings.ToLower(username)}, {sessionports.LoginThrottleIP, ip}} {
		if key.value == "" {
			continue
		}
		r, err := g.throttle.TryAcquire(ctx, key.kind, key.value, time.Now().UTC())
		if err != nil {
			return 0, err
		}
		if !r.Allowed {
			return r.RetryAfterSeconds, nil
		}
	}
	return 0, nil
}

func (g passwordDiagnosticGate) Record(ctx context.Context, username, ip string, accepted bool) error {
	now := time.Now().UTC()
	if accepted {
		return g.throttle.RecordSuccess(ctx, sessionports.LoginThrottleAccount, strings.ToLower(username))
	}
	for _, key := range []struct {
		kind  sessionports.LoginThrottleKind
		value string
	}{{sessionports.LoginThrottleAccount, strings.ToLower(username)}, {sessionports.LoginThrottleIP, ip}} {
		if key.value != "" {
			if _, err := g.throttle.RecordFailure(ctx, key.kind, key.value, now); err != nil {
				return err
			}
		}
	}
	return nil
}
