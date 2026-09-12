package handlers_http_test

// docs/contexts/saml/scenarios.feature.md が宣言する具体例を、製品の正式な入口から観測する。
//
// 入口は `httpadapter.Register` が組み立てた `/saml/*` と `/api/admin/v1/saml/*` である。
// 具体例 1 件につきテスト 1 本を置き、その具体例の `Then` の数だけ観測を書く。`Then` が
// 「発行しない」と「イベントを出す」の 2 つを言っているなら、観測も 2 つ要る。片方だけを
// 読むテストは、拒否の応答を返しながら発行だけ済ませている実装を通してしまう。
//
// 既にテストがある具体例は、そのテストへ注記を足して当該ファイルに残す。ここに集めるのは、
// テストが無かった件と、既存テストが具体例の一部しか読んでいなかった件である。

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"

	"github.com/ambi/idmagic/backend/application"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
	dsig "github.com/russellhaering/goxmldsig"
)

// =====================================================================
// 組み立て
// =====================================================================

// newServerWithKeyStore は newServer と同じスタックを、鍵ストアを呼び出し側へ渡す形で組む。
// 鍵のローテートを起こせるのは鍵ストアを持っている側だけなので、証明書の公開を読む具体例は
// この形でなければ観測できない。
func newServerWithKeyStore(t *testing.T) (*echo.Echo, *keys_memory.InMemoryKeyStore) {
	t.Helper()
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID:      "https://sp.example.com",
		ACSURLs:       []string{"https://sp.example.com/acs"},
		SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice"})
	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		Saml:             saml.Module{SPRepo: spRepo, ProfileRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         userRepo,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		AuthnResolver:    stubResolver{ctx: nil},
	})
	return e, keyStore
}

// newServerBehindAnApplication は SP を Application の SAML binding に属させたスタックを組む。
// 割当が無い利用者に対して発行してはならないという判定は、Application が存在するときだけ
// 効く。Application を持たない組み立てでは、この分岐そのものが経路に無い。
func newServerBehindAnApplication(t *testing.T, assign bool) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID:      "https://sp.example.com",
		ACSURLs:       []string{"https://sp.example.com/acs"},
		SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice"})

	now := time.Now().UTC()
	appRepo := appmemory.NewApplicationRepository()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: "app-1", Name: "SP", Kind: appdomain.ApplicationFederated,
		Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now,
		Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolSAML, EntityID: "https://sp.example.com"},
	}
	if err := appRepo.Save(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	assignments := appmemory.NewApplicationAssignmentRepository()
	if assign {
		if err := assignments.Save(context.Background(), &appdomain.ApplicationAssignment{
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: app.ID,
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user-1",
			Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	captured := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		Emit:             func(event spec.DomainEvent) { *captured = append(*captured, event) },
		Saml:             saml.Module{SPRepo: spRepo, ProfileRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		Application:      application.Module{Repo: appRepo, AssignmentRepo: assignments},
		UserRepo:         userRepo,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		AuthnResolver: stubResolver{ctx: &authdomain.AuthenticationContext{
			UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}},
	})
	return e, captured
}

// federationKeyContext はデフォルトプロファイルのフェデレーション署名鍵を指す context を組む。
// 鍵集合はテナント、用途、スコープの 3 つで決まるので、どれか 1 つでも欠けるとテストは
// ハンドラーが読むのとは別の鍵集合を回すことになる。`withUsage` は用途を自分で足す
// 必要のある呼び出し（`Rotate`）で真にする。`Resolve` は用途を自分で足す。
func federationKeyContext(withUsage bool) context.Context {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
	}, "https://idp.example/realms/default", "/realms/default")
	ctx = samltoken.WithSignerScope(ctx, samldomain.DefaultIDPProfileID)
	if withUsage {
		ctx = signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
	}
	return ctx
}

// certificateFromPEMResponse は PEM の応答を証明書として読む。読めないものを「証明書が
// 返っている」と数えないために、都度ここを通す。
func certificateFromPEMResponse(t *testing.T, body []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(body)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("PEM の証明書ではない: %s", body)
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return certificate
}

