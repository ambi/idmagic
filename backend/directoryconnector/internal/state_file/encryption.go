package state_file

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/fs"
)

// LoadEncryptionKey は署名鍵の世代に束縛した別の鍵を使う。新しい登録で旧鍵を再利用しない。
func (s *Store) LoadEncryptionKey(signingKeyID string) (*ecdsa.PrivateKey, error) {
	var stored struct {
		SigningKeyID string `json:"signing_key_id"`
		DER          []byte `json:"private_key"`
	}
	err := s.readJSON("encryption-key.json", &stored)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && stored.SigningKeyID == signingKeyID {
		parsed, err := x509.ParsePKCS8PrivateKey(stored.DER)
		if err != nil {
			return nil, err
		}
		key, ok := parsed.(*ecdsa.PrivateKey)
		if !ok || key.Curve != elliptic.P256() {
			return nil, errors.New("encryption key must be P-256")
		}
		return key, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	stored.SigningKeyID = signingKeyID
	stored.DER, err = x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(stored) //nolint:gosec // 暗号化の秘密鍵を所有者だけが読める状態ファイルへ保存する。
	if err != nil {
		return nil, err
	}
	if err := s.replace("encryption-key.json", raw); err != nil {
		return nil, err
	}
	return key, nil
}
