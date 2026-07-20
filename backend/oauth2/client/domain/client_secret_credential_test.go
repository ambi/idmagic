package domain

import (
	"testing"
	"time"
)

func TestClientSecretCredentialIsActive(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	hash := HashClientSecret("previous")
	overlap := now.Add(time.Hour)
	revoked := now.Add(-time.Minute)

	tests := []struct {
		name       string
		credential ClientSecretCredential
		want       bool
	}{
		{"current", ClientSecretCredential{SecretHash: hash}, true},
		{"overlap", ClientSecretCredential{SecretHash: hash, ExpiresAt: &overlap}, true},
		{"expired", ClientSecretCredential{SecretHash: hash, ExpiresAt: &now}, false},
		{"revoked", ClientSecretCredential{SecretHash: hash, RevokedAt: &revoked}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.credential.IsActiveAt(now); got != tt.want {
				t.Fatalf("IsActiveAt() = %v, want %v", got, tt.want)
			}
		})
	}
}