// publishedCertificates はメタデータが公開している X509Certificate をすべて返す。
// 「移行期間中に信頼するすべての証明書」は集合の話なので、1 つ見つけたら十分ということはない。
func publishedCertificates(t *testing.T, metadataXML []byte) map[string]bool {
	t.Helper()
	document := etree.NewDocument()
	if err := document.ReadFromBytes(metadataXML); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	out := map[string]bool{}
	for _, element := range document.FindElements("//X509Certificate") {
		out[strings.Join(strings.Fields(element.Text()), "")] = true
	}
	return out
}

// =====================================================================
// REQ-SAML-001 SP は署名証明書を取得できる
// =====================================================================

// EX-SAML-001-01: 証明書ダウンロード URL は「現在有効な」`XmlFederationSigning` 証明書を PEM で
// 返し、同じ時点のメタデータがその証明書を公開し、ローテーションの移行期間中に信頼すべき証明書は
// すべてメタデータから取れる。
//
// 3 つの `Then` に 3 つの観測を置く。ダウンロードとメタデータの一致だけを読むと、ローテート中に
// 旧証明書をメタデータから落とす実装が通る。落とすと、旧鍵で署名済みのアサーションを
// 受け取った SP は検証鍵を得られず、移行期間そのものが成立しない。
func TestSamlSigningCertificateIsTheActiveCredentialAndMetadataCarriesEveryTrustedOne(t *testing.T) {
	e, keyStore := newServerWithKeyStore(t)

	// 1. ダウンロードは PEM の証明書を返し、それが鍵ストアの現在有効な資格情報である。
	download := get(e, "/saml/signing-certificate.pem")
	if download.Code != http.StatusOK {
		t.Fatalf("certificate status=%d body=%s", download.Code, download.Body.String())
	}
	before := certificateFromPEMResponse(t, download.Body.Bytes())
	active, err := samltoken.KeyStoreSignerProvider{KeyStore: keyStore}.Resolve(federationKeyContext(false))
	if err != nil {
		t.Fatal(err)
	}
	if !before.Equal(active.Certificate()) {
		t.Fatal("ダウンロードした証明書が、現在有効なフェデレーション署名資格情報ではない")
	}

	// 2. 同じ時点のメタデータが、その証明書を公開している。
	metadata := get(e, "/saml/metadata")
	if metadata.Code != http.StatusOK {
		t.Fatalf("metadata status=%d body=%s", metadata.Code, metadata.Body.String())
	}
	beforeEncoded := base64.StdEncoding.EncodeToString(before.Raw)
	if !publishedCertificates(t, metadata.Body.Bytes())[beforeEncoded] {
		t.Fatalf("ダウンロードした証明書がメタデータに無い:\n%s", metadata.Body.String())
	}

	// 3. ローテートさせると、ダウンロードは新しい有効証明書へ移り、メタデータは移行期間中の
	//    旧証明書も併せて公開する。両方を読まないと、ローテートで旧鍵の検証手段が消える実装と
	//    区別できない。
	if _, err := keyStore.Rotate(federationKeyContext(true), time.Now().UTC(), time.Hour); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	rotatedDownload := get(e, "/saml/signing-certificate.pem")
	after := certificateFromPEMResponse(t, rotatedDownload.Body.Bytes())
	if after.Equal(before) {
		t.Fatal("ローテートしたのにダウンロードが旧証明書を返している")
	}
	rotatedMetadata := get(e, "/saml/metadata")
	published := publishedCertificates(t, rotatedMetadata.Body.Bytes())
	if !published[base64.StdEncoding.EncodeToString(after.Raw)] {
		t.Fatalf("ローテート後の有効証明書がメタデータに無い:\n%s", rotatedMetadata.Body.String())
	}
	if !published[beforeEncoded] {
		t.Fatalf("移行期間中の旧証明書がメタデータから消えている:\n%s", rotatedMetadata.Body.String())
	}
}

// =====================================================================
// REQ-SAML-002 / REQ-SAML-003 IdP プロファイル
// =====================================================================

