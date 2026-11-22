// 評価による承認の起票と、起票した承認の参照 (REQ-OAUTH2-093)。
package usecases

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ApprovalRecipients は承認の依頼を送る本人を引く。
type ApprovalRecipients interface {
	FindBySub(ctx context.Context, sub string) (*userdomain.User, error)
}

// StartedApproval は、評価が起票したか参照した保留中の承認リクエストである。承認の URL を利用者へ示すためにだけ返す。
type StartedApproval struct {
	ID        string
	Status    string
	ExpiresAt time.Time
	URI       string
}

// referencesStartedApproval は、評価が承認をトークンではなく承認リクエストの識別子で扱うかを返す。
// トークンが CIBA の承認を持ち、参照もなければ、従来どおりトークンの承認で評価する。
func referencesStartedApproval(in ResourceAccessRequest, token *oauthports.IntrospectionResult) bool {
	return in.ApprovalRequestID != "" || (in.ApprovalStart && token.ApprovalRequestID == "")
}

// startedApprovalOutcome は承認の段の結果である。deny があれば不許可で、なければ matched の使用権を得る。
type startedApprovalOutcome struct {
	matched *matchedApproval
	deny    *ResourceAccessDecision
}

// resolveStartedApproval は参照した承認を照合し、使えなければ起票する。トークンの承認の明細は求めない。
func resolveStartedApproval(
	ctx context.Context, deps ResourceAccessDeps, server *oauthdomain.McpResourceServer, rule oauthdomain.ResourceAccessRule,
	detailType string, token *oauthports.IntrospectionResult, subject accessports.PrincipalRef, in ResourceAccessRequest, now time.Time,
) (startedApprovalOutcome, error) {
	if deps.Approvals == nil {
		return startedApprovalOutcome{}, ErrResourceAccessNotConfigured
	}
	deny := func(reason string) (startedApprovalOutcome, error) {
		decision := denyApproval(server, rule, detailType, in, reason)
		return startedApprovalOutcome{deny: &decision}, nil
	}
	digest, err := approvaldomain.CanonicalizeOperationParameters(in.OperationParameters)
	if err != nil {
		return deny(oauthdomain.ReasonApprovedOperationMismatch)
	}
	binding := approvaldomain.OperationBinding{DetailType: detailType, Location: server.Resource, Action: in.Relation, Identifier: in.ResourceID}
	if in.ApprovalRequestID != "" {
		rec, err := deps.Approvals.FindByID(ctx, in.ApprovalRequestID)
		if err != nil {
			return startedApprovalOutcome{}, err
		}
		if referable(rec, server, token) && !approvaldomain.IsExpired(rec, now) {
			if rec.State == spec.ApprovalDenied {
				return deny(oauthdomain.ReasonApprovalDenied)
			}
			if approvaldomain.MatchApprovedOperation(rec, binding, digest, now) != approvaldomain.ApprovedOperationMatched {
				return deny(oauthdomain.ReasonApprovedOperationMismatch)
			}
			if rec.State == spec.ApprovalPending {
				return pendingApproval(deps, rec)
			}
			keyHash := approvaldomain.HashOperationKey(in.ClientID, in.OperationKey)
			if rec.Use != nil && rec.Use.KeyHash != keyHash {
				return deny(oauthdomain.ReasonApprovalAlreadyUsed)
			}
			return startedApprovalOutcome{matched: &matchedApproval{requestID: rec.ID, digest: digest, keyHash: keyHash}}, nil
		}
	}
	if !in.ApprovalStart {
		return deny(oauthdomain.ReasonApprovalExpired)
	}
	// MCP クライアントの前にいる本人だけが承認者になれる。Agent の主体には承認者を解決する規則がない。
	if subject.Type == oauthdomain.PrincipalTypeAgent {
		return deny(oauthdomain.ReasonApprovalRequired)
	}
	return startApproval(ctx, deps, server, detailType, token, binding, digest, in, now)
}

// referable は、参照した承認が評価した登録の、同じ利用者、クライアント、Agent の評価で起票したものかを返す。
// それ以外は、識別子を知るだけで使わせないため、残っていない承認として扱う。
func referable(rec *approvaldomain.ApprovalRequest, server *oauthdomain.McpResourceServer, token *oauthports.IntrospectionResult) bool {
	return rec != nil && rec.InitiatedByResourceServer() && rec.ResourceServerID != nil && *rec.ResourceServerID == server.ID &&
		rec.UserID == token.Sub && rec.ClientID == token.ClientID && optionalValue(rec.AgentID) == token.AgentID
}

