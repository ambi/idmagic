package ports

import (
	"context"
	"time"

	logoutdomain "github.com/ambi/idmagic/backend/oauth2/internal/logout/domain"
)

type ClientSessionStore interface {
	Upsert(ctx context.Context, session *logoutdomain.ClientSession) error
	ListBySid(ctx context.Context, tenantID, sid string) ([]*logoutdomain.ClientSession, error)
}

type LogoutNotificationStore interface {
	Save(ctx context.Context, notification *logoutdomain.LogoutNotification) error
	FindByID(ctx context.Context, tenantID, id string) (*logoutdomain.LogoutNotification, error)
}

type LogoutTokenInput struct {
	Issuer   string
	Subject  string
	Audience string
	Sid      string
	JTI      string
	IssuedAt time.Time
}

type LogoutTokenSigner interface {
	SignLogoutToken(ctx context.Context, input LogoutTokenInput) (string, error)
}

type BackChannelLogoutClient interface {
	Deliver(ctx context.Context, targetURI, logoutToken string) error
}

type BackChannelLogoutJobParams struct {
	NotificationID string `json:"notification_id"`
	Subject        string `json:"sub"`
	Issuer         string `json:"iss"`
}
