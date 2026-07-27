package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
)

type protocolDriverStub struct {
	claims      federationdomain.NormalizedClaims
	startErr    error
	completeErr error
}

func (d protocolDriverStub) Start(
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	return "https://idp.example/authorize", d.startErr
}

func (d protocolDriverStub) Complete(
	_ context.Context,
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ string,
	_ time.Time,
) (federationdomain.NormalizedClaims, error) {
	return d.claims, d.completeErr
}

func TestStartAndCompleteFlowConsumesStateBeforeProtocolValidation(t *testing.T) {
	deps, connection, _, repos := brokerFixture(t)
	deps.Connections = repos.Connections
	deps.Attempts = repos.Attempts
	deps.Drivers = map[federationdomain.Protocol]federationusecases.ProtocolDriver{
		federationdomain.ProtocolOIDC: protocolDriverStub{completeErr: errors.New("invalid token")},
	}
	if err := repos.Connections.Save(context.Background(), &connection); err != nil {
		t.Fatal(err)
	}
	start, err := federationusecases.StartLogin(
		context.Background(), deps, connection.ID, "", "", "https://broker.example/callback", time.Now(),
	)
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	if start.RedirectTo == "" || start.State == "" {
		t.Fatalf("start=%+v", start)
	}
	if _, err := federationusecases.CompleteLogin(
		context.Background(), deps, start.State, "response", "https://broker.example/callback", time.Now(),
	); err == nil {
		t.Fatal("invalid protocol response must fail")
	}
	if _, err := federationusecases.CompleteLogin(
		context.Background(), deps, start.State, "response", "https://broker.example/callback", time.Now(),
	); err == nil {
		t.Fatal("consumed state must not be reusable")
	}
}
