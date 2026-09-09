package domain

import (
	"fmt"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
)

const KindBackChannelLogoutDelivery jobsdomain.JobKind = "backchannel_logout_delivery"

func init() {
	jobsdomain.RegisterKind(KindBackChannelLogoutDelivery, jobsdomain.LaneLatencySensitive)
}

type ClientSession struct {
	TenantID      string
	Sid           string
	ClientID      string
	FirstIssuedAt time.Time
	LastIssuedAt  time.Time
}

type LogoutNotificationState string

const (
	LogoutNotificationPending   LogoutNotificationState = "Pending"
	LogoutNotificationDelivered LogoutNotificationState = "Delivered"
	LogoutNotificationFailed    LogoutNotificationState = "Failed"
)

func (state LogoutNotificationState) Valid() bool {
	switch state {
	case LogoutNotificationPending, LogoutNotificationDelivered, LogoutNotificationFailed:
		return true
	}
	return false
}

type LogoutNotification struct {
	ID             string
	TenantID       string
	Sid            string
	ClientID       string
	LogoutTokenJTI string
	TargetURI      string
	State          LogoutNotificationState
	Attempts       int64
	LastError      *string
	JobID          *string
	CreatedAt      time.Time
	DeliveredAt    *time.Time
}

type LogoutNotificationEvent string

const (
	LogoutNotificationDeliver LogoutNotificationEvent = "Deliver"
	LogoutNotificationExhaust LogoutNotificationEvent = "Exhaust"
)

func TransitionLogoutNotification(from LogoutNotificationState, event LogoutNotificationEvent) (LogoutNotificationState, error) {
	if from != LogoutNotificationPending {
		return "", fmt.Errorf("logout notification: no transition from %q on %q", from, event)
	}
	switch event {
	case LogoutNotificationDeliver:
		return LogoutNotificationDelivered, nil
	case LogoutNotificationExhaust:
		return LogoutNotificationFailed, nil
	default:
		return "", fmt.Errorf("logout notification: no transition from %q on %q", from, event)
	}
}

type FrontChannelLogoutTarget struct {
	ClientID  string
	IframeURI string
}
