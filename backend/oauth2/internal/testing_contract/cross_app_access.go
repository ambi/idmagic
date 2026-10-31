package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// CrossAppAccessFixture は ID-JAG の許可の保存先と、二つのテナントである。
type CrossAppAccessFixture struct {
	Policies ports.CrossAppAccessPolicyRepository
	TenantA  string
	TenantB  string
	Now      time.Time
}

// RunCrossAppAccess は保存先の共有契約である。往復、一意の組、テナント境界、削除を確かめる。
func RunCrossAppAccess(t *testing.T, fixture CrossAppAccessFixture) {
	t.Helper()
	ctx := context.Background()
	repo := fixture.Policies
	const (
		issueID     = "a0000000-0000-4000-8000-000000000001"
		issueDupID  = "a0000000-0000-4000-8000-000000000002"
		issueOther  = "a0000000-0000-4000-8000-000000000003"
		redeemID    = "b0000000-0000-4000-8000-000000000001"
		redeemDupID = "b0000000-0000-4000-8000-000000000002"
		redeemOther = "b0000000-0000-4000-8000-000000000003"
	)
	issue := &domain.CrossAppAccessIssuePolicy{
		TenantID: fixture.TenantA, ID: issueID, ClientID: "client", Audience: "https://as.example/",
		RelyingPartyClientID: "rp", TargetClientID: "target", Resources: []string{"https://r.example/"},
		Scopes: []string{"read"}, AuthorizationDetailTypes: []string{"payment"}, ReleaseEmail: true, CreatedAt: fixture.Now,
	}
	if err := repo.CreateIssuePolicy(ctx, issue); err != nil {
		t.Fatalf("CreateIssuePolicy: %v", err)
	}
	found, err := repo.FindIssuePolicy(ctx, fixture.TenantA, issueID)
	if err != nil || found == nil || found.Audience != issue.Audience || found.TargetClientID != "target" ||
		found.RelyingPartyClientID != "rp" || len(found.Resources) != 1 || found.Scopes[0] != "read" ||
		found.AuthorizationDetailTypes[0] != "payment" || !found.ReleaseEmail || !found.CreatedAt.Equal(fixture.Now) {
		t.Fatalf("FindIssuePolicy = (%+v, %v)", found, err)
	}
	duplicate := *issue
	duplicate.ID = issueDupID
	if err := repo.CreateIssuePolicy(ctx, &duplicate); !errors.Is(err, ports.ErrCrossAppAccessPolicyExists) {
		t.Fatalf("duplicate CreateIssuePolicy = %v", err)
	}
	// 同じ組でも別のテナントなら別の許可である。
	other := *issue
	other.ID, other.TenantID = issueOther, fixture.TenantB
	if err := repo.CreateIssuePolicy(ctx, &other); err != nil {
		t.Fatalf("other tenant CreateIssuePolicy: %v", err)
	}
	if leaked, err := repo.FindIssuePolicy(ctx, fixture.TenantB, issueID); err != nil || leaked != nil {
		t.Fatalf("other tenant FindIssuePolicy = (%+v, %v)", leaked, err)
	}
	listed, err := repo.ListIssuePolicies(ctx, fixture.TenantA)
	if err != nil || len(listed) != 1 || listed[0].ID != issueID {
		t.Fatalf("ListIssuePolicies = (%+v, %v)", listed, err)
	}
	if err := repo.DeleteIssuePolicy(ctx, fixture.TenantB, issueID); err != nil {
		t.Fatalf("other tenant DeleteIssuePolicy: %v", err)
	}
	if kept, err := repo.FindIssuePolicy(ctx, fixture.TenantA, issueID); err != nil || kept == nil {
		t.Fatalf("delete crossed the tenant boundary: (%+v, %v)", kept, err)
	}
	if err := repo.DeleteIssuePolicy(ctx, fixture.TenantA, issueID); err != nil {
		t.Fatalf("DeleteIssuePolicy: %v", err)
	}
	if gone, err := repo.FindIssuePolicy(ctx, fixture.TenantA, issueID); err != nil || gone != nil {
		t.Fatalf("after delete = (%+v, %v)", gone, err)
	}

	redeem := &domain.CrossAppAccessRedeemPolicy{
		TenantID: fixture.TenantA, ID: redeemID, ProviderID: "idp", ClientID: "client", Scopes: []string{"read"}, CreatedAt: fixture.Now,
	}
	if err := repo.CreateRedeemPolicy(ctx, redeem); err != nil {
		t.Fatalf("CreateRedeemPolicy: %v", err)
	}
	if found, err := repo.FindRedeemPolicy(ctx, fixture.TenantA, redeemID); err != nil || found == nil || found.ProviderID != "idp" || found.Scopes[0] != "read" {
		t.Fatalf("FindRedeemPolicy = (%+v, %v)", found, err)
	}
	if found, err := repo.FindRedeemPolicyFor(ctx, fixture.TenantA, "idp", "client"); err != nil || found == nil || found.ID != redeemID {
		t.Fatalf("FindRedeemPolicyFor = (%+v, %v)", found, err)
	}
	if none, err := repo.FindRedeemPolicyFor(ctx, fixture.TenantA, "idp", "another"); err != nil || none != nil {
		t.Fatalf("another client FindRedeemPolicyFor = (%+v, %v)", none, err)
	}
	if leaked, err := repo.FindRedeemPolicyFor(ctx, fixture.TenantB, "idp", "client"); err != nil || leaked != nil {
		t.Fatalf("other tenant FindRedeemPolicyFor = (%+v, %v)", leaked, err)
	}
	duplicateRedeem := *redeem
	duplicateRedeem.ID = redeemDupID
	if err := repo.CreateRedeemPolicy(ctx, &duplicateRedeem); !errors.Is(err, ports.ErrCrossAppAccessPolicyExists) {
		t.Fatalf("duplicate CreateRedeemPolicy = %v", err)
	}
	otherRedeem := *redeem
	otherRedeem.ID, otherRedeem.TenantID = redeemOther, fixture.TenantB
	if err := repo.CreateRedeemPolicy(ctx, &otherRedeem); err != nil {
		t.Fatalf("other tenant CreateRedeemPolicy: %v", err)
	}
	if listed, err := repo.ListRedeemPolicies(ctx, fixture.TenantA); err != nil || len(listed) != 1 {
		t.Fatalf("ListRedeemPolicies = (%+v, %v)", listed, err)
	}
	if err := repo.DeleteRedeemPolicy(ctx, fixture.TenantA, redeemID); err != nil {
		t.Fatalf("DeleteRedeemPolicy: %v", err)
	}
	if gone, err := repo.FindRedeemPolicy(ctx, fixture.TenantA, redeemID); err != nil || gone != nil {
		t.Fatalf("after delete = (%+v, %v)", gone, err)
	}
	if kept, err := repo.FindRedeemPolicy(ctx, fixture.TenantB, redeemOther); err != nil || kept == nil {
		t.Fatalf("delete crossed the tenant boundary: (%+v, %v)", kept, err)
	}
}
