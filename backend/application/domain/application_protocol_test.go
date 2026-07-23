package domain

import (
	"errors"
	"testing"
)

func TestValidateApplicationSingleProtocolContract(t *testing.T) {
	validOIDC := &ApplicationProtocol{Type: ApplicationProtocolOIDC, ClientID: "client-1"}
	validSAML := &ApplicationProtocol{Type: ApplicationProtocolSAML, EntityID: "urn:sp"}

	tests := []struct {
		name    string
		app     Application
		wantErr error
	}{
		{
			name: "federated application has one protocol",
			app: Application{
				Name: "portal", Kind: ApplicationFederated, Status: ApplicationActive,
				Protocol: validSAML,
			},
		},
		{
			name: "service application has oidc protocol",
			app: Application{
				Name: "api", Kind: ApplicationService, Status: ApplicationActive,
				Protocol: validOIDC,
			},
		},
		{
			name: "weblink has no protocol",
			app: Application{
				Name: "docs", Kind: ApplicationWeblink, Status: ApplicationActive,
				LaunchURL: "https://example.com",
			},
		},
		{
			name:    "federated requires protocol",
			app:     Application{Name: "portal", Kind: ApplicationFederated, Status: ApplicationActive},
			wantErr: ErrProtocolRequired,
		},
		{
			name: "service rejects non oidc protocol",
			app: Application{
				Name: "api", Kind: ApplicationService, Status: ApplicationActive,
				Protocol: validSAML,
			},
			wantErr: ErrServiceRequiresOIDC,
		},
		{
			name: "weblink rejects protocol",
			app: Application{
				Name: "docs", Kind: ApplicationWeblink, Status: ApplicationActive,
				LaunchURL: "https://example.com", Protocol: validOIDC,
			},
			wantErr: ErrWeblinkHasProtocol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateApplication(&tt.app)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateApplication() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
