package usecases_test

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/internal/db_memory"
	"github.com/ambi/idmagic/backend/wsfederation/internal/usecases"
)

//spec:covers REQ-WSFEDERATION-002: 別方式の既存セッションは WIA の要求を満たさず、HTTP が Kerberos を試せるよう要求を引き渡す。
func TestWIAWithPasswordSessionRequiresKerberos(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	tenantID := tenantports.TenantID(ctx)
	users := idmanagement.NewMemoryModule().UserRepo
	if err := users.Save(ctx, &userdomain.User{ID: "alice", TenantID: tenantID, PreferredUsername: "alice", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	parties := db_memory.NewWsFedRelyingPartyRepository()
	parties.Seed(&feddomain.WsFedRelyingParty{TenantID: tenantID, Wtrealm: "urn:rp", ReplyURLs: []string{"https://rp.example/wsfed"}})
	service := usecases.SignInService{RPRepo: parties, UserRepo: users}
	outcome, err := service.Issue(ctx, usecases.SignInInput{TenantID: tenantID, Request: feddomain.WsFedSignInRequest{Wa: feddomain.WaSignIn, Wtrealm: "urn:rp", Wauth: "urn:federation:authentication:windows"}, Authn: &authdomain.AuthenticationContext{UserID: "alice", AMR: []string{"pwd"}, AuthTime: time.Now().Unix()}})
	if err != nil || outcome.Kind != usecases.SignInNeedKerberos {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
}