// EX-SAML-002-01: `profile-a` に割り当てられた SP が `profile-a` の SSO エンドポイントへ送った
// AuthnRequest には、`profile-a` の entityID と `profile-a` の署名資格情報で SAMLResponse が返る。
//
// Destination、SP の Issuer、プロファイルとの関連付けの 3 つが揃ったときにだけ発行されることの、
// 発行側の観測である。3 つのうち 1 つを崩した場合の拒否は refusal_effects_test.go が持つ。
// 発行された Response の Issuer と署名鍵の両方を読むのは、正しいプロファイルへ返しながら
// デフォルトプロファイルの鍵で署名する実装を、Issuer だけでは見分けられないためである。
func TestSamlSSOIssuesWithTheAssignedProfileEntityIDAndCredentials(t *testing.T) {
	e, _ := newProfileBoundServer(t, true)

	request := authnRequestRedirectWith(t, refusalSPEntityID, refusalSPACSURL, profileSSOURL(refusalProfileA), false)
	issued := get(e, profileSSOPath(refusalProfileA, request))
	if issued.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", issued.Code, issued.Body.String())
	}

	document := etree.NewDocument()
	if err := document.ReadFromBytes(samlResponseFromPostForm(t, issued.Body.String())); err != nil {
		t.Fatalf("parse SAMLResponse: %v", err)
	}
	const profileEntityID = "https://idp.example/realms/default/saml/idp/" + refusalProfileA
	if issuer := document.FindElement("/Response/Issuer"); issuer == nil || issuer.Text() != profileEntityID {
		t.Fatalf("Response Issuer=%v, want %q", issuer, profileEntityID)
	}
	assertion := document.FindElement("//Assertion")
	if assertion == nil {
		t.Fatalf("Assertion missing: %s", issued.Body.String())
	}

	// 署名は profile-a が公開する証明書で検証でき、デフォルトプロファイルの証明書では検証できない。
	if err := validateAgainst(assertion, profileCertificate(t, e, "/saml/idp/"+refusalProfileA+"/signing-certificate.pem")); err != nil {
		t.Fatalf("profile-a の証明書で assertion 署名が検証できない: %v", err)
	}
	if err := validateAgainst(assertion, profileCertificate(t, e, "/saml/signing-certificate.pem")); err == nil {
		t.Fatal("デフォルトプロファイルの証明書で assertion 署名が検証できてしまう")
	}
}

func profileCertificate(t *testing.T, e *echo.Echo, path string) *x509.Certificate {
	t.Helper()
	recorder := get(e, path)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
	}
	return certificateFromPEMResponse(t, recorder.Body.Bytes())
}

func validateAgainst(assertion *etree.Element, certificate *x509.Certificate) error {
	store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{certificate}}
	validation := dsig.NewDefaultValidationContext(store)
	validation.IdAttribute = "ID"
	_, err := validation.Validate(assertion.Copy())
	return err
}

// EX-SAML-003-01: 専用プロファイルのメタデータは、そのプロファイル固有の entityID、
// SSO / SLO URL、署名証明書を公開し、デフォルトプロファイルのメタデータは別の署名資格情報を公開する。
//
// 「固有である」ことの観測は、2 つのプロファイルのメタデータを並べて値が違うことである。
// 片方だけを読むと、URL にプロファイル ID を差し込んでいるだけで署名鍵は共有している実装を
// 通してしまう。鍵を共有していれば、プロファイルを分ける目的そのものが失われる。
func TestSamlDedicatedProfilePublishesItsOwnEndpointsAndSigningCredential(t *testing.T) {
	e, _ := newProfileBoundServer(t, true)

	dedicated := get(e, "/saml/idp/"+refusalProfileA+"/metadata")
	if dedicated.Code != http.StatusOK {
		t.Fatalf("profile metadata status=%d body=%s", dedicated.Code, dedicated.Body.String())
	}
	for _, want := range []string{
		`entityID="https://idp.example/realms/default/saml/idp/` + refusalProfileA + `"`,
		"https://idp.example/realms/default/saml/idp/" + refusalProfileA + "/sso",
		"https://idp.example/realms/default/saml/idp/" + refusalProfileA + "/slo",
	} {
		if !strings.Contains(dedicated.Body.String(), want) {
			t.Fatalf("専用プロファイルのメタデータに %q が無い:\n%s", want, dedicated.Body.String())
		}
	}
	dedicatedCertificate := profileCertificate(t, e, "/saml/idp/"+refusalProfileA+"/signing-certificate.pem")
	if !publishedCertificates(t, dedicated.Body.Bytes())[base64.StdEncoding.EncodeToString(dedicatedCertificate.Raw)] {
		t.Fatalf("専用プロファイルの署名証明書が自分のメタデータに無い:\n%s", dedicated.Body.String())
	}

	// デフォルトプロファイルは別の署名資格情報を公開する。
	defaultMetadata := get(e, "/saml/metadata")
	if defaultMetadata.Code != http.StatusOK {
		t.Fatalf("default metadata status=%d body=%s", defaultMetadata.Code, defaultMetadata.Body.String())
	}
	if publishedCertificates(t, defaultMetadata.Body.Bytes())[base64.StdEncoding.EncodeToString(dedicatedCertificate.Raw)] {
		t.Fatal("デフォルトプロファイルのメタデータが専用プロファイルの署名証明書を公開している")
	}
}

