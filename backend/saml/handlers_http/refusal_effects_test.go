package handlers_http_test

// SAML が宣言する拒否について、応答と「拒否が起こさなかったこと」の両方を確かめる。
//
// SAML の拒否が素通りしたときの被害は取り消せない。発行された Assertion は署名付きで
// SP へ渡り、こちらが撤回する手段は無い。だから拒否のテストは「400 が返った」ことでは
// 足りず、SAMLResponse が 1 通も出ていないことまで読む。
//
// レスポンスボディと ACS へのリダイレクトを両方読むのは、拒否の応答を返しながら発行だけ済ませて
// いる実装が、片方だけでは見えないためである。SAML の応答は XML なので、要素の有無は
// 文字列一致ではなく etree で解析してから確かめる — 名前空間の接頭辞の違いで誤って
// 成立させないため。
//
// どの拒否にも、同じ入口で同じ操作が通る対照を 1 つ置く。「拒否されたので何も起きて
// いない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
)

const (
	refusalSPEntityID = "https://sp.example.com"
	refusalSPACSURL   = "https://sp.example.com/acs"
	refusalProfileA   = "profile-a"
	refusalProfileB   = "profile-b"
)

// unreachableKeyStore は外部の鍵 provider が止まった状態を作る。到達不能な provider は
// 健全性も鍵の列挙も鍵の取得も返せないので、3 つとも塞ぐ。
//
// production の `KeyStoreSignerProvider` と両ハンドラーはそのまま経路に残るため、
// 確かめているのは「資格情報が無いときに証明書を返さない」という宣言そのものである。
type unreachableKeyStore struct {
	signingports.KeyStore
}

func (unreachableKeyStore) Healthy(context.Context) bool { return false }

func (unreachableKeyStore) GetActiveKey(context.Context) (*signingdomain.SigningKey, error) {
	return nil, fmt.Errorf("federation key provider is unreachable")
}

func (unreachableKeyStore) ListPublicKeys(context.Context, time.Time) ([]*signingdomain.SigningKey, error) {
	return nil, fmt.Errorf("federation key provider is unreachable")
}

// newProfileBoundServer は SP を profile-a だけに割り当てたサーバーを組み立てる。
// profile-b は同じテナントに存在するが、この SP からは使えない。
func newProfileBoundServer(t *testing.T, reachableKeys bool) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()

	spRepo := samlmemory.NewSamlServiceProviderRepository()
	for _, profileID := range []string{refusalProfileA, refusalProfileB} {
		if err := spRepo.SaveIDPProfile(context.Background(), &samldomain.SamlIdentityProviderProfile{
			TenantID: tenancydomain.DefaultTenantID, ProfileID: profileID,
			Name: profileID, Mode: samldomain.IDPProfileModeDedicated,
		}); err != nil {
			t.Fatal(err)
		}
	}
	spRepo.Seed(&samldomain.SamlServiceProvider{
		TenantID:      tenancydomain.DefaultTenantID,
		EntityID:      refusalSPEntityID,
		IDPProfileID:  refusalProfileA,
		ACSURLs:       []string{refusalSPACSURL},
		SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
			},
		},
	})

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice"})

	memoryKeys, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	var keyStore signingports.KeyStore = memoryKeys
	if !reachableKeys {
		keyStore = unreachableKeyStore{KeyStore: memoryKeys}
	}

	captured := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:   "https://idp.example",
		Contract: spec.CurrentRuntimeContract(),
		Emit:     func(event spec.DomainEvent) { *captured = append(*captured, event) },
		Saml: saml.Module{
			SPRepo: spRepo, ProfileRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore(),
		},
		UserRepo:         userRepo,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		AuthnResolver: stubResolver{ctx: &authdomain.AuthenticationContext{
			UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}},
	})
	return e, captured
}

// profileSSOURL はプロファイルの正規 SSO URL を返す。AuthnRequest の Destination と
// 送信先の両方をここから作るので、テストが実装と別の URL を作ってすれ違うことはない。
func profileSSOURL(profileID string) string {
	return "https://idp.example/realms/default/saml/idp/" + profileID + "/sso"
}

func profileSSOPath(profileID, samlRequest string) string {
	return "/realms/default/saml/idp/" + profileID + "/sso?SAMLRequest=" + url.QueryEscape(samlRequest)
}

