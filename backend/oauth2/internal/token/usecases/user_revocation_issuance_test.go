package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// fakeClock は、待機の分だけ時刻を進める。
type fakeClock struct{ now time.Time }

func (c *fakeClock) clock() IssuanceClock {
	return IssuanceClock{
		Now: func() time.Time { return c.now },
		Sleep: func(ctx context.Context, d time.Duration) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			c.now = c.now.Add(d)
			return nil
		},
	}
}

// iatIssuer は、署名器と同じく署名の時点の時刻の秒を iat に使い、その値を記録する。
type iatIssuer struct {
	ports.TokenIssuer
	now  func() time.Time
	iats []int64
}

func (i *iatIssuer) SignAccessToken(context.Context, ports.AccessTokenInput) (string, string, error) {
	i.iats = append(i.iats, i.now().Unix())
	return "token", "jti", nil
}

type failingUserEpochs struct{ stubUserEpochs }

func (failingUserEpochs) FindByUser(context.Context, string, string) (*ssdomain.UserRevocationEpoch, error) {
	return nil, errors.New("epoch store unavailable")
}

//spec:covers REQ-SHAREDSIGNALS-012: 待つのはエポックと同じ秒の発行だけで、待ちの取り消しとエポックの読み取りの失敗では署名しないこと
func TestIssuerAfterUserRevocationWaitsOnlyWithinTheEpochSecond(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	epoch := time.Date(2026, 11, 7, 6, 22, 53, 258_000_000, time.UTC)
	epochs := stubUserEpochs{epoch: &ssdomain.UserRevocationEpoch{TenantID: kernel.DefaultTenantID, UserID: idJagUser, Epoch: epoch}}

	for name, tc := range map[string]struct {
		now     time.Time
		epochs  ssports.UserRevocationEpochRepository
		wantIat int64
	}{
		"同じ秒は次の秒まで待つ":    {epoch.Add(42 * time.Millisecond), epochs, epoch.Unix() + 1},
		"次の秒以降は待たない":     {epoch.Add(2 * time.Second), epochs, epoch.Unix() + 2},
		"エポックのない主体は待たない": {epoch.Add(42 * time.Millisecond), stubUserEpochs{}, epoch.Unix()},
	} {
		t.Run(name, func(t *testing.T) {
			clock := &fakeClock{now: tc.now}
			signer := &iatIssuer{now: func() time.Time { return clock.now }}
			if _, _, err := IssuerAfterUserRevocation(signer, tc.epochs, clock.clock()).SignAccessToken(ctx, ports.AccessTokenInput{Sub: idJagUser}); err != nil {
				t.Fatal(err)
			}
			if len(signer.iats) != 1 || signer.iats[0] != tc.wantIat {
				t.Fatalf("iats=%v, want [%d]", signer.iats, tc.wantIat)
			}
		})
	}

	t.Run("待ちを取り消すと署名しない", func(t *testing.T) {
		clock := &fakeClock{now: epoch.Add(42 * time.Millisecond)}
		signer := &iatIssuer{now: func() time.Time { return clock.now }}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, _, err := IssuerAfterUserRevocation(signer, epochs, clock.clock()).SignAccessToken(canceled, ports.AccessTokenInput{Sub: idJagUser}); !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v, want context.Canceled", err)
		}
		if len(signer.iats) != 0 {
			t.Fatalf("署名した: %v", signer.iats)
		}
	})

	t.Run("エポックを読めなければ署名しない", func(t *testing.T) {
		clock := &fakeClock{now: epoch}
		signer := &iatIssuer{now: func() time.Time { return clock.now }}
		if _, _, err := IssuerAfterUserRevocation(signer, failingUserEpochs{}, clock.clock()).SignAccessToken(ctx, ports.AccessTokenInput{Sub: idJagUser}); err == nil {
			t.Fatal("エポックの読み取りの失敗を返さなかった")
		}
		if len(signer.iats) != 0 {
			t.Fatalf("署名した: %v", signer.iats)
		}
	})
}

// 失効と同じ秒の中で、失効の前に発行したトークンと、失効の後にログインし直して発行した
// トークンは、iat が同じ秒になって見分けられない。後者を次の秒へずらして発行すれば、
// 前者を失効させたまま後者を通せる。
//
//spec:covers REQ-SHAREDSIGNALS-012: 失効と同じ秒の発行を次の秒まで待たせ、失効の前のトークンを止めたまま、失効の後のトークンを有効と判定すること
func TestTokenIssuedRightAfterUserRevocationStaysActive(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	users := idmanagement.NewMemoryModule().UserRepo
	if err := users.Save(ctx, &userdomain.User{
		ID: idJagUser, TenantID: kernel.DefaultTenantID, PreferredUsername: "user",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
	}); err != nil {
		t.Fatal(err)
	}
	// 認証器のリセットは秒より細かい時刻でエポックを進める。
	epoch := time.Date(2026, 11, 7, 6, 22, 53, 258_000_000, time.UTC)
	epochs := stubUserEpochs{epoch: &ssdomain.UserRevocationEpoch{TenantID: kernel.DefaultTenantID, UserID: idJagUser, Epoch: epoch}}
	deps := IntrospectDeps{Clients: oauth2memory.NewClientRepository(), Users: users, UserEpochs: epochs}

	// 失効の前に同じ秒で発行したトークン。
	before := epoch.Add(-158 * time.Millisecond).Unix()
	// 失効の 42 ms 後にログインし直して発行するトークン。
	clock := &fakeClock{now: epoch.Add(42 * time.Millisecond)}
	signer := &iatIssuer{now: func() time.Time { return clock.now }}
	issuer := IssuerAfterUserRevocation(signer, epochs, clock.clock())
	if _, _, err := issuer.SignAccessToken(ctx, ports.AccessTokenInput{Sub: idJagUser}); err != nil {
		t.Fatal(err)
	}
	after := signer.iats[0]

	for name, tc := range map[string]struct {
		iat     int64
		revoked bool
	}{
		"失効の前に発行した": {before, true},
		"失効の後に発行した": {after, false},
	} {
		t.Run(name, func(t *testing.T) {
			revoked, err := AccessTokenIsRevoked(ctx, deps, &ports.IntrospectionResult{Active: true, Sub: idJagUser, Iat: tc.iat})
			if err != nil {
				t.Fatal(err)
			}
			if revoked != tc.revoked {
				t.Fatalf("iat=%d revoked=%v, want %v", tc.iat, revoked, tc.revoked)
			}
		})
	}
}
