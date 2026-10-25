package testing_contract

import (
	"context"
	"slices"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// AttributeWriteAuthorityRoundTrip は、両方の保存先に同じ分類の保存契約を適用する。
func AttributeWriteAuthorityRoundTrip(t *testing.T, repo userports.TenantUserAttributeSchemaRepository, tenantID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	schema := &userdomain.TenantUserAttributeSchema{
		TenantID: tenantID, CreatedAt: now, UpdatedAt: now,
		Attributes: []userdomain.UserAttributeDef{
			{Key: "approval", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable, WriteAuthority: userdomain.AttributeAuthorityLocal},
			{Key: "legacy", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable},
		},
	}
	if err := repo.Save(ctx, schema); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.FindByTenant(ctx, tenantID)
	if err != nil || stored == nil {
		t.Fatalf("reloaded schema=%+v err=%v", stored, err)
	}
	if got := userdomain.UpstreamWritableAttributes([]string{"approval", "legacy", "unknown"}, stored.EffectiveDefs()); !slices.Equal(got, []string{"legacy", "unknown"}) {
		t.Fatalf("persisted authority changed: %v", got)
	}
	stored.Attributes[0].WriteAuthority = userdomain.AttributeAuthorityUpstream
	stored, err = repo.FindByTenant(ctx, tenantID)
	if err != nil || stored.Attributes[0].WriteAuthority != userdomain.AttributeAuthorityLocal {
		t.Fatalf("returned schema aliases stored authority: %+v, %v", stored, err)
	}
}
