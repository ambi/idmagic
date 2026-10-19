package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type credentialPurgeRecorder struct {
	name  string
	calls *[]string
	fail  *string
	err   error
}

func (r credentialPurgeRecorder) DeleteAllForSub(_ context.Context, sub string) error {
	*r.calls = append(*r.calls, r.name+":"+sub)
	if *r.fail == r.name {
		return r.err
	}
	return nil
}

//spec:covers REQ-IDMANAGEMENT-050: 資格情報の削除を既存順に行い、途中失敗を返し、同じ利用者の削除を再実行できる。
func TestUserCredentialsPurgeStopsAtEachFailureAndRetries(t *testing.T) {
	names := []string{"history", "totp", "devices", "webauthn", "recovery"}
	for failedIndex, name := range names {
		t.Run(name, func(t *testing.T) {
			var calls []string
			fail := name
			failure := errors.New("credential deletion failed")
			recorder := func(name string) credentialPurgeRecorder {
				return credentialPurgeRecorder{name: name, calls: &calls, fail: &fail, err: failure}
			}
			credentials := UserCredentials{
				PasswordHistory: recorder("history"), TOTP: recorder("totp"), TrustedDevices: recorder("devices"),
				WebAuthn: recorder("webauthn"), RecoveryCodes: recorder("recovery"),
			}
			if err := credentials.PurgeUserCredentials(context.Background(), "target"); !errors.Is(err, failure) {
				t.Fatalf("purge error = %v, want %v", err, failure)
			}
			var expected []string
			for _, name := range names[:failedIndex+1] {
				expected = append(expected, name+":target")
			}
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("calls = %v, want %v", calls, expected)
			}
			fail = ""
			calls = nil
			if err := credentials.PurgeUserCredentials(context.Background(), "target"); err != nil {
				t.Fatal(err)
			}
			expected = nil
			for _, name := range names {
				expected = append(expected, name+":target")
			}
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("retry calls = %v, want %v", calls, expected)
			}
		})
	}
}

func TestUserCredentialsPurgePreservesPartialConfigurations(t *testing.T) {
	var calls []string
	fail := ""
	credentials := UserCredentials{WebAuthn: credentialPurgeRecorder{name: "webauthn", calls: &calls, fail: &fail}}
	if err := credentials.PurgeUserCredentials(context.Background(), "target"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"webauthn:target"}) {
		t.Fatalf("calls = %v, want only target's WebAuthn", calls)
	}
}
