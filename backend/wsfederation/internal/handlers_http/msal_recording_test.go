package handlers_http_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/internal/requests_wstrust"
)

// mustCreated は記録の RST の Timestamp の Created の直後の時刻を返す。
func mustCreated(t *testing.T, body []byte) time.Time {
	t.Helper()
	document := etree.NewDocument()
	if err := document.ReadFromBytes(body); err != nil {
		t.Fatal(err)
	}
	created := document.FindElement("//Timestamp/Created")
	if created == nil {
		t.Fatal("the recorded RST has no Timestamp/Created")
	}
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(created.Text()))
	if err != nil {
		t.Fatal(err)
	}
	return at.Add(time.Second)
}

func loadMSALRecording(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/msal-wstrust.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// 本物の MSAL.NET が送った RST を、記録のときの時刻で解析と検証にかける。
// 受け口は時刻を現在から取るので、HTTP の層での受理（RSTR を返し、MSAL がそれを読んで Entra ID へ進んだこと）は
// 記録のときに観測し、記録の method と応答に残した。ここでは記録のバイト列から読む値を、WS-Trust 1.3 と
// WS-Security の要素の意味から確かめる。期待値は、MSAL が userrealm で受けた値（利用者 alice、AppliesTo の
// cloud_audience_urn）と、IdMagic の MEX が示した能動の STS の住所である。
//
//spec:covers MSAL-WSTRUST-RST: 本物の MSAL.NET 4.92 が送った WS-Trust 1.3 の RST を、XML 宣言、名前空間の宣言の順序、小数の秒を含む Timestamp のまま解析し、利用者、パスワード、AppliesTo、Bearer の KeyType を取り出すこと
func TestParseRST_AcceptsTheRSTThatMSALSent(t *testing.T) {
	rec := loadMSALRecording(t)
	exchange := rec.Exchange(t, "rst")
	if exchange.Response.Status != http.StatusOK {
		t.Fatalf("the recorded RST was answered %d; record it again", exchange.Response.Status)
	}
	if body := string(testing_httprecord.BodyBytes(exchange.Response.Body)); !strings.Contains(body, "RequestSecurityTokenResponse") {
		t.Fatalf("the recorded answer to MSAL is not an RSTR: %.200s", body)
	}
	body := testing_httprecord.BodyBytes(exchange.Request.Body)

	// 解析の時刻は記録の Timestamp の Created の直後に置く。Created は記録の本文から読むが、検証の窓の判定にしか使わない。
	probe, err := wstrust.ParseRST(body, mustCreated(t, body))
	if err != nil {
		t.Fatalf("ParseRST rejected the RST that MSAL sent: %v", err)
	}
	if probe.Username != "alice" || probe.Password != "correct-password" {
		t.Errorf("credentials = %q / %q, want the ones MSAL was given", probe.Username, probe.Password)
	}
	if probe.AppliesTo != passiveRealm {
		t.Errorf("AppliesTo = %q, want the cloud_audience_urn %q", probe.AppliesTo, passiveRealm)
	}
	if probe.KeyType != "http://docs.oasis-open.org/ws-sx/ws-trust/200512/Bearer" || probe.RequestType != msalRequestIssue || probe.Action != msalActionIssue13 {
		t.Errorf("KeyType/RequestType/Action = %q / %q / %q, want WS-Trust 1.3 Bearer Issue", probe.KeyType, probe.RequestType, probe.Action)
	}
	if !strings.HasSuffix(probe.To, exchange.Request.Path) || probe.To == "" {
		t.Errorf("To = %q, want the address MSAL posted to (%s)", probe.To, exchange.Request.Path)
	}

	t.Run("the RST expires with its Timestamp", func(t *testing.T) {
		if _, err := wstrust.ParseRST(body, probe.Expires); err == nil {
			t.Fatal("an RST past its Expires must be rejected")
		}
	})
}

// IdMagic の今の MEX が、MSAL が実際に RST を送った経路を usernamemixed の住所として示し続けることを確かめる。
// 記録の後に MEX の形が変わり、MSAL が別の経路を選ぶようになったら、記録を取り直す。
//
//spec:covers MSAL-WSTRUST-RST: MSAL が実際に選んだ能動の STS の経路を、今の MEX も UsernamePassword の WS-Trust 1.3 のエンドポイントとして示すこと
func TestMEX_StillOffersTheEndpointMSALChose(t *testing.T) {
	rec := loadMSALRecording(t)
	if rec.Exchange(t, "mex").Request.Path != "/realms/default/trust/mex" {
		t.Fatalf("MSAL fetched MEX from %s", rec.Exchange(t, "mex").Request.Path)
	}
	e, _ := newServer(t, nil)
	mex := get(e, "/realms/default/trust/mex")
	if mex.Code != http.StatusOK {
		t.Fatalf("MEX status = %d", mex.Code)
	}
	endpoint, version := msalMexEndpoint(t, mex.Body.String())
	if !strings.HasSuffix(endpoint, rec.Exchange(t, "rst").Request.Path) || version != "13" {
		t.Fatalf("MEX offers %s (%s), want the endpoint MSAL posted to (%s, WS-Trust 1.3)", endpoint, version, rec.Exchange(t, "rst").Request.Path)
	}
}
