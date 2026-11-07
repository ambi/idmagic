package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/ports"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// IssuanceClock は、発行器の装飾が読む時刻と待機である。テストでは差し替える。
type IssuanceClock struct {
	Now   func() time.Time
	Sleep func(context.Context, time.Duration) error
}

// SystemIssuanceClock は実時間の時刻と、context の取り消しで打ち切る待機を返す。
func SystemIssuanceClock() IssuanceClock {
	return IssuanceClock{
		Now: time.Now,
		Sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

// IssuerAfterUserRevocation は、サブジェクトの User の失効エポックと同じ秒には、
// 次の秒に入るまで待ってから署名する発行器を返す (REQ-SHAREDSIGNALS-012)。
//
// アクセストークンの iat は秒しか持たないので、エポックと同じ秒の発行は、失効の前か後かを
// 判定の側で見分けられない。失効の後の発行を次の秒へずらせば、失効の前のトークンを止めたまま、
// 資格情報の変更の直後に正常な流れで起きる再認証のトークンを通せる。グラントごとではなく
// 発行器で行うのは、ガードレールと同じく、グラントを足すたびに書き写さずに済ませるためである。
func IssuerAfterUserRevocation(issuer ports.TokenIssuer, epochs ssports.UserRevocationEpochRepository, clock IssuanceClock) ports.TokenIssuer {
	if epochs == nil {
		return issuer
	}
	return userRevocationIssuer{TokenIssuer: issuer, epochs: epochs, clock: clock}
}

type userRevocationIssuer struct {
	ports.TokenIssuer
	epochs ssports.UserRevocationEpochRepository
	clock  IssuanceClock
}

func (i userRevocationIssuer) SignAccessToken(ctx context.Context, in ports.AccessTokenInput) (string, string, error) {
	if in.Sub != "" {
		// クライアントがサブジェクトの発行（クライアント認証情報）はエポックの行を持たない。
		epoch, err := i.epochs.FindByUser(ctx, tenantports.TenantID(ctx), in.Sub)
		if err != nil {
			return "", "", err
		}
		if epoch != nil {
			nextSecond := epoch.Epoch.Truncate(time.Second).Add(time.Second)
			if wait := nextSecond.Sub(i.clock.Now()); wait > 0 {
				if err := i.clock.Sleep(ctx, wait); err != nil {
					return "", "", err
				}
			}
		}
	}
	return i.TokenIssuer.SignAccessToken(ctx, in)
}
