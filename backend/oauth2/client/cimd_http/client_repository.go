package cimd_http

import (
	"context"
	"time"

	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	"github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// DocumentFetcher is implemented by Fetcher; a seam so
// ClientRepositoryWithCIMD's branching logic can be tested with a stub,
// without real HTTP.
type DocumentFetcher interface {
	Fetch(ctx context.Context, clientIDURL string) (*clientdomain.OAuth2Client, error)
}

// ClientRepositoryWithCIMD wraps a registered-client repository and adds a
// live Client ID Metadata Document resolution fallback for client_ids
// shaped as HTTPS metadata document URLs (ADR-155). It embeds
// ports.OAuth2ClientRepository so every method other than FindByID
// delegates straight through unchanged — CIMD-resolved clients are never
// persisted, so Save/Delete/FindAll/etc. only ever see registered clients.
//
// Wrap the real repository with this once at the composition root; callers
// in authorize.go, push_authorization_request.go, and client_auth.go need
// no changes, since they already only call FindByID.
type ClientRepositoryWithCIMD struct {
	ports.OAuth2ClientRepository
	Fetcher DocumentFetcher
	Emit    func(spec.DomainEvent)
}

func (r *ClientRepositoryWithCIMD) FindByID(ctx context.Context, tenantID, clientID string) (*clientdomain.OAuth2Client, error) {
	client, err := r.OAuth2ClientRepository.FindByID(ctx, tenantID, clientID)
	if err != nil || client != nil {
		return client, err
	}
	if !clientdomain.IsClientIDMetadataDocumentURL(clientID) {
		return nil, nil //nolint:nilnil // not a CIMD URL and not registered: same "unknown client_id" as a plain registry miss
	}
	resolved, fetchErr := r.Fetcher.Fetch(ctx, clientID)
	if fetchErr != nil {
		// Fetch/validation failure collapses into the same "unknown
		// client_id" outcome as a registry miss (fail-closed, and avoids
		// distinguishing "malformed CIMD document" from "not registered"
		// as an oracle — matching client_auth.go's uniform invalid_client).
		r.emit(&clientdomain.ClientIdMetadataDocumentRejected{
			At: time.Now().UTC(), TenantID: tenantID, ClientID: clientID, Reason: fetchErr.Error(),
		})
		return nil, nil //nolint:nilerr,nilnil // fetchErr is deliberately swallowed, see comment above
	}
	r.emit(&clientdomain.ClientIdMetadataDocumentResolved{
		At: time.Now().UTC(), TenantID: tenantID, ClientID: clientID,
	})
	return resolved, nil
}

func (r *ClientRepositoryWithCIMD) emit(event spec.DomainEvent) {
	if r.Emit != nil {
		r.Emit(event)
	}
}
