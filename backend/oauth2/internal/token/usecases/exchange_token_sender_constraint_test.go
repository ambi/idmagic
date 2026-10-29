package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// 拒否は署名の前に起きなければならない。署名の後で拒否すると、応答に載らなくても
// AccessTokenIssued が監査へ残り、発行していないトークンを発行したと記録する。
//
//spec:covers REQ-OAUTH2-056: 別の鍵の証明による交換は invalid_grant になり、署名せず、AccessTokenIssued と TokenExchanged を発行せず、TokenExchangeRejected だけを発行する。
func TestExchangeTokenRejectsAnotherKeyWithoutIssuing(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read", SenderConstraint: &domain.SenderConstraint{
			Type: spec.SenderConstraintDPoP, JKT: "k1",
		}},
	})
	var events []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }

	_, err := ExchangeToken(testing_tenant.Default(context.Background()), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"}, ProofJKT: "k2",
	}, time.Now().UTC())
	oauthErr, ok := errors.AsType[*OAuthError](err)
	if !ok || oauthErr.Code != "invalid_grant" {
		t.Fatalf("err=%v, want invalid_grant", err)
	}
	if issuer.calls != 0 {
		t.Fatalf("SignAccessToken calls=%d, want 0", issuer.calls)
	}
	if len(events) != 1 {
		t.Fatalf("events=%#v, want only TokenExchangeRejected", events)
	}
	if rejected, ok := events[0].(*domain.TokenExchangeRejected); !ok || rejected.Reason != "invalid_grant" {
		t.Fatalf("event=%#v, want TokenExchangeRejected with invalid_grant", events[0])
	}
}

// 同じ要求に別の DPoP 鍵があっても、mTLS に束縛した subject は証明書の制約を引き継ぐ。
// 両方の証明で DPoP を優先する未束縛の規則をここへ持ち込むと、制約が付け替わる。
//
//spec:covers REQ-OAUTH2-056: mTLS に束縛した subject_token は、同じ証明書の交換で同じ cnf を引き継ぎ、Bearer の提示形式を返す。
func TestExchangeTokenInheritsMTLSConstraint(t *testing.T) {
	issuer := &recordingIssuer{}
	subject := &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: "c1"}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read", SenderConstraint: subject},
	})

	res, err := ExchangeToken(testing_tenant.Default(context.Background()), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
		ProofJKT: "k2", ProofX5TS256: "c1",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}
	if !reflect.DeepEqual(issuer.lastInput.SenderConstraint, subject) {
		t.Fatalf("SenderConstraint=%+v, want %+v", issuer.lastInput.SenderConstraint, subject)
	}
	if res.TokenType != "Bearer" {
		t.Fatalf("token_type=%q, want Bearer for a certificate-bound token", res.TokenType)
	}
}

//spec:covers REQ-OAUTH2-056: 外部ワークロードのアテステーションの subject_token は制約のない subject として扱い、要求の DPoP 鍵に結び付ける。
func TestExchangeTokenBindsWorkloadSubjectToTheProof(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, nil)
	deps.ClientRepo.(*oauth2memory.OAuth2ClientRepository).Seed(&domain.OAuth2Client{
		ClientID: "agent-bound-client", ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, Scope: "read", CreatedAt: time.Now().UTC(),
	})
	deps.WorkloadVerifier = fakeWorkloadVerifier{grant: &workloaddomain.WorkloadIdentityGrant{
		AgentID: "agent_1", ClientID: "agent-bound-client", TrustBundleID: "bundle_1", BindingID: "binding_1",
	}}

	res, err := ExchangeToken(testing_tenant.Default(context.Background()), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "external-svid", SubjectTokenType: tokenTypeJWTURN,
		Resource: []string{"https://api.example"}, ProofJKT: "k2",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}
	want := &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: "k2"}
	if !reflect.DeepEqual(issuer.lastInput.SenderConstraint, want) || res.TokenType != "DPoP" {
		t.Fatalf("SenderConstraint=%+v token_type=%q, want %+v DPoP", issuer.lastInput.SenderConstraint, res.TokenType, want)
	}
}
