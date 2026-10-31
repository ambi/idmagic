package keys_memory

import (
	"context"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-SIGNINGKEYS-013: ID-JAG の鍵が RS256 を宣言し、OAuth2 の鍵の候補と公開の集合に混ざらないこと
func TestIDJAGSigningKeyIsSeparate_REQ_SIGNINGKEYS_013(t *testing.T) {
	store, err := NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := testing_tenant.Default(context.Background())
	idJag := signingports.WithKeyUsage(ctx, signingdomain.KeyUsageIdJagSigning)
	key, err := store.GetActiveKey(idJag)
	if err != nil {
		t.Fatal(err)
	}
	if key.Alg != "RS256" || key.PublicJWK["alg"] != "RS256" {
		t.Fatalf("ID-JAG key alg = %s, JWK alg = %v, want RS256", key.Alg, key.PublicJWK["alg"])
	}
	signing, err := store.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if signing.Alg != "PS256" || signing.Kid == key.Kid {
		t.Fatalf("signing key alg = %s kid = %s, ID-JAG kid = %s", signing.Alg, signing.Kid, key.Kid)
	}
	all, err := store.GetAllKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range all {
		if candidate.Kid == key.Kid {
			t.Fatal("the ID-JAG key is a verification candidate for OAuth2 tokens")
		}
	}
	public, err := store.ListPublicKeys(idJag, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 1 || public[0].Kid != key.Kid {
		t.Fatalf("published ID-JAG keys = %+v", public)
	}
}
