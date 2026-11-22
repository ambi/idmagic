package usecases

import (
	"context"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type OAuthError = oauthusecases.OAuthError

var (
	NewOAuthError                = oauthusecases.NewOAuthError
	errorCode                    = oauthusecases.ErrorCode
	emit                         = oauthusecases.Emit
	generateOpaqueToken          = oauthusecases.GenerateOpaqueToken
	ParseAuthorizationDetails    = oauthusecases.ParseAuthorizationDetails
	ValidateAuthorizationDetails = oauthusecases.ValidateAuthorizationDetails
	ResolveResourceIndicator     = oauthusecases.ResolveResourceIndicator
)

// detailsRequest は、登録済みのクライアントが送った authorization_details である。
type detailsRequest struct {
	ClientID string
	Raw      string
	Now      time.Time
}

// acceptAuthorizationDetails は authorization_details を解析し、テナントに登録した種類に対して検証する。
// 拒否したときは AuthorizationDetailsRejected を発行する。種類の保管先の障害は拒否ではないので発行しない。
func acceptAuthorizationDetails(
	ctx context.Context,
	repo oauthports.AuthorizationDetailTypeRepository,
	emitEvent func(spec.DomainEvent),
	req detailsRequest,
) ([]spec.AuthorizationDetail, error) {
	details, err := ParseAuthorizationDetails(req.Raw)
	if err == nil {
		err = ValidateAuthorizationDetails(ctx, repo, details)
	}
	if err == nil {
		return details, nil
	}
	if code := errorCode(err); code == "invalid_authorization_details" {
		emit(emitEvent, &oauthdomain.AuthorizationDetailsRejected{
			At: req.Now.UTC(), TenantID: tenantports.TenantID(ctx), ClientID: req.ClientID, Reason: code,
		})
	}
	return nil, err
}
