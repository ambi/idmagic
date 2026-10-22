package handlers_http_test

import (
	"context"
	"net/http"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// withUniqueBadge は、本人が編集できる unique の属性 badge を定義し、Bob に値を持たせる。
func withUniqueBadge(t *testing.T, f *idmRefusalFixture) {
	t.Helper()
	ctx := context.Background()
	if err := f.schemas.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{{
			Key: "badge", Type: idmdomain.AttributeTypeString, Unique: true, EditableByUser: true,
			Visibility: idmdomain.AttrVisibilitySelfReadable,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	bob, err := f.users.FindBySub(ctx, idmRefusalBob)
	if err != nil || bob == nil {
		t.Fatalf("bob = (%+v, %v)", bob, err)
	}
	badge := "B-1"
	bob.Attributes = map[string]userdomain.AttributeValue{"badge": {Type: idmdomain.AttributeTypeString, String: &badge}}
	if err := f.users.Save(ctx, bob); err != nil {
		t.Fatal(err)
	}
}

func badgeOf(t *testing.T, f *idmRefusalFixture, id string) *string {
	t.Helper()
	u, err := f.users.FindBySub(context.Background(), id)
	if err != nil || u == nil {
		t.Fatalf("FindBySub(%s) = (%+v, %v)", id, u, err)
	}
	return u.Attributes["badge"].String
}

//spec:covers REQ-IDMANAGEMENT-090: 管理者が unique の属性をほかの User の値へ変える更新を 409 と attribute_value_conflict で拒否し、User を変えないこと。
//spec:covers REQ-IDMANAGEMENT-051: 本人が unique の属性をほかの User の値へ変える更新を 409 と attribute_value_conflict で拒否し、どの項目も変えないこと。
//verify:operation UpdateAdminUser: unique の属性の重なりへの 409 attribute_value_conflict の Problem Details と、保存されていないこと
//verify:operation UpdateUserProfile: unique の属性の重なりへの 409 attribute_value_conflict の Problem Details と、保存されていないこと
func TestUniqueAttributeConflictHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	withUniqueBadge(t, f)
	taken := map[string]any{"badge": map[string]any{"type": "string", "string": "B-1"}}

	admin := f.seedSession(t, "unique-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin, withFreshStepUp)
	byAdmin := f.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/admin/v1/users/" + idmRefusalAlice,
		body: map[string]any{"attributes": taken}, sessionID: admin, csrf: idmRefusalCSRF,
	})
	idmContractProblem(t, byAdmin, http.StatusConflict, "attribute_value_conflict")
	if got := badgeOf(t, f, idmRefusalAlice); got != nil {
		t.Fatalf("refused admin update stored badge %q", *got)
	}

	write := f.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountWrite)
	bySelf := f.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: write,
		body: map[string]any{"name": "Alice Impostor", "attributes": taken},
	})
	idmContractProblem(t, bySelf, http.StatusConflict, "attribute_value_conflict")
	alice, err := f.users.FindBySub(context.Background(), idmRefusalAlice)
	if err != nil || alice == nil || alice.Attributes["badge"].String != nil || (alice.Name != nil && *alice.Name == "Alice Impostor") {
		t.Fatalf("refused self-service update changed alice: (%+v, %v)", alice, err)
	}
	if got := badgeOf(t, f, idmRefusalBob); got == nil || *got != "B-1" {
		t.Fatalf("the holder lost its badge: %v", got)
	}
}
