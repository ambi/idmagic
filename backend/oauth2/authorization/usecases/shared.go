package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type OAuthError = sharedusecases.OAuthError

var (
	NewOAuthError                = sharedusecases.NewOAuthError
	errorCode                    = sharedusecases.ErrorCode
	emit                         = sharedusecases.Emit
	generateOpaqueToken          = sharedusecases.GenerateOpaqueToken
	ParseAuthorizationDetails    = sharedusecases.ParseAuthorizationDetails
	ValidateAuthorizationDetails = sharedusecases.ValidateAuthorizationDetails
	ResolveResourceIndicator     = sharedusecases.ResolveResourceIndicator
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
	repo ports.AuthorizationDetailTypeRepository,
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
		emit(emitEvent, &domain.AuthorizationDetailsRejected{
			At: req.Now.UTC(), TenantID: tenancy.TenantID(ctx), ClientID: req.ClientID, Reason: code,
		})
	}
	return nil, err
}
