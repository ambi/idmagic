package spec

import "testing"

func TestSaltedHashIsDeterministic(t *testing.T) {
	salt := []byte("tenant-a-salt")
	got1 := SaltedHash(salt, "alice")
	got2 := SaltedHash(salt, "alice")
	if got1 != got2 {
		t.Fatalf("SaltedHash not deterministic: %q != %q", got1, got2)
	}
	if len(got1) != 64 { // hex(sha256) は 32 byte = 64 hex 桁
		t.Fatalf("unexpected hash length %d: %q", len(got1), got1)
	}
}

func TestSaltedHashSeparatesTenants(t *testing.T) {
	a := SaltedHash([]byte("tenant-a-salt"), "alice")
	b := SaltedHash([]byte("tenant-b-salt"), "alice")
	if a == b {
		t.Fatalf("same value hashed identically across tenants: %q", a)
	}
}

func TestSaltedHashSeparatesValues(t *testing.T) {
	salt := []byte("salt")
	if SaltedHash(salt, "alice") == SaltedHash(salt, "bob") {
		t.Fatal("distinct values produced the same hash")
	}
}