// assertNoAssertionIssued は応答が Assertion を 1 つも運んでいないことを確かめる。
// 自動 POST フォームの hidden input、ACS へのリダイレクト、生の XML の 3 経路すべてを塞ぐ。
func assertNoAssertionIssued(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	body := recorder.Body.String()
	if strings.Contains(body, "SAMLResponse") {
		t.Fatalf("拒否されたのに応答が SAMLResponse を運んでいる: %s", body)
	}
	if location := recorder.Header().Get("Location"); strings.Contains(location, refusalSPACSURL) ||
		strings.Contains(location, "SAMLResponse") {
		t.Fatalf("拒否されたのに ACS へのリダイレクトが起きた: %q", location)
	}
	// 名前空間の接頭辞に依存しないよう、本文が XML として読める場合は要素として確かめる。
	document := etree.NewDocument()
	if err := document.ReadFromString(body); err != nil {
		return
	}
	if assertion := document.FindElement("//Assertion"); assertion != nil {
		t.Fatalf("拒否されたのに Assertion 要素が応答に含まれる: %s", body)
	}
}

// EX-SAML-002-02: `profile-a` に割り当てられた SP の AuthnRequest を `profile-b` の
// SSO エンドポイントへ送っても、SAMLResponse は発行されず SamlSignInRejected だけが出る。
//
// この拒否が素通りすれば、SP は自分に割り当てられていないプロファイルの署名資格情報で
// アサーションを受け取れる。プロファイルの分離は、どの鍵で誰に対して何を主張するかの
// 分離そのものなので、素通りは鍵の分離を無効にする。
func TestSamlSSORefusesUnassignedIDPProfileAndIssuesNoAssertion(t *testing.T) {
	e, events := newProfileBoundServer(t, true)

	// profile-b の SSO エンドポイント宛に正しく組み立てた AuthnRequest。Destination は
	// 送信先と一致しているので、拒否の理由はプロファイルの割り当てだけになる。
	request := authnRequestRedirectWith(t, refusalSPEntityID, refusalSPACSURL, profileSSOURL(refusalProfileB), false)
	refused := get(e, profileSSOPath(refusalProfileB, request))

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "not assigned") {
		t.Fatalf("body=%s, want the profile assignment refusal", refused.Body.String())
	}
	assertNoAssertionIssued(t, refused)
	if !hasEvent(*events, "SamlSignInRejected") {
		t.Fatal("SamlSignInRejected が発行されていない")
	}
	if hasEvent(*events, "SamlSignInIssued") {
		t.Fatal("拒否されたのに SamlSignInIssued が発行された")
	}

	// 対照: 同じ SP、同じ利用者、同じ組み立てで、割り当てられた profile-a なら発行される。
	// これが無いと、SSO の配線を落としただけのテストがそのまま緑になる。
	allowed := get(e, profileSSOPath(refusalProfileA,
		authnRequestRedirectWith(t, refusalSPEntityID, refusalSPACSURL, profileSSOURL(refusalProfileA), false)))
	if allowed.Code != http.StatusOK {
		t.Fatalf("対照が status=%d body=%s, want 200", allowed.Code, allowed.Body.String())
	}
	if !strings.Contains(allowed.Body.String(), `name="SAMLResponse"`) {
		t.Fatalf("対照が SAMLResponse を返していない: %s", allowed.Body.String())
	}
}

