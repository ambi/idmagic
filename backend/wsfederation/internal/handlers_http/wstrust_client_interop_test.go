package handlers_http_test

// 能動的な STS が、MSAL が送る RST を受け、MSAL が MEX から usernamemixed のエンドポイントを選べる
// ことを観測する。RST の形と MEX の選択の手順は、MSAL.NET の WsTrustEndpoint.cs と MexDocument.cs
// （src/client/Microsoft.Identity.Client/WsTrust）から写した。

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

const (
	msalActionIssue13 = "http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue"
	msalRequestIssue  = "http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue"
)

// msalRST は、MSAL の WsTrustEndpoint.BuildTokenRequestMessage が WS-Trust 1.3 の
// UsernameToken で組み立てる RST と同じ要素、名前空間、順序の RST を作る。
func msalRST(now time.Time, action, appliesTo string) string {
	stamp := func(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.068Z") }
	return `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:wsa="http://www.w3.org/2005/08/addressing" xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">` +
		`<s:Header>` +
		`<wsa:Action s:mustUnderstand="1">` + action + `</wsa:Action>` +
		`<wsa:MessageID>urn:uuid:5b1a3f0e-6c1d-4a5e-9a3c-2f8e7d6c5b4a</wsa:MessageID>` +
		`<wsa:ReplyTo><wsa:Address>http://www.w3.org/2005/08/addressing/anonymous</wsa:Address></wsa:ReplyTo>` +
		`<wsa:To s:mustUnderstand="1">https://idp.example/realms/default/trust/usernamemixed</wsa:To>` +
		`<wsse:Security s:mustUnderstand="1" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">` +
		`<wsu:Timestamp wsu:Id="MSATimeStamp"><wsu:Created>` + stamp(now) + `</wsu:Created><wsu:Expires>` + stamp(now.Add(10*time.Minute)) + `</wsu:Expires></wsu:Timestamp>` +
		`<wsse:UsernameToken wsu:Id="UnPwSecTok13-0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"><wsse:Username>alice</wsse:Username><wsse:Password>correct-password</wsse:Password></wsse:UsernameToken>` +
		`</wsse:Security>` +
		`</s:Header>` +
		`<s:Body>` +
		`<wst:RequestSecurityToken xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512">` +
		`<wsp:AppliesTo xmlns:wsp="http://schemas.xmlsoap.org/ws/2004/09/policy"><wsa:EndpointReference><wsa:Address>` + appliesTo + `</wsa:Address></wsa:EndpointReference></wsp:AppliesTo>` +
		`<wst:KeyType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Bearer</wst:KeyType>` +
		`<wst:RequestType>` + msalRequestIssue + `</wst:RequestType>` +
		`</wst:RequestSecurityToken>` +
		`</s:Body>` +
		`</s:Envelope>`
}

//spec:covers REQ-WSFEDERATION-005: Action が WS-Trust 1.3 の Issue の要求（.../RST/Issue）である MSAL の形の RST に RSTR を返し、RequestType の値（.../Issue）を Action に置いた RST は 400 と WsTrustTokenRejected で拒否してトークンを発行しないこと。
func TestWsTrustIssue_AcceptsTheRSTThatMSALSends(t *testing.T) {
	e, events := newServer(t, nil)
	now := time.Now().UTC()

	accepted := postWsTrustSOAP(e, msalRST(now, msalActionIssue13, passiveRealm))
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), "RequestSecurityTokenResponse") {
		t.Fatalf("MSAL RST status=%d body=%s, want 200 with an RSTR", accepted.Code, accepted.Body.String())
	}
	if !hasEvent(*events, "WsTrustTokenIssued") {
		t.Fatal("WsTrustTokenIssued not emitted for the MSAL RST")
	}

	issuedBefore := countEvents(*events, "WsTrustTokenIssued")
	refused := postWsTrustSOAP(e, strings.Replace(msalRST(now, msalRequestIssue, passiveRealm),
		"5b1a3f0e-6c1d-4a5e-9a3c-2f8e7d6c5b4a", "9c8b7a6f-5e4d-4c3b-8a29-1f0e0d0c0b0a", 1))
	if refused.Code != http.StatusBadRequest || !hasEvent(*events, "WsTrustTokenRejected") {
		t.Fatalf("RequestType as Action status=%d body=%s, want 400 and WsTrustTokenRejected", refused.Code, refused.Body.String())
	}
	assertNoWsTrustTokenIssued(t, refused, *events, issuedBefore)
}