func startApproval(
	ctx context.Context, deps ResourceAccessDeps, server *oauthdomain.McpResourceServer, detailType string,
	token *oauthports.IntrospectionResult, binding approvaldomain.OperationBinding, digest string, in ResourceAccessRequest, now time.Time,
) (startedApprovalOutcome, error) {
	if deps.ApprovalURL == "" {
		return startedApprovalOutcome{}, ErrResourceAccessNotConfigured
	}
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		return startedApprovalOutcome{}, err
	}
	// 明細は要求と登録の写しで、照合 (MatchApprovedOperation) と同じ規則で読める引数だけを fields にする。
	details := approvalRemediation(server, oauthdomain.ResourceAccessRule{}, detailType, in)
	operationDigest := approvaldomain.OperationStartDigest(binding, digest)
	rec := &approvaldomain.ApprovalRequest{
		ID: id, TenantID: tenantports.TenantID(ctx), ClientID: token.ClientID, AgentID: optionalPointer(token.AgentID), UserID: token.Sub,
		AuthorizationDetails: details, State: spec.ApprovalPending,
		Initiator: approvaldomain.InitiatedByResourceServer, ResourceServerID: &server.ID, OperationDigest: &operationDigest,
		RequestedAt: now, ExpiresAt: now.Add(approvaldomain.ResourceServerApprovalTTL),
	}
	if err := rec.Validate(); err != nil {
		return startedApprovalOutcome{}, err
	}
	stored, outcome, err := deps.Approvals.StartForResourceServer(ctx, rec, approvaldomain.MaxPendingResourceServerApprovals, now)
	if err != nil {
		return startedApprovalOutcome{}, err
	}
	switch outcome {
	case approvalports.StartLimited:
		decision := denyResourceAccess(oauthdomain.ReasonApprovalRequestLimitExceeded)
		return startedApprovalOutcome{deny: &decision}, nil
	case approvalports.StartCreated:
		notifyStartedApproval(ctx, deps, stored, approvalURI(deps, stored.ID))
	}
	return pendingApproval(deps, stored)
}

func pendingApproval(deps ResourceAccessDeps, rec *approvaldomain.ApprovalRequest) (startedApprovalOutcome, error) {
	decision := denyResourceAccess(oauthdomain.ReasonApprovalPending)
	decision.StartedApproval = &StartedApproval{ID: rec.ID, Status: "pending", ExpiresAt: rec.ExpiresAt, URI: approvalURI(deps, rec.ID)}
	return startedApprovalOutcome{deny: &decision}, nil
}

// approvalURI は承認リクエストの UUID だけを付ける。開いた者が本人でなければ、ポータルは内容も判断も示さない。
func approvalURI(deps ResourceAccessDeps, id string) string {
	return deps.ApprovalURL + "?request=" + url.QueryEscape(id)
}

// notifyStartedApproval は CIBA と同じテンプレートで本人へ承認の依頼を送る。承認リクエストは保存済みで、
// 本人はポータルの一覧と MCP クライアントの URL からも開けるので、通知の失敗では評価を止めない
// (Notifier も fail-open)。本人を引けなければ送らない。
func notifyStartedApproval(ctx context.Context, deps ResourceAccessDeps, rec *approvaldomain.ApprovalRequest, uri string) {
	if deps.Notifier == nil || deps.Recipients == nil {
		return
	}
	user, err := deps.Recipients.FindBySub(ctx, rec.UserID)
	if err != nil || user == nil || user.Email == nil || strings.TrimSpace(*user.Email) == "" {
		return
	}
	deps.Notifier.Notify(ctx, notificationports.Notification{
		TenantID: user.TenantID, To: *user.Email, Key: notificationports.TemplateKeyAgentActionApprovalRequest,
		RecipientLocale: user.LocaleAttribute(), Vars: map[string]string{
			"approval_url": uri, "client_name": clientDisplayName(ctx, deps, rec.ClientID), "agent_name": optionalOrDash(rec.AgentID),
			"binding_message": "—", "expires_in_minutes": fmt.Sprintf("%d", int(rec.ExpiresAt.Sub(rec.RequestedAt).Minutes())),
			"user_display_name": user.DisplayName(),
		},
	})
}

// clientDisplayName は承認を求めた MCP クライアントの表示名で、引けなければ client_id を使う。
func clientDisplayName(ctx context.Context, deps ResourceAccessDeps, clientID string) string {
	if deps.Clients == nil {
		return clientID
	}
	client, err := deps.Clients.FindByID(ctx, tenantports.TenantID(ctx), clientID)
	if err != nil || client == nil || client.ClientName == nil || strings.TrimSpace(*client.ClientName) == "" {
		return clientID
	}
	return *client.ClientName
}

func optionalPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalOrDash(value *string) string {
	if value == nil || *value == "" {
		return "—"
	}
	return *value
}
