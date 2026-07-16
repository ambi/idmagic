package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestAuthenticationEventPIIFieldsMatchPlaintextContract(t *testing.T) {
	t.Parallel()

	assertJSONFields(t, UserAuthenticated{}, []string{
		"tenantId", "userId", "amr", "sessionId", "clientId", "acr", "ip", "userAgent",
		"countryCode", "deviceFingerprint", "riskScore",
	})
	assertJSONFields(t, AuthenticationFailed{}, []string{
		"tenantId", "username", "reason", "sessionId", "clientId", "ip", "userAgent",
		"countryCode", "deviceFingerprint", "riskScore",
	})
	assertJSONFields(t, SessionStarted{}, []string{
		"tenantId", "userId", "sessionId", "amr", "acr", "ip", "userAgent",
	})
	assertJSONFields(t, AuthenticationStepFailed{}, []string{
		"tenantId", "step", "reason",
	})
}

func assertJSONFields(t *testing.T, event any, want []string) {
	t.Helper()

	typ := reflect.TypeOf(event)
	got := make([]string, 0, typ.NumField())
	for field := range typ.Fields() {
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonName != "-" {
			got = append(got, jsonName)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s JSON fields = %v, want %v", typ.Name(), got, want)
	}
}