// EX-SAML-003-02: 存在しないプロファイル ID と、別テナントに属するプロファイル ID は、
// どちらも not found を返し、メタデータも証明書も公開しない。
//
// 状態コードだけでなく本文も読む。404 を返しながらデフォルトプロファイルのメタデータを本文に
// 載せる実装は、状態コードだけでは見分けられない。テナント越えのほうは、プロファイル ID を
// 知っているだけの相手に別テナントの entityID と署名証明書を渡さないことそのものである。
func TestSamlProfileEndpointsRefuseAnUnknownOrForeignProfileID(t *testing.T) {
	e, repo := newProfileBoundServerWithRepository(t)

	const foreignProfileID = "foreign-profile"
	if err := repo.SaveIDPProfile(context.Background(), &samldomain.SamlIdentityProviderProfile{
		TenantID: "another-tenant", ProfileID: foreignProfileID,
		Name: "Foreign", Mode: samldomain.IDPProfileModeDedicated,
	}); err != nil {
		t.Fatal(err)
	}

	for _, profileID := range []string{"no-such-profile", foreignProfileID} {
		for _, suffix := range []string{"/metadata", "/signing-certificate.pem"} {
			path := "/saml/idp/" + profileID + suffix
			recorder := get(e, path)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("%s status=%d body=%s, want 404", path, recorder.Code, recorder.Body.String())
			}
			body := recorder.Body.String()
			if strings.Contains(body, "IDPSSODescriptor") || strings.Contains(body, "X509Certificate") {
				t.Fatalf("%s が拒否しながらメタデータを載せている: %s", path, body)
			}
			if block, _ := pem.Decode(recorder.Body.Bytes()); block != nil {
				t.Fatalf("%s が拒否しながら PEM を載せている: %s", path, body)
			}
		}
	}

	// 対照: 同じテナントの実在するプロファイルなら、同じ 2 つの URL が公開する。
	if recorder := get(e, "/saml/idp/"+refusalProfileA+"/metadata"); recorder.Code != http.StatusOK {
		t.Fatalf("対照の metadata status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := get(e, "/saml/idp/"+refusalProfileA+"/signing-certificate.pem"); recorder.Code != http.StatusOK {
		t.Fatalf("対照の certificate status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

// =====================================================================
// REQ-SAML-006 / REQ-SAML-007 SP 起点 SSO
// =====================================================================

// EX-SAML-006-02 / EX-SAML-007-01: entityID、ACS、Destination、対象者の割り当てのいずれかが
// 不正なら、SAMLResponse を発行せず SamlSignInRejected を発行してフェイルクローズで拒否する。
//
// 4 つの次元を 1 本で回すのは、どれか 1 つだけを読むテストが「その次元だけ検査する実装」を
// 通してしまうためである。次元ごとに、応答が Assertion を運んでいないことと、
// SamlSignInRejected が出て SamlSignInIssued が出ていないことの 2 つを読む。
// 拒否のたびに対照を置き、同じ入口で無傷の要求が発行に進むことを確かめる。
func TestSamlSSOFailsClosedOnEveryInvalidRequestDimension(t *testing.T) {
	authenticated := &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}

	for _, tc := range []struct {
		name    string
		refused func(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, string)
	}{
		{
			// 登録されていない entityID。素通りすれば、誰でも自分の ACS 宛に
			// 署名済みアサーションを受け取れる。
			name: "unregistered entityID",
			refused: func(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, string) {
				t.Helper()
				e, events := newServer(t, authenticated)
				return e, events, "/saml/sso?SAMLRequest=" + url.QueryEscape(
					authnRequestRedirect(t, "https://evil.example.com", "https://evil.example.com/acs"))
			},
		},
		{
			// 登録集合の外にある ACS。素通りは、そのままオープンリダイレクトになる。
			name: "assertion consumer service URL outside the registered set",
			refused: func(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, string) {
				t.Helper()
				e, events := newServer(t, authenticated)
				return e, events, "/saml/sso?SAMLRequest=" + url.QueryEscape(
					authnRequestRedirect(t, "https://sp.example.com", "https://evil.example.com/steal"))
			},
		},
		{
			// この IdP 宛ではない Destination。素通りは、他所宛の要求へこちらが署名する。
			name: "destination that is not this SSO endpoint",
			refused: func(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, string) {
				t.Helper()
				e, events := newServer(t, authenticated)
				return e, events, "/saml/sso?SAMLRequest=" + url.QueryEscape(authnRequestRedirectWith(
					t, "https://sp.example.com", "https://sp.example.com/acs",
					"https://evil-idp.example/saml/sso", false))
			},
		},
		{
			// Application に属する SP へ、割り当てられていない利用者が入ろうとする。
			name: "subject not assigned to the application",
			refused: func(t *testing.T) (*echo.Echo, *[]spec.DomainEvent, string) {
				t.Helper()
				e, events := newServerBehindAnApplication(t, false)
				return e, events, "/saml/sso?SAMLRequest=" + url.QueryEscape(
					authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, events, target := tc.refused(t)
			recorder := get(e, target)
			if recorder.Code < 400 {
				t.Fatalf("status=%d body=%s, want a refusal", recorder.Code, recorder.Body.String())
			}
			assertNoAssertionIssued(t, recorder)
			if !hasEvent(*events, "SamlSignInRejected") {
				t.Fatal("SamlSignInRejected が発行されていない")
			}
			if hasEvent(*events, "SamlSignInIssued") {
				t.Fatal("拒否されたのに SamlSignInIssued が発行された")
			}
		})
	}

	// 対照: 同じ入口で、4 つの次元をすべて満たす要求は発行に進む。これが無いと、
	// 上の 4 件は SSO を配線していないスタックでもそのまま緑になる。
	t.Run("control: an intact request still issues", func(t *testing.T) {
		e, events := newServerBehindAnApplication(t, true)
		recorder := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(
			authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs")))
		if !issuedAnAssertion(t, recorder) {
			t.Fatalf("対照で Assertion が発行されない: status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if !hasEvent(*events, "SamlSignInIssued") {
			t.Fatal("対照で SamlSignInIssued が発行されていない")
		}
	})
}

// EX-SAML-006-03: AuthnRequest の解析または署名検証に失敗したら、SamlSignInRejected を発行して
// プロトコルエラーを返す。
//
// 解析の失敗と署名検証の失敗は別の入口条件なので、2 つとも通す。どちらも「Assertion が
// 1 通も出ていないこと」と「SamlSignInRejected が出ていること」を読む。署名検証のほうには
// 対照を置く: 同じ署名の無い要求でも、検証を要求していない SP なら通る。
func TestSamlSSORejectsUnparsableAndUnverifiableAuthnRequests(t *testing.T) {
	t.Run("the AuthnRequest does not parse", func(t *testing.T) {
		e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})
		// deflate として展開できないバイト列。SAMLRequest の形は満たすが中身が要求ではない。
		recorder := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("not a deflated AuthnRequest"))))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s, want 400", recorder.Code, recorder.Body.String())
		}
		assertNoAssertionIssued(t, recorder)
		if !hasEvent(*events, "SamlSignInRejected") {
			t.Fatal("SamlSignInRejected が発行されていない")
		}
	})

	t.Run("the required AuthnRequest signature is absent", func(t *testing.T) {
		requiring := newServerRequiringSignedAuthnRequests(t)
		recorder := redirectSSO(t, requiring, authnRequestOptions{acsURL: "https://sp.example.com/acs"})
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s, want 400", recorder.Code, recorder.Body.String())
		}
		assertNoAssertionIssued(t, recorder)

		// 対照: 検証を要求していない SP では、同じ署名の無い要求が発行に進む。
		permissive, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})
		if !issuedAnAssertion(t, redirectSSO(t, permissive, authnRequestOptions{acsURL: "https://sp.example.com/acs"})) {
			t.Fatal("対照で Assertion が発行されない")
		}
	})
}

