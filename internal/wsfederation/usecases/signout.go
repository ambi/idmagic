package usecases

import (
	"context"
	"slices"

	wsfedports "github.com/ambi/idmagic/internal/wsfederation/ports"

	feddomain "github.com/ambi/idmagic/internal/wsfederation/domain"
)

// SignOutService は WS-Federation sign-out の返送先解決を所有する。
type SignOutService struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

// ResolveReply は wreply を wtrealm で解決した RP の許可集合に対して検証する。
// 検証を通らない (または wtrealm/wreply 不在) なら空文字を返し、リダイレクトしない
// (open redirect 防止)。エラーは「リダイレクトしない」判断へ畳む。
func (s SignOutService) ResolveReply(ctx context.Context, tenantID string, req feddomain.WsFedSignInRequest) string {
	if req.Wtrealm == "" || req.Wreply == "" {
		return ""
	}
	rp, err := s.RPRepo.FindByWtrealm(ctx, tenantID, req.Wtrealm)
	if err != nil || rp == nil {
		return ""
	}
	if slices.Contains(rp.ReplyURLs, req.Wreply) {
		return req.Wreply
	}
	return ""
}
