package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolvableUserEventsDoNotExposeUsernamePayloadFields(t *testing.T) {
	t.Parallel()

	events := []any{
		AuthorizationCodeIssued{},
		AuthorizationCodeRedeemed{},
		AccessTokenIssued{},
		RefreshTokenIssued{},
	}
	for _, event := range events {
		typ := reflect.TypeOf(event)
		for field := range typ.Fields() {
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonName == "username" || jsonName == "usernameHash" {
				t.Errorf("%s must not expose %q; resolve username to userId at query time", typ.Name(), jsonName)
			}
		}
	}
}