// EX-SAML-006-06: 同じテナント、SP、AuthnRequest ID の組み合わせに対する Assertion が発行済みなら、
// Assertion を発行せず SamlSignInRejected を発行してフェイルクローズで拒否する。
//
// リプレイの観測は同じ要求を 2 回送ることでしかできない。1 回目が発行することと 2 回目が
// 発行しないことを 1 本で読むので、対照は本体に含まれている。ID を変えた 3 回目を足すのは、
// 2 回目の拒否が「2 回目だから」ではなく「同じ ID だから」であることを示すためである。
func TestSamlSSORefusesAReplayedAuthnRequestIDAndIssuesNoSecondAssertion(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	replayed := authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs")
	first := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(replayed))
	if !issuedAnAssertion(t, first) {
		t.Fatalf("1 回目で Assertion が発行されない: status=%d body=%s", first.Code, first.Body.String())
	}
	if !hasEvent(*events, "SamlSignInIssued") {
		t.Fatal("1 回目の SamlSignInIssued が発行されていない")
	}

	second := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(replayed))
	if second.Code != http.StatusBadRequest {
		t.Fatalf("2 回目 status=%d body=%s, want 400", second.Code, second.Body.String())
	}
	assertNoAssertionIssued(t, second)
	if !hasEvent(*events, "SamlSignInRejected") {
		t.Fatal("2 回目の SamlSignInRejected が発行されていない")
	}

	// 3 回目は ID だけを変えた同じ要求。ここが通ることで、2 回目の拒否が ID の再利用に
	// 由来すると言える。
	fresh := redirectSSO(t, e, authnRequestOptions{acsURL: "https://sp.example.com/acs"})
	if !issuedAnAssertion(t, fresh) {
		t.Fatalf("ID を変えた要求が拒否された: status=%d body=%s", fresh.Code, fresh.Body.String())
	}
}

