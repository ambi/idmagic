package handlers_http_test

// Entra のドメインの詳細の再取得と、管理者が取り込んだ Graph の登録値の照合を、管理 API から観測する。
// 期待値はテストが Entra の登録すべき値として独立に書き、製品の生成関数から作らない。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	entraDomainsPath   = "/api/admin/v1/wsfed/entra-federation/domains"
	contosoDomainPath  = entraDomainsPath + "/contoso.com"
	contosoObservePath = contosoDomainPath + "/observations"
	contosoTenantID    = "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"
)

// observedConfig は、Graph PowerShell が読み戻す、正しく適用した contoso.com の構成である。
type observedConfig map[string]string

func appliedContosoConfig(signingDER []byte) observedConfig {
	return observedConfig{
		"issuer_uri":                        contosoIssuer,
		"passive_sign_in_uri":               "https://idp.example/realms/default/wsfed",
		"active_sign_in_uri":                "https://idp.example/realms/default/trust/usernamemixed",
		"metadata_exchange_uri":             "https://idp.example/realms/default/trust/mex",
		"sign_out_uri":                      "https://idp.example/realms/default/wsfed?wa=wsignout1.0",
		"signing_certificate":               base64.StdEncoding.EncodeToString(signingDER),
		"next_signing_certificate":          "",
		"preferred_authentication_protocol": "wsFed",
		"federated_idp_mfa_behavior":        "acceptIfMfaDoneByFederatedIdp",
	}
}

