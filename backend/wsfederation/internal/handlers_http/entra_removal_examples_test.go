package handlers_http_test

// Entra のドメインの解除を管理 API から観測する。Entra の送り先は、Microsoft のレルム探索のテストダブルが
// ドメインごとに答える。期待値はテストが Entra の状態として独立に書き、製品の判定関数から作らない。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

const (
	fabrikamDomainPath = entraDomainsPath + "/fabrikam.com"
	// idmagicPassive は、テストのサーバーが Entra に登録させる受動のエンドポイントである。
	idmagicPassive    = "https://idp.example/realms/default/wsfed"
	previousIdPSignIn = "https://adfs.fabrikam.com/adfs/ls/"
)

// fakeRealms は、Microsoft のレルム探索のテストダブルである。ドメインごとの答えと失敗を返し、呼ばれた
// ドメインを記録する。
type fakeRealms struct {
	mu      sync.Mutex
	answers map[string]feddomain.EntraDomainRealm
	calls   []string
}

func (f *fakeRealms) answer(domain string, realm feddomain.EntraDomainRealm) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers[domain] = realm
}

func (f *fakeRealms) Discover(_ context.Context, domain string) (feddomain.EntraDomainRealm, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, domain)
	realm, ok := f.answers[domain]
	if !ok {
		return feddomain.EntraDomainRealm{}, errors.New("realm discovery unavailable")
	}
	return realm, nil
}

// federatedRealm は、Entra のサインインの画面が WS-Fed の受動のエンドポイントへ送るときの探索の答えである。
// AuthURL は、登録した PassiveSignInUri に Entra が問い合わせ文字列を付けた形になる。
func federatedRealm(passive, domain string) feddomain.EntraDomainRealm {
	return feddomain.EntraDomainRealm{
		NamespaceType: "Federated",
		AuthURL:       passive + "?username=idmagic-realm-check%40" + domain + "&wa=wsignin1.0&wtrealm=urn%3afederation%3aMicrosoftOnline&wctx=",
	}
}

var managedRealm = feddomain.EntraDomainRealm{NamespaceType: "Managed"}

