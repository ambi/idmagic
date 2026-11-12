package state_file_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/state_file"
)

//spec:covers REQ-DIRECTORYCONNECTOR-007: 暗号化秘密鍵を所有者限定で保持し、署名用の登録の更新で別の鍵へ置き換える。
func TestEncryptionKeyIsBoundToTheSigningRegistration(t *testing.T) {
	dir := t.TempDir()
	store := state_file.New(dir)
	first, err := store.LoadEncryptionKey("first-signing-key")
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.LoadEncryptionKey("first-signing-key")
	if err != nil || !first.Equal(again) {
		t.Fatalf("key changed on restart: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "encryption-key.json"))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("permissions: %v %v", info, err)
		}
	}
	rotated, err := store.LoadEncryptionKey("second-signing-key")
	if err != nil || first.Equal(rotated) {
		t.Fatalf("old encryption key survived enrollment: %v", err)
	}
}