func observationDocument(t *testing.T, tenantID, domain string, retrievedAt time.Time, configs ...observedConfig) string {
	t.Helper()
	if configs == nil {
		configs = []observedConfig{}
	}
	body, err := json.Marshal(map[string]any{
		"format": "idmagic.entra-federation-observation.v1", "entra_tenant_id": tenantID, "domain": domain,
		"authentication_type": "Federated", "retrieved_at": retrievedAt.UTC().Format(time.RFC3339Nano),
		"federation_configurations": configs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

type entraVerificationBody struct {
	Status                       string `json:"status"`
	RetrievedAt                  string `json:"retrieved_at"`
	CheckedAt                    string `json:"checked_at"`
	ObservedEntraTenantID        string `json:"observed_entra_tenant_id"`
	SigningCertificateThumbprint string `json:"signing_certificate_thumbprint"`
	Differences                  *[]struct {
		Field    string `json:"field"`
		Expected string `json:"expected"`
		Observed string `json:"observed"`
	} `json:"differences"`
}

func (v entraVerificationBody) fields() []string {
	if v.Differences == nil {
		return nil
	}
	out := []string{}
	for _, d := range *v.Differences {
		out = append(out, d.Field)
	}
	return out
}

type entraDomainDetail struct {
	Domain struct {
		Domain        string `json:"domain"`
		EntraTenantID string `json:"entra_tenant_id"`
	} `json:"domain"`
	ReplyURL   string            `json:"reply_url"`
	PowerShell map[string]string `json:"powershell"`
	Commands   struct {
		Create string `json:"create"`
		Update string `json:"update"`
	} `json:"commands"`
	Procedure    map[string]string `json:"procedure"`
	Certificates struct {
		Signing struct {
			Thumbprint string `json:"sha256_thumbprint"`
			NotAfter   string `json:"not_after"`
		} `json:"signing"`
		Next *struct {
			Thumbprint string `json:"sha256_thumbprint"`
		} `json:"next"`
	} `json:"certificates"`
	Verification     entraVerificationBody `json:"verification"`
	KnownLimitations []struct {
		Code string `json:"code"`
	} `json:"known_limitations"`
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status=%d body=%s, want %d", rec.Code, rec.Body.String(), status)
	}
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

func thumbprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

func listedStatus(t *testing.T, s entraServer, domain string) string {
	t.Helper()
	list := decodeBody[struct {
		Domains []struct {
			Domain       string                `json:"domain"`
			Verification entraVerificationBody `json:"verification"`
		} `json:"domains"`
	}](t, get(s.e, entraDomainsPath), http.StatusOK)
	for _, d := range list.Domains {
		if d.Domain == domain {
			return d.Verification.Status
		}
	}
	t.Fatalf("domain %s is not listed: %+v", domain, list)
	return ""
}

// entraRPUpdatedAt は、詳細の取得と取り込みが RP を変えないことを読むための更新時刻である。
func entraRPUpdatedAt(t *testing.T, s entraServer) string {
	t.Helper()
	list := decodeBody[struct {
		RelyingParties []struct {
			Wtrealm   string `json:"wtrealm"`
			UpdatedAt string `json:"updated_at"`
		} `json:"relying_parties"`
	}](t, get(s.e, "/api/admin/v1/wsfed/relying-parties"), http.StatusOK)
	for _, rp := range list.RelyingParties {
		if rp.Wtrealm == entraRealm {
			return rp.UpdatedAt
		}
	}
	t.Fatal("Entra RP is missing")
	return ""
}

type entraKeys struct {
	store   signingports.FederationKeyStore
	xmlCtx  context.Context
	current []byte
}

func newEntraVerificationServer(t *testing.T) (entraServer, entraKeys) {
	t.Helper()
	return newEntraVerificationServerWith(t, nil, entraAlice)
}

// newEntraVerificationServerWith は、XML 署名鍵を持つ鍵ストアの署名器で、users を保存したサーバーを返す。
func newEntraVerificationServerWith(t *testing.T, configure func(*httpadapter.Deps), users ...entraUser) (entraServer, entraKeys) {
	t.Helper()
	store, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	xmlCtx := signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
	active, err := store.GetActiveKey(xmlCtx)
	if err != nil {
		t.Fatal(err)
	}
	module := signingkeys.Module{KeyStore: store}
	s := newEntraServerConfigured(t, protectedSourceAnchor(), func(d *httpadapter.Deps) {
		d.FederationSigner = module.FederationSigner()
		if configure != nil {
			configure(d)
		}
	}, users...)
	return s, entraKeys{store: store, xmlCtx: xmlCtx, current: active.CertificateDER}
}

// 保存しただけでは一致にならず、正しく適用した登録値の取り込みで一致、鍵の準備と構成の変更で
// 再確認が必要、異なる登録値で項目ごとの差分つきの不一致になる。詳細の再取得と取り込みは RP を変えない。
//
//spec:covers REQ-WSFEDERATION-001: 詳細の再取得、手順、確認の状態（未確認、一致、再確認が必要、不一致）、証明書の空白差の無視、テナントの比較、RP を変えないこと。
//verify:operation GetEntraFederationDomain: 構成済みのドメインの詳細が、Graph の設定値、コマンド、手順、証明書、確認の状態を返すこと。
//verify:operation ListEntraFederationDomains: 構成済みのドメインごとに確認の状態を返すこと。
//verify:operation RecordEntraFederationObservation: 観測文書の取り込みが、比べた状態と差分を返すこと。
func TestEntraVerificationFollowsObservedSettings(t *testing.T) {
	s, keys := newEntraVerificationServer(t)
	configure := `{"domain":"contoso.com","source_anchor_attribute":"object_guid","entra_tenant_id":"` + strings.ToUpper(contosoTenantID) + `"}`
	if rec := configureEntra(s.e, configure); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := listedStatus(t, s, "contoso.com"); got != "unverified" {
		t.Fatalf("saved domain status = %q, want unverified", got)
	}
	updatedAt := entraRPUpdatedAt(t, s)

	detail := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK)
	if detail.Domain.Domain != "contoso.com" || detail.Domain.EntraTenantID != contosoTenantID || detail.ReplyURL != entraReply {
		t.Fatalf("saved settings = %+v", detail)
	}
	if detail.Verification.Status != "unverified" || detail.Verification.Differences != nil || detail.Verification.CheckedAt != "" {
		t.Fatalf("verification = %+v, want unverified without a record", detail.Verification)
	}
	if detail.PowerShell["IssuerUri"] != contosoIssuer || detail.PowerShell["SigningCertificate"] != base64.StdEncoding.EncodeToString(keys.current) {
		t.Fatalf("Graph settings = %v", detail.PowerShell)
	}
	if !strings.HasPrefix(detail.Commands.Create, "New-MgDomainFederationConfiguration -DomainId 'contoso.com'") ||
		!strings.HasPrefix(detail.Commands.Update, "Update-MgDomainFederationConfiguration -DomainId 'contoso.com'") {
		t.Fatalf("commands = %+v", detail.Commands)
	}
	if detail.Certificates.Signing.Thumbprint != thumbprint(keys.current) || detail.Certificates.Signing.NotAfter == "" || detail.Certificates.Next != nil {
		t.Fatalf("certificates = %+v", detail.Certificates)
	}
	if len(detail.KnownLimitations) != 1 || detail.KnownLimitations[0].Code != "hybrid_join_not_supported" {
		t.Fatalf("limitations = %+v", detail.KnownLimitations)
	}
	connect := detail.Procedure["connect"]
	for _, want := range []string{"Connect-MgGraph", "-TenantId '" + contosoTenantID + "'", "'Domain.Read.All'", "'Domain-InternalFederation.ReadWrite.All'", "throw"} {
		if !strings.Contains(connect, want) {
			t.Fatalf("connect procedure lacks %q: %s", want, connect)
		}
	}
	readOnly := detail.Procedure["connect_read_only"]
	if !strings.Contains(readOnly, "'Domain-InternalFederation.Read.All'") || strings.Contains(readOnly, "ReadWrite") {
		t.Fatalf("read-only connect procedure = %s", readOnly)
	}
	if backup := detail.Procedure["backup"]; !strings.Contains(backup, "Get-MgDomainFederationConfiguration -DomainId 'contoso.com'") || strings.Contains(backup, "Get-MgUser") {
		t.Fatalf("backup procedure = %s", backup)
	}
	if apply := detail.Procedure["apply"]; !strings.Contains(apply, "Update-MgDomainFederationConfiguration") || !strings.Contains(apply, "New-MgDomainFederationConfiguration") {
		t.Fatalf("apply procedure = %s", apply)
	}
	if readBack := detail.Procedure["read_back"]; !strings.Contains(readBack, "idmagic.entra-federation-observation.v1") || !strings.Contains(readBack, "WriteAllText") {
		t.Fatalf("read-back procedure = %s", readBack)
	}
	if detail.Procedure["compare"] == "" {
		t.Fatal("compare procedure is missing")
	}

	// 証明書の改行と空白は、Graph や転記で入っても設定の違いではない。
	applied := appliedContosoConfig(keys.current)
	wrapped := applied["signing_certificate"]
	applied["signing_certificate"] = wrapped[:64] + "\r\n " + wrapped[64:]
	applied["federated_idp_mfa_behavior"] = "AcceptIfMfaDoneByFederatedIdp"
	retrievedAt := time.Now().UTC().Add(-time.Minute)
	recorded := decodeBody[struct {
		Verification entraVerificationBody `json:"verification"`
	}](t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, strings.ToUpper(contosoTenantID), "Contoso.com", retrievedAt, applied)), http.StatusOK)
	if recorded.Verification.Status != "matched" || recorded.Verification.Differences == nil || len(*recorded.Verification.Differences) != 0 {
		t.Fatalf("applied observation = %+v, want matched with no differences", recorded.Verification)
	}
	revisited := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK)
	if revisited.Verification.Status != "matched" || revisited.Verification.SigningCertificateThumbprint != thumbprint(keys.current) ||
		!strings.EqualFold(revisited.Verification.ObservedEntraTenantID, contosoTenantID) || revisited.Verification.CheckedAt == "" {
		t.Fatalf("revisited verification = %+v", revisited.Verification)
	}
	if got, err := time.Parse(time.RFC3339Nano, revisited.Verification.RetrievedAt); err != nil || !got.Equal(retrievedAt) {
		t.Fatalf("retrieved_at = %q, want %s", revisited.Verification.RetrievedAt, retrievedAt)
	}
	if got := entraRPUpdatedAt(t, s); got != updatedAt {
		t.Fatalf("detail and observation changed the RP: updated_at %s -> %s", updatedAt, got)
	}

	// 鍵を準備すると Entra に次の証明書の登録が要るので、前の一致は今の証拠にならない。
	prepared, err := keys.store.Prepare(keys.xmlCtx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	stale := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK)
	if stale.Verification.Status != "stale" || stale.Verification.Differences != nil || stale.Certificates.Next == nil ||
		stale.Certificates.Next.Thumbprint != thumbprint(prepared.Next.CertificateDER) {
		t.Fatalf("after key preparation = %+v", stale)
	}
	if got := listedStatus(t, s, "contoso.com"); got != "stale" {
		t.Fatalf("listed status after key preparation = %q, want stale", got)
	}

	// 次の証明書を登録していない、IssuerUri の異なる、別テナントの登録値は、項目ごとの不一致である。
	drifted := appliedContosoConfig(keys.current)
	drifted["issuer_uri"] = "urn:other"
	mismatched := decodeBody[struct {
		Verification entraVerificationBody `json:"verification"`
	}](t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, "99999999-9999-9999-9999-999999999999", "contoso.com", time.Now().UTC(), drifted)), http.StatusOK)
	if got := strings.Join(mismatched.Verification.fields(), ","); mismatched.Verification.Status != "mismatched" ||
		got != "entra_tenant_id,issuer_uri,next_signing_certificate" {
		t.Fatalf("drifted observation = %+v (fields %s)", mismatched.Verification, got)
	}
	// Entra に構成がないことの観測は、未確認ではなく不一致である。
	absent := decodeBody[struct {
		Verification entraVerificationBody `json:"verification"`
	}](t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", time.Now().UTC())), http.StatusOK)
	if got := strings.Join(absent.Verification.fields(), ","); absent.Verification.Status != "mismatched" || got != "federation_configurations" {
		t.Fatalf("absent configuration = %+v", absent.Verification)
	}

	// 構成を変えると、前の観測は今の期待値と比べたものではない。
	if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid","entra_tenant_id":"`+contosoTenantID+`","issuer_uri":"urn:contoso:new"}`); rec.Code != http.StatusOK {
		t.Fatalf("reconfigure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := listedStatus(t, s, "contoso.com"); got != "stale" {
		t.Fatalf("status after reconfiguration = %q, want stale", got)
	}
}

