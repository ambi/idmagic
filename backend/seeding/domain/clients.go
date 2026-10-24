package domain

import (
	"reflect"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func FirstPartyClients(seeds []FirstPartyClientSeed, redirectURIs []string, now time.Time) []*oauthdomain.OAuth2Client {
	result := make([]*oauthdomain.OAuth2Client, 0, len(seeds))
	for _, client := range seeds {
		name := client.Name
		result = append(result, &oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: client.ID, ClientName: &name,
			ClientType: spec.ClientPublic, RedirectURIs: redirectURIs,
			GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: oauthdomain.AuthMethodNone,
			Scope: client.Scope, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
			FapiProfile: oauthdomain.FapiNone, FirstParty: true, CreatedAt: now, UpdatedAt: now,
		})
	}
	return result
}

func SameFirstPartyClient(actual, desired *oauthdomain.OAuth2Client) bool {
	left, right := *actual, *desired
	left.ApplicationID, right.ApplicationID = "", ""
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
