package usecases

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestBuildAuthenticationEventAttributes(t *testing.T) {
	salt := []byte("tenant-a")
	got := BuildAuthenticationEventAttributes(salt, " Alice ", "203.0.113.9", "UA/1")
	if got.UsernameHash != spec.SaltedHash(salt, "alice") {
		t.Fatalf("username hash = %q", got.UsernameHash)
	}
	if got.IPTruncated != "203.0.113.0/24" {
		t.Fatalf("ip truncated = %q", got.IPTruncated)
	}
	if got.IPHash != spec.SaltedHash(salt, "203.0.113.9") {
		t.Fatalf("ip hash = %q", got.IPHash)
	}
	if got.UAHash != spec.SaltedHash(salt, "UA/1") {
		t.Fatalf("ua hash = %q", got.UAHash)
	}
}

func TestBuildAuthenticationEventAttributesSeparatesTenants(t *testing.T) {
	a := BuildAuthenticationEventAttributes([]byte("tenant-a"), "alice", "203.0.113.9", "")
	b := BuildAuthenticationEventAttributes([]byte("tenant-b"), "alice", "203.0.113.9", "")
	if a.UsernameHash == b.UsernameHash {
		t.Fatal("username hash must differ by tenant salt")
	}
	if a.IPHash == b.IPHash {
		t.Fatal("ip hash must differ by tenant salt")
	}
	if a.IPTruncated != b.IPTruncated {
		t.Fatal("ip truncation should not depend on tenant salt")
	}
}