// テナント ID を持たないドメインは、観測したテナントを記録するだけで比べない。
//
//spec:covers REQ-WSFEDERATION-001: テナント ID を持たない組では観測したテナントを比べずに記録すること。
func TestEntraVerificationWithoutTenantRecordsTheObservedTenant(t *testing.T) {
	s, keys := newEntraVerificationServer(t)
	if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	detail := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK)
	if strings.Contains(detail.Procedure["connect"], "-TenantId") || detail.Domain.EntraTenantID != "" {
		t.Fatalf("connect without a tenant = %s", detail.Procedure["connect"])
	}
	recorded := decodeBody[struct {
		Verification entraVerificationBody `json:"verification"`
	}](t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", time.Now().UTC(), appliedContosoConfig(keys.current))), http.StatusOK)
	if recorded.Verification.Status != "matched" || recorded.Verification.ObservedEntraTenantID != contosoTenantID {
		t.Fatalf("verification = %+v", recorded.Verification)
	}
}

// 拒否した観測文書と要求は、保存済みの確認を変えない。
//
//spec:covers REQ-WSFEDERATION-001: 形式の不正、別のドメイン、上限、未来の取得時刻の 400、古い取得時刻の 409、未構成のドメインの 404、GUID でないテナント ID の 400、管理者でない利用者の 403 が観測記録と RP を変えないこと。
func TestEntraObservationRefusalsKeepTheRecord(t *testing.T) {
	s, keys := newEntraVerificationServer(t)
	if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid","entra_tenant_id":"`+contosoTenantID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	applied := appliedContosoConfig(keys.current)
	retrievedAt := time.Now().UTC().Add(-time.Hour)
	if rec := doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", retrievedAt, applied)); rec.Code != http.StatusOK {
		t.Fatalf("record: status=%d body=%s", rec.Code, rec.Body.String())
	}
	before := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK).Verification
	unchanged := func(label string) {
		t.Helper()
		after := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK).Verification
		if after.Status != before.Status || after.CheckedAt != before.CheckedAt || after.RetrievedAt != before.RetrievedAt {
			t.Fatalf("%s changed the record: %+v -> %+v", label, before, after)
		}
	}
	now := time.Now().UTC()
	valid := observationDocument(t, contosoTenantID, "contoso.com", now, applied)
	missingNext := appliedContosoConfig(keys.current)
	delete(missingNext, "next_signing_certificate")
	invalid := map[string]string{
		"unknown field":   strings.Replace(valid, `"domain"`, `"extra":"x","domain"`, 1),
		"format":          strings.Replace(valid, "observation.v1", "observation.v2", 1),
		"missing field":   observationDocument(t, contosoTenantID, "contoso.com", now, missingNext),
		"missing tenant":  strings.Replace(valid, `"entra_tenant_id":"`+contosoTenantID+`",`, "", 1),
		"other domain":    observationDocument(t, contosoTenantID, "fabrikam.com", now, applied),
		"five configs":    observationDocument(t, contosoTenantID, "contoso.com", now, applied, applied, applied, applied, applied),
		"future":          observationDocument(t, contosoTenantID, "contoso.com", now.Add(6*time.Minute), applied),
		"not JSON":        `{"format":`,
		"over 64 KiB":     strings.Replace(valid, `"domain"`, `"pad":"`+strings.Repeat("x", 64<<10)+`","domain"`, 1),
		"empty authority": strings.Replace(valid, `"authentication_type":"Federated"`, `"authentication_type":""`, 1),
	}
	for label, body := range invalid {
		problemOf(t, doJSON(s.e, http.MethodPost, contosoObservePath, body), http.StatusBadRequest, "invalid_request")
		unchanged(label)
	}
	// 4 件までの構成は形式として受ける（照合では一つでないことを不一致にする）。境界の対照である。
	if rec := doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", retrievedAt, applied, applied, applied, applied)); rec.Code != http.StatusOK {
		t.Fatalf("four configurations: status=%d body=%s", rec.Code, rec.Body.String())
	}
	before = decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK).Verification
	problemOf(t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", retrievedAt.Add(-time.Second), applied)), http.StatusConflict, "observation_outdated")
	unchanged("older retrieval")

	problemOf(t, get(s.e, entraDomainsPath+"/fabrikam.com"), http.StatusNotFound, "not_found")
	problemOf(t, doJSON(s.e, http.MethodPost, entraDomainsPath+"/fabrikam.com/observations", observationDocument(t, contosoTenantID, "fabrikam.com", now, applied)), http.StatusNotFound, "not_found")
	problemOf(t, configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid","entra_tenant_id":"contoso"}`), http.StatusBadRequest, "invalid_request")
	if got := decodeBody[entraDomainDetail](t, get(s.e, contosoDomainPath), http.StatusOK).Domain.EntraTenantID; got != contosoTenantID {
		t.Fatalf("refused tenant ID replaced the saved one: %q", got)
	}

	s.actAs(entraAlice.id)
	problemOf(t, get(s.e, contosoDomainPath), http.StatusForbidden, "access_denied")
	problemOf(t, get(s.e, entraDomainsPath), http.StatusForbidden, "access_denied")
	problemOf(t, doJSON(s.e, http.MethodPost, contosoObservePath, observationDocument(t, contosoTenantID, "contoso.com", time.Now().UTC(), applied)), http.StatusForbidden, "access_denied")
	s.actAs(entraAdmin.id)
	unchanged("non-admin")
}