// EX-SAML-002-03: `profile-a` の SSO URL と異なる Destination を指定した AuthnRequest は
// フェイルクローズで拒否され、アサーションは 1 通も発行されない。
//
// Destination は「このリクエストは自分宛か」を確かめる唯一の手段である。素通りすれば、
// 別の IdP 宛に作られたリクエストに対してこちらが署名付きのアサーションを発行する。
func TestSamlSSORefusesDestinationMismatchAndIssuesNoAssertion(t *testing.T) {
	e, events := newProfileBoundServer(t, true)

	// 送信先は profile-a、Destination は profile-b の SSO URL。どちらも実在する URL なので、
	// 拒否は「存在しない宛先」ではなく Destination の一致判定そのものに由来する。
	request := authnRequestRedirectWith(t, refusalSPEntityID, refusalSPACSURL, profileSSOURL(refusalProfileB), false)
	refused := get(e, profileSSOPath(refusalProfileA, request))

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	assertNoAssertionIssued(t, refused)
	if !hasEvent(*events, "SamlSignInRejected") {
		t.Fatal("SamlSignInRejected が発行されていない")
	}
	if hasEvent(*events, "SamlSignInIssued") {
		t.Fatal("拒否されたのに SamlSignInIssued が発行された")
	}

	// 対照: Destination を profile-a の SSO URL に直すだけで、同じ経路が発行に進む。
	allowed := get(e, profileSSOPath(refusalProfileA,
		authnRequestRedirectWith(t, refusalSPEntityID, refusalSPACSURL, profileSSOURL(refusalProfileA), false)))
	if allowed.Code != http.StatusOK {
		t.Fatalf("対照が status=%d body=%s, want 200", allowed.Code, allowed.Body.String())
	}
	if !strings.Contains(allowed.Body.String(), `name="SAMLResponse"`) {
		t.Fatalf("対照が SAMLResponse を返していない: %s", allowed.Body.String())
	}
}

// EX-SAML-001-02: フェデレーション署名資格情報を利用できないとき、証明書ダウンロードと
// メタデータのどちらも証明書を返さずエラーを返す。
//
// これは可用性ではなくフェイルクローズの宣言である。空の PEM や空の X509Certificate 要素を
// 返せば、受け取った SP は署名検証を諦めるか、検証なしで受理する実装に当たる。
// 「エラーを返すこと」と「空を返さないこと」は別の要求なので、両方を読む。
func TestSamlSigningCertificateRefusesWithoutCredentialsAndReturnsNoCertificate(t *testing.T) {
	e, _ := newProfileBoundServer(t, false)

	certificate := get(e, "/saml/signing-certificate.pem")
	if certificate.Code != http.StatusInternalServerError {
		t.Fatalf("certificate status=%d body=%s, want 500", certificate.Code, certificate.Body.String())
	}
	// 証明書として読めるものが 1 つも返っていないこと。空文字も既定値も PEM も含まない。
	if block, _ := pem.Decode(certificate.Body.Bytes()); block != nil {
		t.Fatalf("資格情報が無いのに PEM ブロックが返った: %s", certificate.Body.String())
	}
	if strings.Contains(certificate.Body.String(), "CERTIFICATE") {
		t.Fatalf("応答が証明書らしきものを運んでいる: %s", certificate.Body.String())
	}
	// ダウンロードとして保存させないこと。空の .pem がディスクに落ちると、
	// 受け取った側は「証明書が空である」ことを設定の完了と取り違える。
	if disposition := certificate.Header().Get("Content-Disposition"); disposition != "" {
		t.Fatalf("Content-Disposition=%q, want none on the refusal", disposition)
	}

	metadata := get(e, "/saml/metadata")
	if metadata.Code != http.StatusInternalServerError {
		t.Fatalf("metadata status=%d body=%s, want 500", metadata.Code, metadata.Body.String())
	}
	// メタデータ側も、証明書の要素そのものが無いことを解析して確かめる。
	document := etree.NewDocument()
	if err := document.ReadFromString(metadata.Body.String()); err == nil {
		if element := document.FindElement("//X509Certificate"); element != nil {
			t.Fatalf("資格情報が無いのにメタデータが X509Certificate を公開した: %s", metadata.Body.String())
		}
	}

	// 対照: 鍵ストアが到達可能なだけで、同じ 2 経路が本物の証明書を返す。
	// 拒否の側だけを読むと、そもそも証明書を返せない配線と区別がつかない。
	reachable, _ := newProfileBoundServer(t, true)
	published := get(reachable, "/saml/signing-certificate.pem")
	if published.Code != http.StatusOK {
		t.Fatalf("対照 certificate status=%d body=%s", published.Code, published.Body.String())
	}
	block, _ := pem.Decode(published.Body.Bytes())
	if block == nil {
		t.Fatalf("対照が PEM を返していない: %s", published.Body.String())
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("対照の証明書を解析できない: %v", err)
	}
	if reachableMetadata := get(reachable, "/saml/metadata"); reachableMetadata.Code != http.StatusOK {
		t.Fatalf("対照 metadata status=%d body=%s", reachableMetadata.Code, reachableMetadata.Body.String())
	}
}
