package spec

import "fmt"

// ApprovalRequestLifecycle mirrors the normative transition table in
// spec/contexts/oauth2/states.md. Only Approved can reach Consumed, so a satisfied request
// can be converted to tokens exactly once.

type ApprovalRequestEvent string

const (
	ApprovalEventApprove ApprovalRequestEvent = "Approve"
	ApprovalEventDeny    ApprovalRequestEvent = "Deny"
	ApprovalEventConsume ApprovalRequestEvent = "Consume"
	ApprovalEventExpire  ApprovalRequestEvent = "Expire"
)

type approvalRequestTransition struct {
	From  ApprovalRequestState
	Event ApprovalRequestEvent
	To    ApprovalRequestState
}

var approvalRequestTransitions = []approvalRequestTransition{
	{ApprovalPending, ApprovalEventApprove, ApprovalApproved},
	{ApprovalPending, ApprovalEventDeny, ApprovalDenied},
	{ApprovalPending, ApprovalEventExpire, ApprovalExpired},
	{ApprovalApproved, ApprovalEventConsume, ApprovalConsumed},
	{ApprovalApproved, ApprovalEventExpire, ApprovalExpired},
}

func TransitionApprovalRequest(from ApprovalRequestState, event ApprovalRequestEvent) (ApprovalRequestState, error) {
	for _, t := range approvalRequestTransitions {
		if t.From == from && t.Event == event {
			return t.To, nil
		}
	}
	return "", fmt.Errorf("no transition from %q on event %q", from, event)
}

func IsApprovalRequestTerminal(s ApprovalRequestState) bool {
	switch s {
	case ApprovalDenied, ApprovalExpired, ApprovalConsumed:
		return true
	}
	return false
}
