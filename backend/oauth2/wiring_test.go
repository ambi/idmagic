package oauth2

import (
	"context"
	"testing"
	"time"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type recordingIssuer struct {
	oauthports.TokenIssuer
	signedAt []time.Time
}

func (r *recordingIssuer) SignAccessToken(context.Context, oauthports.AccessTokenInput) (string, string, error) {
	r.signedAt = append(r.signedAt, time.Now())
	return "token", "jti", nil
}

type currentSecondEpochs struct{ epoch time.Time }

func (e currentSecondEpochs) FindByUser(_ context.Context, tenantID, userID string) (*ssdomain.UserRevocationEpoch, error) {
	return &ssdomain.UserRevocationEpoch{TenantID: tenantID, UserID: userID, Epoch: e.epoch}, nil
}

func (currentSecondEpochs) Advance(context.Context, ssdomain.UserRevocationEpoch) error { return nil }

// グラントのユースケースが受け取る発行器が、User の失効と同じ秒の発行を次の秒へずらすことを、
// 正式な組み立てから確かめる。装飾を組み立てで付け忘れると、単体テストは通ったまま、
// 失効の直後の再認証だけが失効と判定される。
//
//spec:covers REQ-SHAREDSIGNALS-012: トークンエンドポイントの発行器が、失効と同じ秒の発行を次の秒まで待たせること
func TestHTTPDepsIssuerWaitsForTheSecondAfterUserRevocation(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	epoch := time.Now()
	signer := &recordingIssuer{}
	deps := Module{TokenIssuer: signer}.httpDeps(RouteDeps{UserRevocationEpochRepo: currentSecondEpochs{epoch: epoch}})

	if _, _, err := deps.TokenIssuer.SignAccessToken(ctx, oauthports.AccessTokenInput{Sub: "user-1"}); err != nil {
		t.Fatal(err)
	}
	if len(signer.signedAt) != 1 || signer.signedAt[0].Unix() <= epoch.Unix() {
		t.Fatalf("signed at %v, want a second after the epoch %v", signer.signedAt, epoch)
	}
}