func problemOf(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	problem := decodeBody[struct {
		Type string `json:"type"`
	}](t, rec, status)
	if problem.Type != "urn:idmagic:error:"+code {
		t.Fatalf("problem type = %q, want %s", problem.Type, code)
	}
}

// wsfed:read は詳細と一覧に届いて取り込みに届かず、wsfed:write は取り込みに届いて詳細に届かない。
// 別テナントのトークンはどちらも 401 で拒否する。拒否のたびに確認の状態を読み直す。
//
//spec:covers REQ-WSFEDERATION-001: wsfed:read による一覧と詳細の取得、wsfed:write による観測文書の取り込み、読み取りのスコープと別テナントのトークンが観測記録を変えないこと。
func TestEntraVerificationOperationsFollowTheGranularScopes(t *testing.T) {
	s := newWsFedScopeStack(t)
	read, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedRead)
	write, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedWrite)
	foreign, _ := s.IssueApiToken(t, stack.OtherRealm, apitokendomain.ScopeWsFedWrite)
	if rec := wsfedAdminRequest(s, http.MethodPost, wsfedScopeEntraPath, write, wsfedScopeEntraBody()); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	status := func() string {
		t.Helper()
		return decodeBody[entraDomainDetail](t, wsfedAdminRequest(s, http.MethodGet, contosoDomainPath, read, ""), http.StatusOK).Verification.Status
	}
	if rec := wsfedAdminRequest(s, http.MethodGet, entraDomainsPath, read, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "contoso.com") {
		t.Fatalf("wsfed:read list: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := wsfedAdminRequest(s, http.MethodGet, contosoDomainPath, write, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("wsfed:write detail: status=%d", rec.Code)
	}
	document := observationDocument(t, contosoTenantID, "contoso.com", time.Now().UTC())
	refused := wsfedAdminRequest(s, http.MethodPost, contosoObservePath, read, document)
	problemOf(t, refused, http.StatusForbidden, "insufficient_scope")
	if got := refused.Header().Get("WWW-Authenticate"); !strings.Contains(got, `scope="wsfed:write"`) {
		t.Fatalf("WWW-Authenticate=%q, want wsfed:write", got)
	}
	if rec := wsfedAdminRequest(s, http.MethodPost, contosoObservePath, foreign, document); rec.Code != http.StatusUnauthorized {
		t.Fatalf("foreign token: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := status(); got != "unverified" {
		t.Fatalf("refused observations recorded %q", got)
	}
	if rec := wsfedAdminRequest(s, http.MethodPost, contosoObservePath, write, document); rec.Code != http.StatusOK {
		t.Fatalf("wsfed:write observation: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := status(); got != "mismatched" {
		t.Fatalf("recorded status = %q, want mismatched for an absent configuration", got)
	}
}