// managedRollbackObservation は、contoso.com を Managed に切り戻した後に読み戻した観測文書である。
func managedRollbackObservation(t *testing.T, retrievedAt time.Time) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"format": "idmagic.entra-federation-observation.v1", "entra_tenant_id": contosoTenantID, "domain": "contoso.com",
		"authentication_type": "Managed", "retrieved_at": retrievedAt.UTC().Format(time.RFC3339Nano),
		"federation_configurations": []observedConfig{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

type removalResult struct {
	Domain              string   `json:"domain"`
	RemainingDomains    []string `json:"remaining_domains"`
	RelyingPartyDeleted bool     `json:"relying_party_deleted"`
	ConfirmedBy         string   `json:"confirmed_by"`
}

// listedEntraRP は、RP の一覧から読む Entra の RP の保存値である。
type listedEntraRP struct {
	Wtrealm     string          `json:"wtrealm"`
	ReplyURLs   []string        `json:"reply_urls"`
	ClaimPolicy json.RawMessage `json:"claim_policy"`
	CreatedAt   string          `json:"created_at"`
	Profile     struct {
		Domains []struct {
			Domain        string `json:"domain"`
			IssuerURI     string `json:"issuer_uri"`
			EntraTenantID string `json:"entra_tenant_id"`
		} `json:"domains"`
		SourceAnchorAttribute string `json:"source_anchor_attribute"`
		ImmutableIDAttribute  string `json:"immutable_id_attribute"`
		PassiveLogOnURI       string `json:"passive_logon_uri"`
		ActiveLogOnURI        string `json:"active_logon_uri"`
		MetadataExchangeURI   string `json:"metadata_exchange_uri"`
	} `json:"entra_profile"`
}

// entraRelyingParty は、RP の一覧から Entra の RP を読む。なければ nil を返す。
func entraRelyingParty(t *testing.T, s entraServer) *listedEntraRP {
	t.Helper()
	list := decodeBody[struct {
		RelyingParties []listedEntraRP `json:"relying_parties"`
	}](t, get(s.e, "/api/admin/v1/wsfed/relying-parties"), http.StatusOK)
	for _, rp := range list.RelyingParties {
		if rp.Wtrealm == entraRealm {
			return &rp
		}
	}
	return nil
}

// entraRemovalServer は、contoso.com（Entra のテナント ID つき）と fabrikam.com を管理 API で構成した
// サーバー、観測記録を読み戻すリポジトリ、レルム探索のテストダブルを返す。探索は最初はどのドメインにも
// 答えず失敗する。
func entraRemovalServer(t *testing.T) (entraServer, entraKeys, wsfedports.EntraFederationObservationRepository, *fakeRealms) {
	t.Helper()
	var observations wsfedports.EntraFederationObservationRepository
	realms := &fakeRealms{answers: map[string]feddomain.EntraDomainRealm{}}
	s, keys := newEntraVerificationServerWith(t, func(d *httpadapter.Deps) {
		observations = d.WsFederation.ObservationRepo
		d.WsFederation.RealmDiscovery = realms
	}, entraAlice, entraDave)
	for _, body := range []string{
		`{"domain":"contoso.com","source_anchor_attribute":"object_guid","entra_tenant_id":"` + contosoTenantID + `"}`,
		`{"domain":"fabrikam.com","issuer_uri":"` + fabrikamIssuer + `","source_anchor_attribute":"object_guid"}`,
	} {
		if rec := configureEntra(s.e, body); rec.Code != http.StatusCreated {
			t.Fatalf("configure %s: status=%d body=%s", body, rec.Code, rec.Body.String())
		}
	}
	*s.events = nil
	return s, keys, observations, realms
}

func recordObservation(t *testing.T, s entraServer, document string) {
	t.Helper()
	if rec := doJSON(s.e, http.MethodPost, contosoObservePath, document); rec.Code != http.StatusOK {
		t.Fatalf("record observation: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func removedEvent(t *testing.T, events []spec.DomainEvent) map[string]any {
	t.Helper()
	for _, event := range events {
		if event.EventType() != "EntraFederationDomainRemoved" {
			continue
		}
		data, err := spec.MarshalDomainEvent(event)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(data, &wire); err != nil {
			t.Fatal(err)
		}
		return wire
	}
	return nil
}

func issueTo(t *testing.T, s entraServer, user entraUser) (int, string) {
	t.Helper()
	rec := postWsTrustSOAP(s.e, entraRST(t, "urn:uuid:removal-"+user.id+"-"+time.Now().Format(time.RFC3339Nano), user.upn))
	return rec.Code, rec.Body.String()
}

// Microsoft のレルム探索が Managed と答える contoso.com だけを外し、fabrikam.com は同じ設定で署名とサインインを
// 続ける。探索が以前の IdP へ送ると答える最後の fabrikam.com を外すと、RP を消し、署名鍵は残す。
//
//spec:covers REQ-WSFEDERATION-001: ドメインの解除が、探索の Managed か IdMagic を指さない Federated を根拠に、そのドメインの組だけを外してほかの組、sourceAnchor、エンドポイント、返信先、クレーム、作成時刻を保ち、観測記録を消し、確認の根拠を持つ EntraFederationDomainRemoved を発行し、残るドメインを返すこと。最後のドメインの解除が RP を削除し、署名鍵を残すこと。解除したドメインの利用者への発行を拒否すること。
//verify:operation RemoveEntraFederationDomain: 探索で切り戻しを確かめたドメインだけを解除し、残るドメイン、RP の削除の有無、確認の根拠を返すこと。
func TestEntraDomainRemovalKeepsTheOtherDomainSigningIn(t *testing.T) {
	s, keys, observations, realms := entraRemovalServer(t)
	before := entraRelyingParty(t, s)
	if before == nil || len(before.Profile.Domains) != 2 || before.Profile.PassiveLogOnURI != idmagicPassive {
		t.Fatalf("configured RP = %+v", before)
	}
	// 構成の後に取り込んだ一致の観測は、解除と同時に消える。
	recordObservation(t, s, observationDocument(t, contosoTenantID, "contoso.com", time.Now().UTC(), appliedContosoConfig(keys.current)))
	realms.answer("contoso.com", managedRealm)

	rec := doJSON(s.e, http.MethodDelete, contosoDomainPath, "")
	removed := decodeBody[removalResult](t, rec, http.StatusOK)
	if removed.Domain != "contoso.com" || !slices.Equal(removed.RemainingDomains, []string{"fabrikam.com"}) ||
		removed.RelyingPartyDeleted || removed.ConfirmedBy != "realm_discovery" {
		t.Fatalf("removal = %+v, want contoso.com removed by realm discovery with fabrikam.com remaining", removed)
	}

	after := entraRelyingParty(t, s)
	if after == nil || len(after.Profile.Domains) != 1 {
		t.Fatalf("RP after removal = %+v, want only fabrikam.com", after)
	}
	if got := after.Profile.Domains[0]; got.Domain != "fabrikam.com" || got.IssuerURI != fabrikamIssuer {
		t.Fatalf("remaining pair = %+v, want fabrikam.com with %s", got, fabrikamIssuer)
	}
	if after.Profile.SourceAnchorAttribute != "object_guid" || after.Profile.PassiveLogOnURI != before.Profile.PassiveLogOnURI ||
		after.Profile.ActiveLogOnURI != before.Profile.ActiveLogOnURI || after.Profile.MetadataExchangeURI != before.Profile.MetadataExchangeURI ||
		after.Profile.ImmutableIDAttribute != before.Profile.ImmutableIDAttribute || after.CreatedAt != before.CreatedAt ||
		!slices.Equal(after.ReplyURLs, before.ReplyURLs) || string(after.ClaimPolicy) != string(before.ClaimPolicy) {
		t.Fatalf("removal changed the remaining settings:\nbefore %+v\nafter  %+v", before, after)
	}
	if record, err := observations.Find(context.Background(), tenancydomain.DefaultTenantID, "contoso.com"); err != nil || record != nil {
		t.Fatalf("observation record of the removed domain = %+v, %v; want deleted", record, err)
	}
	if rec := get(s.e, contosoDomainPath); rec.Code != http.StatusNotFound {
		t.Fatalf("removed domain detail status=%d, want 404", rec.Code)
	}
	wire := removedEvent(t, *s.events)
	if wire == nil || wire["tenantId"] != tenancydomain.DefaultTenantID || wire["domain"] != "contoso.com" ||
		wire["issuerUri"] != contosoIssuer || wire["confirmedBy"] != "realm_discovery" || wire["relyingPartyDeleted"] != false {
		t.Fatalf("EntraFederationDomainRemoved = %v", wire)
	}

	// 残る fabrikam.com の利用者は同じ IssuerUri で署名したアサーションを受け取り、外した contoso.com の利用者は受け取らない。
	status, body := issueTo(t, s, entraDave)
	if status != http.StatusOK {
		t.Fatalf("fabrikam.com issue status=%d body=%s", status, body)
	}
	if got := entraAssertion(t, body, false).SelectAttrValue("Issuer", ""); got != fabrikamIssuer {
		t.Fatalf("fabrikam.com Issuer = %q, want %q", got, fabrikamIssuer)
	}
	if status, body := issueTo(t, s, entraAlice); status != http.StatusInternalServerError || strings.Contains(body, "Assertion") {
		t.Fatalf("removed contoso.com issue status=%d body=%s, want a refusal without a token", status, body)
	}
	// 解除を済ませたドメインは、探索を呼ばずに 404 を返す。
	calls := len(realms.calls)
	if rec := doJSON(s.e, http.MethodDelete, contosoDomainPath, ""); rec.Code != http.StatusNotFound || len(realms.calls) != calls {
		t.Fatalf("repeated removal status=%d calls %d -> %d, want 404 without a lookup", rec.Code, calls, len(realms.calls))
	}

	// 最後のドメインは、探索が以前の IdP へ送ると答えた後に外し、RP を消す。署名鍵は残る。
	*s.events = nil
	realms.answer("fabrikam.com", federatedRealm(previousIdPSignIn, "fabrikam.com"))
	last := decodeBody[removalResult](t, doJSON(s.e, http.MethodDelete, fabrikamDomainPath, ""), http.StatusOK)
	if last.Domain != "fabrikam.com" || len(last.RemainingDomains) != 0 || !last.RelyingPartyDeleted || last.ConfirmedBy != "realm_discovery" {
		t.Fatalf("last removal = %+v, want the RP deleted", last)
	}
	if rp := entraRelyingParty(t, s); rp != nil {
		t.Fatalf("Entra RP after the last removal = %+v, want deleted", rp)
	}
	if wire := removedEvent(t, *s.events); wire == nil || wire["relyingPartyDeleted"] != true {
		t.Fatalf("last EntraFederationDomainRemoved = %v", wire)
	}
	if status, body := issueTo(t, s, entraDave); status == http.StatusOK {
		t.Fatalf("issue after the last removal status=%d body=%s, want a refusal", status, body)
	}
	// 署名鍵を破棄していないので、次に構成するドメインは同じ証明書を Entra へ登録させる。
	if rec := configureEntra(s.e, `{"domain":"fabrikam.com","issuer_uri":"`+fabrikamIssuer+`","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
		t.Fatalf("reconfigure after the last removal: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if detail := decodeBody[entraDomainDetail](t, get(s.e, fabrikamDomainPath), http.StatusOK); detail.Certificates.Signing.Thumbprint != thumbprint(keys.current) ||
		detail.Verification.Status != "unverified" {
		t.Fatalf("reconfigured domain = %+v, want the same signing certificate and no carried-over observation", detail)
	}
}

// Entra がまだ IdMagic へ送っているか、確かめられない間は解除を拒否し、RP、観測記録、イベントを変えない。
// 探索で確かめられなくても、切り戻しを示す観測を取り込めば解除できる。
//
//spec:covers REQ-WSFEDERATION-001: 探索が IdMagic の受動のエンドポイントを答える間は観測記録を問わずに 422 の entra_federation_still_active で、探索が失敗するか Unknown で切り戻しを示す観測記録がない間は 422 の entra_rollback_unconfirmed で拒否し、RP、観測記録、イベントを変えないこと。探索が失敗しても切り戻しを示す観測記録を根拠に解除すること。構成していないドメインを探索を呼ばずに 404 で拒否すること。
func TestEntraDomainRemovalRefusesWhileEntraStillFederatesTheDomain(t *testing.T) {
	retrievedAt := time.Now().UTC().Add(-time.Minute)
	managedObservation := func(t *testing.T) string {
		t.Helper()
		return managedRollbackObservation(t, retrievedAt)
	}
	for _, tc := range []struct {
		name        string
		realm       *feddomain.EntraDomainRealm // nil なら探索は失敗する
		observation func(*testing.T, entraKeys) string
		path        string
		status      int
		code        string
		lookups     int
	}{
		{
			name: "still federated to IdMagic", realm: new(federatedRealm(idmagicPassive, "contoso.com")),
			status: http.StatusUnprocessableEntity, code: "entra_federation_still_active", lookups: 1,
		},
		{
			name: "still federated to IdMagic despite a managed observation", realm: new(federatedRealm(idmagicPassive, "contoso.com")),
			observation: func(t *testing.T, _ entraKeys) string { t.Helper(); return managedObservation(t) },
			status:      http.StatusUnprocessableEntity, code: "entra_federation_still_active", lookups: 1,
		},
		{
			name: "lookup failed without an observation", status: http.StatusUnprocessableEntity, code: "entra_rollback_unconfirmed", lookups: 1,
		},
		{
			name: "federated with an unreadable destination", realm: &feddomain.EntraDomainRealm{NamespaceType: "Federated", AuthURL: "https://%zz"},
			status: http.StatusUnprocessableEntity, code: "entra_rollback_unconfirmed", lookups: 1,
		},
		{
			name: "unknown with an observation that still points to IdMagic", realm: &feddomain.EntraDomainRealm{NamespaceType: "Unknown"},
			observation: func(t *testing.T, keys entraKeys) string {
				t.Helper()
				return observationDocument(t, contosoTenantID, "contoso.com", retrievedAt, appliedContosoConfig(keys.current))
			},
			status: http.StatusUnprocessableEntity, code: "entra_rollback_unconfirmed", lookups: 1,
		},
		{
			name: "domain not configured", path: entraDomainsPath + "/northwind.com",
			status: http.StatusNotFound, code: "not_found", lookups: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, keys, observations, realms := entraRemovalServer(t)
			if tc.realm != nil {
				realms.answer("contoso.com", *tc.realm)
			}
			if tc.observation != nil {
				recordObservation(t, s, tc.observation(t, keys))
			}
			before := entraRelyingParty(t, s)
			recordBefore, err := observations.Find(context.Background(), tenancydomain.DefaultTenantID, "contoso.com")
			if err != nil {
				t.Fatal(err)
			}
			*s.events = nil
			path := tc.path
			if path == "" {
				path = contosoDomainPath
			}
			rec := doJSON(s.e, http.MethodDelete, path, "")
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), "urn:idmagic:error:"+tc.code) {
				t.Fatalf("status=%d body=%s, want %d %s", rec.Code, rec.Body.String(), tc.status, tc.code)
			}
			if len(realms.calls) != tc.lookups {
				t.Fatalf("realm discovery calls = %v, want %d", realms.calls, tc.lookups)
			}
			after := entraRelyingParty(t, s)
			beforeJSON, _ := json.Marshal(before)
			afterJSON, _ := json.Marshal(after)
			if !bytes.Equal(beforeJSON, afterJSON) {
				t.Fatalf("refused removal changed the RP:\nbefore %s\nafter  %s", beforeJSON, afterJSON)
			}
			recordAfter, err := observations.Find(context.Background(), tenancydomain.DefaultTenantID, "contoso.com")
			if err != nil || (recordBefore == nil) != (recordAfter == nil) ||
				(recordBefore != nil && !recordAfter.Observation.RetrievedAt.Equal(recordBefore.Observation.RetrievedAt)) {
				t.Fatalf("refused removal changed the observation record: %+v -> %+v (%v)", recordBefore, recordAfter, err)
			}
			if wire := removedEvent(t, *s.events); wire != nil {
				t.Fatalf("refused removal emitted %v", wire)
			}
		})
	}

	t.Run("lookup failed with a managed observation", func(t *testing.T) {
		s, _, _, _ := entraRemovalServer(t)
		recordObservation(t, s, managedObservation(t))
		removed := decodeBody[removalResult](t, doJSON(s.e, http.MethodDelete, contosoDomainPath, ""), http.StatusOK)
		if removed.ConfirmedBy != "observation" || !slices.Equal(removed.RemainingDomains, []string{"fabrikam.com"}) {
			t.Fatalf("removal = %+v, want contoso.com removed on the observation", removed)
		}
		if wire := removedEvent(t, *s.events); wire == nil || wire["confirmedBy"] != "observation" {
			t.Fatalf("EntraFederationDomainRemoved = %v", wire)
		}
	})
}

// 解除は変更の操作なので wsfed:write だけが届き、wsfed:read と別テナントのトークンの拒否は RP を変えない。
// このスタックは探索を配線しないので、解除は取り込んだ観測を根拠にする。
//
//spec:covers REQ-WSFEDERATION-001: wsfed:read のトークンによるドメインの解除を 403 の insufficient_scope で、別テナントのトークンを 401 で拒否して RP を変えず、wsfed:write のトークンの解除が RP から組を外すこと。
func TestEntraDomainRemovalFollowsTheGranularScopes(t *testing.T) {
	s := newWsFedScopeStack(t)
	realm := tenancydomain.DefaultRealm
	read, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeWsFedRead)
	write, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeWsFedWrite)
	foreign, _ := s.IssueApiToken(t, stack.OtherRealm, apitokendomain.ScopeWsFedWrite)
	if rec := wsfedAdminRequest(s, http.MethodPost, wsfedScopeEntraPath, write, wsfedScopeEntraBody()); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	managedAt := time.Now().UTC().Add(-time.Minute)
	if rec := wsfedAdminRequest(s, http.MethodPost, contosoObservePath, write, managedRollbackObservation(t, managedAt)); rec.Code != http.StatusOK {
		t.Fatalf("observation: status=%d body=%s", rec.Code, rec.Body.String())
	}
	domains := func() int {
		t.Helper()
		rp, err := s.WsFedRPs.FindByWtrealm(s.RealmContext(t, realm), s.TenantID(t, realm), wsfedScopeEntraRP)
		if err != nil {
			t.Fatal(err)
		}
		if rp == nil || rp.EntraProfile == nil {
			return 0
		}
		return len(rp.EntraProfile.Domains)
	}
	assertWsFedInsufficientScope(t, wsfedAdminRequest(s, http.MethodDelete, contosoDomainPath, read, ""))
	assertWsFedInvalidToken(t, wsfedAdminRequest(s, http.MethodDelete, contosoDomainPath, foreign, ""), "foreign wsfed:write removal")
	if got := domains(); got != 1 {
		t.Fatalf("refused removals changed the RP: %d domains, want 1", got)
	}
	if rec := wsfedAdminRequest(s, http.MethodDelete, contosoDomainPath, write, ""); rec.Code != http.StatusOK {
		t.Fatalf("wsfed:write removal: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := domains(); got != 0 {
		t.Fatalf("removal kept %d domains, want the RP deleted", got)
	}
}

// 汎用の削除は Entra の RP を消さず、全ドメインの信頼を残す。
//
//spec:covers REQ-WSFEDERATION-001: Entra の定型設定を持つ RP の汎用の削除を 409 の entra_relying_party で拒否し、RP と観測記録を残すこと。
//verify:operation DeleteWsFedRelyingParty: Entra の RP の削除を拒否すること。
func TestDeleteWsFedRelyingPartyRefusesTheEntraRelyingParty(t *testing.T) {
	s, _, observations, _ := entraRemovalServer(t)
	recordObservation(t, s, managedRollbackObservation(t, time.Now().UTC()))
	rec := doJSON(s.e, http.MethodDelete, "/api/admin/v1/wsfed/relying-parties?wtrealm="+entraRealm, "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "urn:idmagic:error:entra_relying_party") {
		t.Fatalf("status=%d body=%s, want 409 entra_relying_party", rec.Code, rec.Body.String())
	}
	if rp := entraRelyingParty(t, s); rp == nil || len(rp.Profile.Domains) != 2 {
		t.Fatalf("Entra RP after the refused delete = %+v, want both domains", rp)
	}
	if record, err := observations.Find(context.Background(), tenancydomain.DefaultTenantID, "contoso.com"); err != nil || record == nil {
		t.Fatalf("observation record after the refused delete = %+v, %v; want kept", record, err)
	}
}

// 保存先の障害を、実際のリポジトリと同じ RP の確定の境界で起こす。
type failingRemovalRPs struct {
	wsfedports.WsFedRelyingPartyRepository
	fail error
}

func (r *failingRemovalRPs) Update(ctx context.Context, tenantID, realm string, change wsfedports.RelyingPartyChange) error {
	return r.WsFedRelyingPartyRepository.Update(ctx, tenantID, realm, func(rp *feddomain.WsFedRelyingParty) (wsfedports.RelyingPartyOutcome, error) {
		outcome, err := change(rp)
		if err == nil && outcome != wsfedports.RelyingPartyUnchanged && r.fail != nil {
			return wsfedports.RelyingPartyUnchanged, r.fail
		}
		return outcome, err
	})
}

type failingRemovalObservations struct {
	wsfedports.EntraFederationObservationRepository
	failDelete error
}

func (r *failingRemovalObservations) Delete(ctx context.Context, tenantID, domain string) error {
	if r.failDelete != nil {
		return r.failDelete
	}
	return r.EntraFederationObservationRepository.Delete(ctx, tenantID, domain)
}

//spec:covers REQ-WSFEDERATION-001: 解除の保存に失敗してもドメインの構成を残し、成功のイベントを発行せず、再実行で解除を完了できること。
func TestEntraDomainRemovalCanResumeAfterStorageFailure(t *testing.T) {
	for _, failure := range []string{"observation delete", "RP commit"} {
		t.Run(failure, func(t *testing.T) {
			var rps *failingRemovalRPs
			var observations *failingRemovalObservations
			realms := &fakeRealms{answers: map[string]feddomain.EntraDomainRealm{}}
			s, _ := newEntraVerificationServerWith(t, func(d *httpadapter.Deps) {
				rps = &failingRemovalRPs{WsFedRelyingPartyRepository: d.WsFederation.RPRepo}
				observations = &failingRemovalObservations{EntraFederationObservationRepository: d.WsFederation.ObservationRepo}
				d.WsFederation.RPRepo = rps
				d.WsFederation.ObservationRepo = observations
				d.WsFederation.RealmDiscovery = realms
			}, entraAlice)
			if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
				t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
			}
			recordObservation(t, s, managedRollbackObservation(t, time.Now().UTC()))
			before, _ := json.Marshal(entraRelyingParty(t, s))
			*s.events = nil
			fault := errors.New("storage unavailable")
			if failure == "observation delete" {
				observations.failDelete = fault
			} else {
				rps.fail = fault
			}
			rec := doJSON(s.e, http.MethodDelete, contosoDomainPath, "")
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("failed removal: status=%d body=%s, want 500", rec.Code, rec.Body.String())
			}
			after, _ := json.Marshal(entraRelyingParty(t, s))
			if !bytes.Equal(before, after) || removedEvent(t, *s.events) != nil {
				t.Fatalf("failed removal changed the RP or emitted an event: before=%s after=%s events=%v", before, after, *s.events)
			}
			record, err := observations.Find(context.Background(), tenancydomain.DefaultTenantID, "contoso.com")
			if err != nil || (failure == "observation delete" && record == nil) {
				t.Fatalf("failed observation delete lost the record: %+v, %v", record, err)
			}
			// 外部の切り戻しは済んでいる。保存先の復旧後、探索をやり直して同じドメインを解除する。
			rps.fail, observations.failDelete = nil, nil
			realms.answer("contoso.com", managedRealm)
			removed := decodeBody[removalResult](t, doJSON(s.e, http.MethodDelete, contosoDomainPath, ""), http.StatusOK)
			if !removed.RelyingPartyDeleted || entraRelyingParty(t, s) != nil || removedEvent(t, *s.events) == nil {
				t.Fatalf("retry did not complete the removal: %+v", removed)
			}
		})
	}
}

type realmDiscoveryFunc func(context.Context, string) (feddomain.EntraDomainRealm, error)

func (f realmDiscoveryFunc) Discover(ctx context.Context, domain string) (feddomain.EntraDomainRealm, error) {
	return f(ctx, domain)
}

//spec:covers REQ-WSFEDERATION-001: レルム探索の後に最新の RP を固定して読み直し、探索の間に追加された別のドメインを残して対象だけを外すこと。
func TestEntraDomainRemovalPreservesAConfigurationSavedDuringDiscovery(t *testing.T) {
	var s entraServer
	s, _ = newEntraVerificationServerWith(t, func(d *httpadapter.Deps) {
		d.WsFederation.RealmDiscovery = realmDiscoveryFunc(func(context.Context, string) (feddomain.EntraDomainRealm, error) {
			// 探索の呼び出しの間に別の管理者が保存する操作列を、待機に頼らず再現する。
			if rec := configureEntra(s.e, `{"domain":"fabrikam.com","issuer_uri":"`+fabrikamIssuer+`","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
				t.Fatalf("configure during discovery: status=%d body=%s", rec.Code, rec.Body.String())
			}
			return managedRealm, nil
		})
	}, entraAlice, entraDave)
	if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	removed := decodeBody[removalResult](t, doJSON(s.e, http.MethodDelete, contosoDomainPath, ""), http.StatusOK)
	rp := entraRelyingParty(t, s)
	if removed.RelyingPartyDeleted || !slices.Equal(removed.RemainingDomains, []string{"fabrikam.com"}) ||
		rp == nil || len(rp.Profile.Domains) != 1 || rp.Profile.Domains[0].IssuerURI != fabrikamIssuer {
		t.Fatalf("removal lost the concurrent configuration: removal=%+v RP=%+v", removed, rp)
	}
}
