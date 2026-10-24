package state_file_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/state_file"
)

//spec:covers REQ-DIRECTORYCONNECTOR-001: 秘密鍵と登録の応答を所有者だけが読み書きできるファイルに保存し、読み戻せる。
func TestStoreKeepsTheCredentialsOwnerOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	store := state_file.New(dir)
	if _, _, err := store.LoadCredentials(); !errors.Is(err, state_file.ErrNotEnrolled) {
		t.Fatalf("before enrollment: %v", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	enrollment := ports.Enrollment{ConnectorID: "c1", KeyID: "k1", Audience: "https://idp.test/realms/a/directory-connector/v1"}
	if err := store.SaveCredentials(context.Background(), key, enrollment); err != nil {
		t.Fatal(err)
	}
	loaded, got, err := store.LoadCredentials()
	if err != nil || !loaded.Equal(key) || got != enrollment {
		t.Fatalf("loaded key=%v enrollment=%+v err=%v", loaded != nil, got, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	for _, name := range []string{"connector-key.pem", "enrollment.json"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %v err=%v, want 0600", name, info.Mode().Perm(), err)
		}
	}
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0o700 {
		t.Fatalf("state dir mode = %v, want 0700", info.Mode().Perm())
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-003: 差分の位置を保存して読み戻し、位置のない状態を nil として返す。
func TestStoreRoundTripsTheCursor(t *testing.T) {
	store := state_file.New(t.TempDir())
	if cursor, err := store.Load(context.Background()); err != nil || cursor != nil {
		t.Fatalf("empty cursor = %+v err=%v", cursor, err)
	}
	want := domain.Cursor{
		Dialect: domain.ActiveDirectory, Scope: domain.Scope{BaseDNs: []string{"OU=Staff", "OU=Contractors"}, Filter: "(objectClass=user)"},
		Server: "CN=NTDS Settings,CN=DC1", USN: 9000, LastFullAt: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
	}
	if err := store.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background())
	if err != nil || got == nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("cursor = %+v err=%v, want %+v", got, err, want)
	}
}
