// Package state_file は、Connector の秘密鍵、登録の情報、差分の位置を状態のディレクトリのファイルに置く。
// どのファイルも所有者だけが読み書きでき、置き換えは一時ファイルからの rename で行う。
package state_file

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

const (
	keyFile        = "connector-key.pem"
	enrollmentFile = "enrollment.json"
	cursorFile     = "cursor.json"
)

// ErrNotEnrolled は、登録の前に同期を始めたときに返す。
var ErrNotEnrolled = errors.New("the connector is not enrolled; run the enroll command first")

type Store struct{ dir string }

func New(dir string) *Store { return &Store{dir: dir} }

func (s *Store) SaveCredentials(_ context.Context, key *ecdsa.PrivateKey, enrollment ports.Enrollment) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(enrollment)
	if err != nil {
		return err
	}
	if err := s.replace(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})); err != nil {
		return err
	}
	return s.replace(enrollmentFile, encoded)
}

func (s *Store) LoadCredentials() (*ecdsa.PrivateKey, ports.Enrollment, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, keyFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ports.Enrollment{}, ErrNotEnrolled
	}
	if err != nil {
		return nil, ports.Enrollment{}, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, ports.Enrollment{}, fmt.Errorf("%s is not PEM", keyFile)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, ports.Enrollment{}, fmt.Errorf("%s: %w", keyFile, err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, ports.Enrollment{}, fmt.Errorf("%s is not an EC key", keyFile)
	}
	var enrollment ports.Enrollment
	if err := s.readJSON(enrollmentFile, &enrollment); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ports.Enrollment{}, ErrNotEnrolled
		}
		return nil, ports.Enrollment{}, err
	}
	return key, enrollment, nil
}

func (s *Store) Load(context.Context) (*domain.Cursor, error) {
	var cursor domain.Cursor
	if err := s.readJSON(cursorFile, &cursor); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil //nolint:nilnil // 位置がないことは完全な再同期の合図である。
		}
		return nil, err
	}
	return &cursor, nil
}

func (s *Store) Save(_ context.Context, cursor domain.Cursor) error {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	return s.replace(cursorFile, encoded)
}

func (s *Store) readJSON(name string, into any) error {
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// replace は content を一時ファイルに書いてから name へ rename する。途中で止まっても古い内容が残る。
func (s *Store) replace(name string, content []byte) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.dir, "."+name+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), filepath.Join(s.dir, name))
}