// msalMexEndpoint は、MSAL の MexDocument が UsernamePassword に選ぶ WS-Trust の
// エンドポイントと版を返す。方針、束縛、ポートの順にたどる。
func msalMexEndpoint(t *testing.T, body string) (endpoint, version string) {
	t.Helper()
	const (
		nsWsdl    = "http://schemas.xmlsoap.org/wsdl/"
		nsWsp     = "http://schemas.xmlsoap.org/ws/2004/09/policy"
		nsSp      = "http://docs.oasis-open.org/ws-sx/ws-securitypolicy/200702"
		nsSp2005  = "http://schemas.xmlsoap.org/ws/2005/07/securitypolicy"
		nsSoap12  = "http://schemas.xmlsoap.org/wsdl/soap12/"
		nsWsa10   = "http://www.w3.org/2005/08/addressing"
		nsWsu     = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
		issue2005 = "http://schemas.xmlsoap.org/ws/2005/02/trust/RST/Issue"
		transport = "http://schemas.xmlsoap.org/soap/http"
	)
	doc := etree.NewDocument()
	if err := doc.ReadFromString(body); err != nil {
		t.Fatalf("parse MEX: %v", err)
	}
	children := func(parent *etree.Element, ns, local string) []*etree.Element {
		var out []*etree.Element
		for _, child := range parent.ChildElements() {
			if child.Tag == local && child.NamespaceURI() == ns {
				out = append(out, child)
			}
		}
		return out
	}
	var descendants func(parent *etree.Element, ns, local string) []*etree.Element
	descendants = func(parent *etree.Element, ns, local string) []*etree.Element {
		var out []*etree.Element
		for _, child := range parent.ChildElements() {
			if child.Tag == local && child.NamespaceURI() == ns {
				out = append(out, child)
			}
			out = append(out, descendants(child, ns, local)...)
		}
		return out
	}
	first := func(elements []*etree.Element) *etree.Element {
		if len(elements) == 0 {
			return nil
		}
		return elements[0]
	}
	attr := func(el *etree.Element, ns, local string) string {
		for _, a := range el.Attr {
			if a.Key == local && a.NamespaceURI() == ns {
				return a.Value
			}
		}
		return ""
	}

	root := doc.Root()
	usernamePolicies := map[string]bool{}
	for _, policy := range descendants(root, nsWsp, "Policy") {
		exactlyOne := first(children(policy, nsWsp, "ExactlyOne"))
		if exactlyOne == nil {
			continue
		}
		for _, all := range descendants(exactlyOne, nsWsp, "All") {
			auth := first(children(all, nsSp, "SignedEncryptedSupportingTokens"))
			if auth == nil {
				auth = first(children(all, nsSp2005, "SignedSupportingTokens"))
			}
			if auth == nil {
				continue
			}
			inner := first(children(auth, nsWsp, "Policy"))
			if inner == nil {
				continue
			}
			token := first(children(inner, nsSp2005, "UsernameToken"))
			if token == nil {
				continue
			}
			tokenPolicy := first(children(token, nsWsp, "Policy"))
			if tokenPolicy == nil || first(children(tokenPolicy, nsSp2005, "WssUsernameToken10")) == nil {
				continue
			}
			hasTransport := len(descendants(policy, nsSp, "TransportBinding")) > 0 || len(descendants(policy, nsSp2005, "TransportBinding")) > 0
			if id := attr(policy, nsWsu, "Id"); hasTransport && id != "" {
				usernamePolicies["#"+id] = true
			}
		}
	}

	bindingVersions := map[string]string{}
	for _, binding := range descendants(root, nsWsdl, "binding") {
		for _, reference := range children(binding, nsWsp, "PolicyReference") {
			name := binding.SelectAttrValue("name", "")
			if !usernamePolicies[reference.SelectAttrValue("URI", "")] || name == "" {
				continue
			}
			operation := first(children(binding, nsWsdl, "operation"))
			if operation == nil {
				continue
			}
			soapOperation := first(children(operation, nsSoap12, "operation"))
			if soapOperation == nil {
				continue
			}
			action := soapOperation.SelectAttrValue("soapAction", "")
			if !strings.EqualFold(action, msalActionIssue13) && !strings.EqualFold(action, issue2005) {
				continue
			}
			soapBinding := first(children(binding, nsSoap12, "binding"))
			if soapBinding == nil || !strings.EqualFold(soapBinding.SelectAttrValue("transport", ""), transport) {
				continue
			}
			bindingVersions[name] = map[bool]string{true: "2005", false: "13"}[strings.EqualFold(action, issue2005)]
		}
	}

	service := first(descendants(root, nsWsdl, "service"))
	if service == nil {
		t.Fatalf("MEX has no wsdl:service:\n%s", body)
	}
	for _, port := range children(service, nsWsdl, "port") {
		_, bindingName, ok := strings.Cut(port.SelectAttrValue("binding", ""), ":")
		portVersion, known := bindingVersions[bindingName]
		if !ok || !known {
			continue
		}
		reference := first(children(port, nsWsa10, "EndpointReference"))
		if reference == nil {
			continue
		}
		address := first(children(reference, nsWsa10, "Address"))
		if address != nil && strings.HasPrefix(strings.ToLower(address.Text()), "https://") {
			if endpoint == "" || portVersion == "13" {
				endpoint, version = address.Text(), portVersion
			}
		}
	}
	return endpoint, version
}

//verify:operation WsTrustMetadataExchange: MSAL の MexDocument と同じ手順で、方針、束縛、ポートをたどり、usernamemixed の WS-Trust 1.3 のエンドポイントを選べる。
func TestWsTrustMEX_LetsMSALFindTheUsernameMixedEndpoint(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{})
	rec := get(e, "/realms/default/trust/mex")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	endpoint, version := msalMexEndpoint(t, rec.Body.String())
	if endpoint != "https://idp.example/realms/default/trust/usernamemixed" || version != "13" {
		t.Fatalf("MSAL selected (%q, WS-Trust %q), want the WS-Trust 1.3 usernamemixed endpoint:\n%s", endpoint, version, rec.Body.String())
	}
}
