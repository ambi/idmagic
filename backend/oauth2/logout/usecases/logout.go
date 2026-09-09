package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/logout/domain"
	"github.com/ambi/idmagic/backend/oauth2/logout/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type EnqueueFunc func(context.Context, jobsports.EnqueueInput, time.Time) (*jobsdomain.Job, error)

type StartBackChannelLogoutDeps struct {
	ClientSessions ports.ClientSessionStore
	Clients        clientports.OAuth2ClientRepository
	Notifications  ports.LogoutNotificationStore
	Enqueue        EnqueueFunc
	NewID          func() (string, error)
}

func StartBackChannelLogout(ctx context.Context, deps StartBackChannelLogoutDeps, sid, subject, issuer string, now time.Time) ([]*domain.LogoutNotification, error) {
	if deps.ClientSessions == nil || deps.Clients == nil || deps.Notifications == nil || deps.Enqueue == nil {
		return nil, errors.New("logout: incomplete back-channel dependencies")
	}
	newID := deps.NewID
	if newID == nil {
		newID = spec.NewUUIDv4
	}
	tenantID := tenancy.TenantID(ctx)
	sessions, err := deps.ClientSessions.ListBySid(ctx, tenantID, sid)
	if err != nil {
		return nil, fmt.Errorf("logout: list client sessions: %w", err)
	}
	created := make([]*domain.LogoutNotification, 0, len(sessions))
	for _, session := range sessions {
		client, findErr := deps.Clients.FindByID(ctx, tenantID, session.ClientID)
		if findErr != nil {
			return nil, fmt.Errorf("logout: find client %q: %w", session.ClientID, findErr)
		}
		if client == nil || client.BackChannelLogoutURI == nil {
			continue
		}
		notificationID, idErr := newID()
		if idErr != nil {
			return nil, fmt.Errorf("logout: create notification id: %w", idErr)
		}
		jti, idErr := newID()
		if idErr != nil {
			return nil, fmt.Errorf("logout: create logout token jti: %w", idErr)
		}
		notification := &domain.LogoutNotification{
			ID: notificationID, TenantID: tenantID, Sid: sid, ClientID: client.ClientID,
			LogoutTokenJTI: jti, TargetURI: *client.BackChannelLogoutURI,
			State: domain.LogoutNotificationPending, CreatedAt: now,
		}
		if saveErr := deps.Notifications.Save(ctx, notification); saveErr != nil {
			return nil, fmt.Errorf("logout: save notification: %w", saveErr)
		}
		params, marshalErr := json.Marshal(ports.BackChannelLogoutJobParams{NotificationID: notification.ID, Subject: subject, Issuer: issuer})
		if marshalErr != nil {
			return nil, fmt.Errorf("logout: encode job params: %w", marshalErr)
		}
		dedupKey := "logout-notification:" + notification.ID
		job, enqueueErr := deps.Enqueue(ctx, jobsports.EnqueueInput{TenantID: tenantID, Kind: domain.KindBackChannelLogoutDelivery, Params: params, DedupKey: &dedupKey}, now)
		if enqueueErr != nil {
			return nil, fmt.Errorf("logout: enqueue delivery: %w", enqueueErr)
		}
		notification.JobID = &job.ID
		if saveErr := deps.Notifications.Save(ctx, notification); saveErr != nil {
			return nil, fmt.Errorf("logout: save notification job: %w", saveErr)
		}
		created = append(created, notification)
	}
	return created, nil
}

type FrontChannelLogoutDeps struct {
	ClientSessions ports.ClientSessionStore
	Clients        clientports.OAuth2ClientRepository
}

func FrontChannelLogoutTargets(ctx context.Context, deps FrontChannelLogoutDeps, sid, issuer string) ([]domain.FrontChannelLogoutTarget, error) {
	if deps.ClientSessions == nil || deps.Clients == nil {
		return nil, errors.New("logout: incomplete front-channel dependencies")
	}
	tenantID := tenancy.TenantID(ctx)
	sessions, err := deps.ClientSessions.ListBySid(ctx, tenantID, sid)
	if err != nil {
		return nil, fmt.Errorf("logout: list client sessions: %w", err)
	}
	targets := make([]domain.FrontChannelLogoutTarget, 0, len(sessions))
	for _, session := range sessions {
		client, findErr := deps.Clients.FindByID(ctx, tenantID, session.ClientID)
		if findErr != nil {
			return nil, fmt.Errorf("logout: find client %q: %w", session.ClientID, findErr)
		}
		if client == nil || client.FrontChannelLogoutURI == nil {
			continue
		}
		iframeURI := *client.FrontChannelLogoutURI
		if client.FrontChannelLogoutSessionRequired {
			u, parseErr := url.Parse(iframeURI)
			if parseErr != nil {
				return nil, fmt.Errorf("logout: parse front-channel URI: %w", parseErr)
			}
			query := u.Query()
			query.Set("iss", issuer)
			query.Set("sid", sid)
			u.RawQuery = query.Encode()
			iframeURI = u.String()
		}
		targets = append(targets, domain.FrontChannelLogoutTarget{ClientID: client.ClientID, IframeURI: iframeURI})
	}
	return targets, nil
}

type BackChannelLogoutHandlerDeps struct {
	Notifications ports.LogoutNotificationStore
	Signer        ports.LogoutTokenSigner
	Client        ports.BackChannelLogoutClient
	Now           func() time.Time
}

func BackChannelLogoutHandler(deps BackChannelLogoutHandlerDeps) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		if deps.Notifications == nil || deps.Signer == nil || deps.Client == nil {
			return nil, errors.New("logout: incomplete delivery dependencies")
		}
		var params ports.BackChannelLogoutJobParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, fmt.Errorf("logout: decode job params: %w", err)
		}
		notification, err := deps.Notifications.FindByID(ctx, job.TenantID, params.NotificationID)
		if err != nil {
			return nil, fmt.Errorf("logout: find notification: %w", err)
		}
		if notification == nil {
			return nil, errors.New("logout: notification not found")
		}
		if notification.State != domain.LogoutNotificationPending {
			return json.RawMessage("{}"), nil
		}
		now := time.Now().UTC()
		if deps.Now != nil {
			now = deps.Now().UTC()
		}
		token, deliveryErr := deps.Signer.SignLogoutToken(ctx, ports.LogoutTokenInput{Issuer: params.Issuer, Subject: params.Subject, Audience: notification.ClientID, Sid: notification.Sid, JTI: notification.LogoutTokenJTI, IssuedAt: now})
		if deliveryErr == nil {
			deliveryErr = deps.Client.Deliver(ctx, notification.TargetURI, token)
		}
		notification.Attempts = int64(job.Attempts)
		if deliveryErr != nil {
			message := deliveryErr.Error()
			notification.LastError = &message
			if job.Attempts >= job.MaxAttempts {
				notification.State, err = domain.TransitionLogoutNotification(notification.State, domain.LogoutNotificationExhaust)
				if err != nil {
					return nil, err
				}
			}
			if err := deps.Notifications.Save(ctx, notification); err != nil {
				return nil, fmt.Errorf("logout: save failed notification: %w", err)
			}
			return nil, deliveryErr
		}
		notification.State, err = domain.TransitionLogoutNotification(notification.State, domain.LogoutNotificationDeliver)
		if err != nil {
			return nil, err
		}
		notification.LastError = nil
		notification.DeliveredAt = &now
		if err := deps.Notifications.Save(ctx, notification); err != nil {
			return nil, fmt.Errorf("logout: save delivered notification: %w", err)
		}
		return json.RawMessage("{}"), nil
	}
}
