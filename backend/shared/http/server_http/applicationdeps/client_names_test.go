package applicationdeps

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

type faultyClientNames struct {
	clientports.OAuth2ClientRepository
	err error
}

func (r faultyClientNames) FindByID(context.Context, string, string) (*oauthdomain.OAuth2Client, error) {
	return nil, r.err
}

func TestClientNamesPreserveMissingValuesAndErrors(t *testing.T) {
	ctx := context.Background()
	repo := oauth2.NewMemoryModule().ClientRepo
	if NewClientNames(nil) != nil {
		t.Fatal("nil repository must leave the optional name query unwired")
	}
	if name, err := NewClientNames(repo).FindClientName(ctx, "tenant", "missing"); name != nil || err != nil {
		t.Fatalf("missing name=%v err=%v", name, err)
	}
	failure := errors.New("read failed")
	if _, err := NewClientNames(faultyClientNames{err: failure}).FindClientName(ctx, "tenant", "client"); !errors.Is(err, failure) {
		t.Fatalf("name error=%v", err)
	}
	if err := repo.Save(ctx, &oauthdomain.OAuth2Client{TenantID: "tenant", ClientID: "client"}); err != nil {
		t.Fatal(err)
	}
	if name, err := NewClientNames(repo).FindClientName(ctx, "tenant", "client"); name != nil || err != nil {
		t.Fatalf("unset name=%v err=%v", name, err)
	}
	name := "Payroll"
	if err := repo.Save(ctx, &oauthdomain.OAuth2Client{TenantID: "tenant", ClientID: "client", ClientName: &name}); err != nil {
		t.Fatal(err)
	}
	gotName, err := NewClientNames(repo).FindClientName(ctx, "tenant", "client")
	if err != nil || gotName == nil || *gotName != name {
		t.Fatalf("name=%v err=%v", gotName, err)
	}
	*gotName = "changed"
	stored, _ := repo.FindByID(ctx, "tenant", "client")
	if *stored.ClientName != name {
		t.Fatal("query result aliased the stored name")
	}
}
