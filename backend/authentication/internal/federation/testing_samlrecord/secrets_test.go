package main

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

func recordingWithResponse(t *testing.T, xml string) testing_httprecord.Recording {
	t.Helper()
	form := url.Values{"SAMLResponse": {base64.StdEncoding.EncodeToString([]byte(xml))}, "RelayState": {RelayState}}
	body, err := json.Marshal(form.Encode())
	if err != nil {
		t.Fatal(err)
	}
	return testing_httprecord.Recording{Exchanges: []testing_httprecord.Exchange{{
		Name: "assertion-signed", Request: testing_httprecord.RecordedRequest{Body: body},
	}}}
}

func TestCheckSecrets_AcceptsPublicValues(t *testing.T) {
	rec := recordingWithResponse(t, `<Response><X509Certificate>MIIB</X509Certificate></Response>`)
	if err := CheckSecrets(rec, "rec-admin", "rec-user", "eyJtoken"); err != nil {
		t.Fatalf("CheckSecrets: %v", err)
	}
}

// 応答の XML は base64 とフォームのエンコードを経るので、記録の JSON をそのまま探しても値は見つからない。
func TestCheckSecrets_FindsCredentialsInsideTheEncodedResponse(t *testing.T) {
	for name, xml := range map[string]string{
		"user password":  `<Response><Attribute>rec-user</Attribute></Response>`,
		"admin token":    `<Response><Extensions>eyJtoken</Extensions></Response>`,
		"private key":    `<Response>-----BEGIN PRIVATE KEY-----</Response>`,
		"admin password": `<Response>rec-admin</Response>`,
	} {
		t.Run(name, func(t *testing.T) {
			if err := CheckSecrets(recordingWithResponse(t, xml), "rec-admin", "rec-user", "eyJtoken"); err == nil {
				t.Fatal("CheckSecrets accepted a recording that contains a secret")
			}
		})
	}
}

func TestCheckSecrets_FindsCredentialsInTheRecordingText(t *testing.T) {
	rec := recordingWithResponse(t, `<Response/>`)
	rec.Method = "logged in with rec-user"
	if err := CheckSecrets(rec, "rec-user"); err == nil {
		t.Fatal("CheckSecrets accepted a recording whose method text contains the password")
	}
}
