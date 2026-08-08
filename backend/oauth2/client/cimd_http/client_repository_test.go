package cimd_http

import (
	"context"
	"errors"
	"testing"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type stubFetcher struct {
	client *clientdomain.OAuth2Client
	err    error
	calls  int
}

func (f *stubFetcher) Fetch(_ context.Context, _ string) (*clientdomain.OAuth2Client, error) {
	f.calls++
	return f.client, f.err
}

func TestClientRepositoryWithCIMD_RegisteredClientTakesPriorityOverFetch(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	repo.Seed(&clientdomain.OAuth2Client{
		TenantID: "default", ClientID: "registered-client",
		GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode},
	})
	fetcher := &stubFetcher{}
	decorated := &ClientRepositoryWithCIMD{OAuth2ClientRepository: repo, Fetcher: fetcher}

	client, err := decorated.FindByID(t.Context(), "default", "registered-client")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil || client.ClientID != "registered-client" {
		t.Fatalf("expected registered client, got %+v", client)
	}
	if fetcher.calls != 0 {
		t.Errorf("fetcher called %d times, want 0 (repo hit should short-circuit)", fetcher.calls)
	}
}

func TestClientRepositoryWithCIMD_NonURLUnknownClientIDReturnsNotFoundWithoutFetch(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	fetcher := &stubFetcher{}
	decorated := &ClientRepositoryWithCIMD{OAuth2ClientRepository: repo, Fetcher: fetcher}

	client, err := decorated.FindByID(t.Context(), "default", "unknown-opaque-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != nil {
		t.Errorf("expected nil client for unknown opaque client_id, got %+v", client)
	}
	if fetcher.calls != 0 {
		t.Errorf("fetcher called %d times, want 0 (not URL-shaped)", fetcher.calls)
	}
}

func TestClientRepositoryWithCIMD_ResolvesViaFetcherOnRepoMiss(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	const url = "https://app.example.com/oauth/client-metadata.json"
	resolved := &clientdomain.OAuth2Client{ClientID: url, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}}
	fetcher := &stubFetcher{client: resolved}
	var emitted []spec.DomainEvent
	decorated := &ClientRepositoryWithCIMD{
		OAuth2ClientRepository: repo, Fetcher: fetcher,
		Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) },
	}

	client, err := decorated.FindByID(t.Context(), "default", url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != resolved {
		t.Fatalf("expected fetcher's resolved client, got %+v", client)
	}
	if fetcher.calls != 1 {
		t.Errorf("fetcher called %d times, want 1", fetcher.calls)
	}
	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(emitted))
	}
	if _, ok := emitted[0].(*clientdomain.ClientIdMetadataDocumentResolved); !ok {
		t.Errorf("expected ClientIdMetadataDocumentResolved, got %T", emitted[0])
	}
}

func TestClientRepositoryWithCIMD_FetchFailureIsTreatedAsNotFound(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	const url = "https://app.example.com/oauth/client-metadata.json"
	fetcher := &stubFetcher{err: errors.New("fetch client id metadata document: status 404")}
	var emitted []spec.DomainEvent
	decorated := &ClientRepositoryWithCIMD{
		OAuth2ClientRepository: repo, Fetcher: fetcher,
		Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) },
	}

	client, err := decorated.FindByID(t.Context(), "default", url)
	if err != nil {
		t.Fatalf("expected fetch failure to surface as not-found (nil, nil), got error: %v", err)
	}
	if client != nil {
		t.Errorf("expected nil client on fetch failure, got %+v", client)
	}
	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(emitted))
	}
	if _, ok := emitted[0].(*clientdomain.ClientIdMetadataDocumentRejected); !ok {
		t.Errorf("expected ClientIdMetadataDocumentRejected, got %T", emitted[0])
	}
}

func TestClientRepositoryWithCIMD_EmbeddedMethodsDelegateUnchanged(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	repo.Seed(&clientdomain.OAuth2Client{TenantID: "default", ClientID: "c1"})
	decorated := &ClientRepositoryWithCIMD{OAuth2ClientRepository: repo, Fetcher: &stubFetcher{}}

	all, err := decorated.FindAll(t.Context(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 || all[0].ClientID != "c1" {
		t.Errorf("FindAll = %+v, want the seeded registered client via passthrough", all)
	}
}