// =====================================================================
// REQ-SAML-004 管理者は SAML IdP プロファイルを管理できる
// =====================================================================

type adminIDPProfile struct {
	Profile              samldomain.SamlIdentityProviderProfile `json:"profile"`
	EntityID             string                                 `json:"entity_id"`
	MetadataURL          string                                 `json:"metadata_url"`
	ServiceProviderCount int                                    `json:"service_provider_count"`
}

func createIDPProfile(t *testing.T, e *echo.Echo, name string, mode samldomain.IDPProfileMode) adminIDPProfile {
	t.Helper()
	recorder := doAdminJSON(e, http.MethodPost, "/api/admin/v1/saml/idp-profiles",
		`{"name":"`+name+`","mode":"`+string(mode)+`"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create profile status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var created adminIDPProfile
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	return created
}

func profileBoundServiceProviderBody(entityID, profileID string) string {
	return `{"entity_id":"` + entityID + `","idp_profile_id":"` + profileID +
		`","acs_urls":["` + entityID + `/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",` +
		`"source_attribute":"user_id"}}}`
}

func listedServiceProviders(t *testing.T, e *echo.Echo) string {
	t.Helper()
	recorder := get(e, "/api/admin/v1/saml/service-providers")
	if recorder.Code != http.StatusOK {
		t.Fatalf("list service providers status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

// EX-SAML-004-01: 管理者は追加プロファイルを作成し、名前とモードを変更し、`shared` を複数の SP に、
// `dedicated` を 1 つの SP に割り当て、未使用の追加プロファイルを削除できる。
//
// 具体例の `Then` は 5 つある。作成の応答だけを読むと、保存されないまま応答だけ返す実装が
// 通るので、変更と割り当てはいずれも保存側から読み直す。画面遷移の `Then`（一覧と詳細が
// 表示される）は frontend/src/features/admin-saml-idp-profiles が同じ id で持つ。
func TestAdminManagesSharedAndDedicatedIDPProfilesEndToEnd(t *testing.T) {
	e := newAdminServer(t)

	// 変更できない default の shared プロファイルが最初から在る。
	list := get(e, "/api/admin/v1/saml/idp-profiles")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"profile_id":"default"`) {
		t.Fatalf("default profile is missing: status=%d body=%s", list.Code, list.Body.String())
	}

	// shared プロファイルは複数の SP から選べる。2 つ目が保存されることまで読む。
	shared := createIDPProfile(t, e, "Partners", samldomain.IDPProfileModeShared)
	for _, entityID := range []string{"https://sp-one.example.com", "https://sp-two.example.com"} {
		if recorder := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
			profileBoundServiceProviderBody(entityID, shared.Profile.ProfileID)); recorder.Code != http.StatusCreated {
			t.Fatalf("%s の登録 status=%d body=%s", entityID, recorder.Code, recorder.Body.String())
		}
	}
	listed := listedServiceProviders(t, e)
	for _, entityID := range []string{"https://sp-one.example.com", "https://sp-two.example.com"} {
		if !strings.Contains(listed, `"entity_id":"`+entityID+`"`) ||
			!strings.Contains(listed, `"idp_profile_id":"`+shared.Profile.ProfileID+`"`) {
			t.Fatalf("shared プロファイルへの割り当てが保存されていない: %s", listed)
		}
	}

	// 追加プロファイルの名前とモードの変更が保存される。
	updated := doAdminJSON(e, http.MethodPut, "/api/admin/v1/saml/idp-profiles/"+shared.Profile.ProfileID,
		`{"name":"Renamed partners","mode":"shared"}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	if reread := get(e, "/api/admin/v1/saml/idp-profiles"); !strings.Contains(reread.Body.String(), "Renamed partners") {
		t.Fatalf("変更が保存されていない: %s", reread.Body.String())
	}

	// dedicated プロファイルと 1 つの SP の関連付けが保存される。
	dedicated := createIDPProfile(t, e, "One partner", samldomain.IDPProfileModeDedicated)
	if recorder := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
		profileBoundServiceProviderBody("https://sp-solo.example.com", dedicated.Profile.ProfileID)); recorder.Code != http.StatusCreated {
		t.Fatalf("dedicated への割り当て status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(listedServiceProviders(t, e), `"idp_profile_id":"`+dedicated.Profile.ProfileID+`"`) {
		t.Fatal("dedicated プロファイルへの関連付けが保存されていない")
	}

	// 未使用の追加プロファイルは削除でき、一覧から消える。
	unused := createIDPProfile(t, e, "Unused", samldomain.IDPProfileModeShared)
	if recorder := doAdminJSON(e, http.MethodDelete, "/api/admin/v1/saml/idp-profiles/"+unused.Profile.ProfileID, ""); recorder.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if remaining := get(e, "/api/admin/v1/saml/idp-profiles"); strings.Contains(remaining.Body.String(), unused.Profile.ProfileID) {
		t.Fatalf("削除したプロファイルが一覧に残っている: %s", remaining.Body.String())
	}
}

// EX-SAML-004-02: `dedicated` プロファイルを別の SP にも割り当てる要求は、InvalidRequestError で
// 拒否される。
//
// 契約 `RegisterSamlServiceProvider` は 400 `InvalidRequestError` を宣言し、500 を宣言していない。
// 状態コードと本文の型に加えて、2 つ目の SP が保存されていないことまで読む。拒否の応答を
// 返しながら保存だけ済ませる実装は、応答だけでは見分けられない。専用プロファイルの意味は
// 「この鍵と entityID を使うのはこの SP だけ」なので、保存されてしまえば分離は失われている。
func TestAdminServiceProviderRefusesASecondBindingToADedicatedProfile(t *testing.T) {
	e := newAdminServer(t)
	dedicated := createIDPProfile(t, e, "Only one", samldomain.IDPProfileModeDedicated)

	first := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
		profileBoundServiceProviderBody("https://sp-first.example.com", dedicated.Profile.ProfileID))
	if first.Code != http.StatusCreated {
		t.Fatalf("1 つ目の割り当て status=%d body=%s", first.Code, first.Body.String())
	}

	second := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
		profileBoundServiceProviderBody("https://sp-second.example.com", dedicated.Profile.ProfileID))
	if second.Code != http.StatusBadRequest {
		t.Fatalf("2 つ目の割り当て status=%d body=%s, want 400", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), "urn:idmagic:error:invalid_request") {
		t.Fatalf("2 つ目の割り当ての本文=%s, want invalid_request", second.Body.String())
	}
	if listed := listedServiceProviders(t, e); strings.Contains(listed, "https://sp-second.example.com") {
		t.Fatalf("拒否されたのに 2 つ目の SP が保存されている: %s", listed)
	}

	// 対照: 同じ 2 つ目の要求でも、shared プロファイル宛なら受理される。これが無いと、
	// 上の拒否が基数の規則ではなく登録そのものの失敗でも緑になる。
	shared := createIDPProfile(t, e, "Many", samldomain.IDPProfileModeShared)
	if recorder := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
		profileBoundServiceProviderBody("https://sp-second.example.com", shared.Profile.ProfileID)); recorder.Code != http.StatusCreated {
		t.Fatalf("対照 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

// EX-SAML-004-03: SP から参照されているプロファイルとデフォルトプロファイルは、削除を conflict で
// 拒否される。
//
// 2 つの条件は別の理由で同じ答えを返すので、両方を通す。どちらも、拒否のあとにプロファイルが
// 一覧へ残っていることまで読む。参照されたまま消えれば、その SP の SSO は次の要求から
// プロファイル未解決で落ちる。デフォルトが消えれば、テナントの SAML そのものが入口を失う。
func TestAdminIDPProfileDeletionIsRefusedWhileReferencedOrDefault(t *testing.T) {
	e := newAdminServer(t)
	referenced := createIDPProfile(t, e, "Referenced", samldomain.IDPProfileModeShared)
	if recorder := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers",
		profileBoundServiceProviderBody("https://sp-bound.example.com", referenced.Profile.ProfileID)); recorder.Code != http.StatusCreated {
		t.Fatalf("SP の登録 status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	for _, profileID := range []string{referenced.Profile.ProfileID, samldomain.DefaultIDPProfileID} {
		recorder := doAdminJSON(e, http.MethodDelete, "/api/admin/v1/saml/idp-profiles/"+profileID, "")
		if recorder.Code != http.StatusConflict {
			t.Fatalf("%s の削除 status=%d body=%s, want 409", profileID, recorder.Code, recorder.Body.String())
		}
		remaining := get(e, "/api/admin/v1/saml/idp-profiles")
		if !strings.Contains(remaining.Body.String(), `"profile_id":"`+profileID+`"`) {
			t.Fatalf("拒否されたのに %s が一覧から消えている: %s", profileID, remaining.Body.String())
		}
	}

	// 対照: 参照を外せば同じプロファイルが削除できる。これが無いと、追加プロファイルを
	// 一切削除できない実装でも上の 2 件は緑になる。
	if recorder := doJSON(e, http.MethodDelete,
		"/api/admin/v1/saml/service-providers?entity_id="+url.QueryEscape("https://sp-bound.example.com"), ""); recorder.Code != http.StatusNoContent {
		t.Fatalf("SP の削除 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := doAdminJSON(e, http.MethodDelete, "/api/admin/v1/saml/idp-profiles/"+referenced.Profile.ProfileID, ""); recorder.Code != http.StatusNoContent {
		t.Fatalf("対照の削除 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
