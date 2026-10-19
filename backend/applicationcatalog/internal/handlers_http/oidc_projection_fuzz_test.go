package handlers_http

import (
	"reflect"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// FuzzOIDCProjection は、管理表示の射影で任意の値と省略の区別が失われないことを確かめる。
func FuzzOIDCProjection(f *testing.F) {
	f.Add("", "client_secret_basic", "", int64(99), false)
	for _, method := range []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none", "unknown"} {
		for _, empty := range []bool{false, true} {
			f.Add("client", method, "employee_id", int64(100), empty)
		}
	}
	f.Fuzz(func(t *testing.T, id, method, sub string, seconds int64, empty bool) {
		now := time.Unix(100, 0).UTC()
		at := time.Unix(seconds, 0).UTC()
		client := &oauthdomain.OAuth2Client{ClientID: id, ClientType: spec.ClientPublic, TokenEndpointAuthMethod: oauthdomain.TokenEndpointAuthMethod(method), Scope: sub, FapiProfile: oauthdomain.FapiProfile(sub)}
		switch {
		case id == "":
			client.RedirectURIs = nil
			client.GrantTypes = nil
			client.ResponseTypes = nil
		case empty:
			client.RedirectURIs = []string{}
			client.GrantTypes = []spec.GrantType{}
			client.ResponseTypes = []spec.ResponseType{}
		default:
			client.RedirectURIs = []string{id}
			client.GrantTypes = []spec.GrantType{spec.GrantType(sub)}
			client.ResponseTypes = []spec.ResponseType{spec.ResponseType(sub)}
		}
		client.ClaimPolicy = &claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{SourceAttribute: sub}, Rules: []claimdomain.ClaimMappingRule{{ClaimType: id, SourceKey: sub, FixedValue: method, Required: true}}}
		credentials := []oauthdomain.ClientSecretCredential{
			{ID: id, CreatedAt: at}, {ID: sub, CreatedAt: at, ExpiresAt: &at}, {ID: method, CreatedAt: at, ExpiresAt: &at, RevokedAt: &at},
		}
		got := configuration(client, credentials, now)
		wantSub := sub
		if sub == "" {
			wantSub = "user_id"
		}
		if got.ClientID != id || got.ClientType != client.ClientType || got.TokenEndpointAuthMethod != method || got.Scope != sub || got.FapiProfile != sub || got.SubSourceAttribute != wantSub || !reflect.DeepEqual(got.RedirectURIs, client.RedirectURIs) || !reflect.DeepEqual(got.GrantTypes, client.GrantTypes) || !reflect.DeepEqual(got.ResponseTypes, client.ResponseTypes) || !reflect.DeepEqual(got.Rules, client.ClaimPolicy.Rules) {
			t.Fatalf("projection=%+v input=%+v", got, client)
		}
		if got.ClientSecretRotatable != (method == "client_secret_basic" || method == "client_secret_post") {
			t.Fatalf("rotatable=%v method=%q", got.ClientSecretRotatable, method)
		}
		for i, credential := range credentials {
			metadata := got.SecretCredentials[i]
			if metadata.CredentialID != credential.ID || metadata.CreatedAt != at || !reflect.DeepEqual(metadata.ExpiresAt, credential.ExpiresAt) || !reflect.DeepEqual(metadata.RevokedAt, credential.RevokedAt) {
				t.Fatalf("metadata=%+v credential=%+v", metadata, credential)
			}
		}
		wantExpiry := "Active"
		if seconds <= 100 {
			wantExpiry = "Expired"
		}
		if got.SecretCredentials[0].Status != "Active" || got.SecretCredentials[1].Status != wantExpiry || got.SecretCredentials[2].Status != "Revoked" {
			t.Fatalf("statuses=%+v", got.SecretCredentials)
		}
		got.Rules[0].FixedValue = "changed"
		if client.ClaimPolicy.Rules[0].FixedValue != method {
			t.Fatal("projection aliased the saved rules")
		}
	})
}
