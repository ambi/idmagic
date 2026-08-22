package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateApplicationFieldInvariants(t *testing.T) {
	valid := Application{Name: "app", Kind: ApplicationWeblink, Status: ApplicationActive, LaunchURL: "https://example.com"}

	t.Run("rejects an empty name", func(t *testing.T) {
		app := valid
		app.Name = ""
		if err := ValidateApplication(&app); !errors.Is(err, ErrNameRequired) {
			t.Fatalf("err=%v, want ErrNameRequired", err)
		}
	})

	t.Run("rejects an invalid kind", func(t *testing.T) {
		app := valid
		app.Kind = "bogus"
		if err := ValidateApplication(&app); !errors.Is(err, ErrInvalidKind) {
			t.Fatalf("err=%v, want ErrInvalidKind", err)
		}
	})

	t.Run("rejects an invalid status", func(t *testing.T) {
		app := valid
		app.Status = "bogus"
		if err := ValidateApplication(&app); !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("err=%v, want ErrInvalidStatus", err)
		}
	})

	t.Run("weblink requires a launch_url", func(t *testing.T) {
		app := valid
		app.LaunchURL = ""
		if err := ValidateApplication(&app); !errors.Is(err, ErrWeblinkLaunchURL) {
			t.Fatalf("err=%v, want ErrWeblinkLaunchURL", err)
		}
	})
}

func TestValidateProtocol(t *testing.T) {
	t.Run("rejects an invalid protocol type", func(t *testing.T) {
		err := ValidateProtocol(ApplicationProtocol{Type: "bogus"})
		if !errors.Is(err, ErrInvalidProtocolType) {
			t.Fatalf("err=%v, want ErrInvalidProtocolType", err)
		}
	})

	t.Run("oidc requires client_id", func(t *testing.T) {
		err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolOIDC})
		if !errors.Is(err, ErrOIDCClientID) {
			t.Fatalf("err=%v, want ErrOIDCClientID", err)
		}
	})

	t.Run("oidc with client_id is valid", func(t *testing.T) {
		if err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolOIDC, ClientID: "c1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wsfed requires wtrealm", func(t *testing.T) {
		err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolWsFed})
		if !errors.Is(err, ErrWsFedWtrealm) {
			t.Fatalf("err=%v, want ErrWsFedWtrealm", err)
		}
	})

	t.Run("wsfed with wtrealm is valid", func(t *testing.T) {
		if err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolWsFed, Wtrealm: "urn:my-app"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wsfed rejects an oversized wtrealm", func(t *testing.T) {
		huge := strings.Repeat("a", 5000)
		if err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolWsFed, Wtrealm: huge}); err == nil {
			t.Fatal("expected a length error for an oversized wtrealm")
		}
	})

	t.Run("saml requires entity_id", func(t *testing.T) {
		err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolSAML})
		if !errors.Is(err, ErrSAMLEntityID) {
			t.Fatalf("err=%v, want ErrSAMLEntityID", err)
		}
	})

	t.Run("saml with entity_id is valid", func(t *testing.T) {
		if err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolSAML, EntityID: "urn:sp"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("saml rejects an oversized entity_id", func(t *testing.T) {
		huge := strings.Repeat("a", 5000)
		if err := ValidateProtocol(ApplicationProtocol{Type: ApplicationProtocolSAML, EntityID: huge}); err == nil {
			t.Fatal("expected a length error for an oversized entity_id")
		}
	})
}
