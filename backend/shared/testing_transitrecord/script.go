package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// product は記録する相手と、その往復の手順である。
type product struct {
	title       string
	binary      string
	counterpart string
	method      string
	output      string
	script      func(r *recorder) ([]testing_httprecord.Exchange, error)
}

const devServerConditions = "開発用サーバー（-dev、インメモリの保存、root トークン、監査デバイス、名前空間、ポリシーなし）を起動し、transit に Transit を有効にしてから往復を送った。" +
	"本番の構成（監査デバイス、名前空間、ポリシー、HA の待機系）での応答は保証しない。X-Vault-Token は記録に使った使い捨ての root トークンを recorded-root-token へ置き換えた。"

var products = map[string]product{
	"vault": {
		title:       "Vault",
		binary:      "vault",
		counterpart: "HashiCorp Vault Transit secrets engine",
		method:      "mise run record-transit -- vault が " + devServerConditions + "要求は SigningKeys の HTTPTransitEngine と同じ本文とヘッダーで送り、順序は VaultKeyStore の初回取得、PS256 の署名、ローテーション、回転後の署名に合わせた。",
		output:      "backend/signingkeys/internal/keys_vault/testdata/vault-transit.json",
		script:      recordVault,
	},
	"openbao": {
		title:       "OpenBao",
		binary:      "bao",
		counterpart: "OpenBao Transit secrets engine",
		method:      "mise run record-transit -- openbao が " + devServerConditions + "要求は envelope_openbao の HTTPTransitEngine と同じ本文とヘッダーで送り、順序は OpenBaoMasterKeyProvider のラップと戻しに合わせた。回転と min_decryption_version の設定は運用者の操作であり、アダプターは送らない。",
		output:      "backend/shared/security/envelope_openbao/testdata/openbao-transit.json",
		script:      recordOpenBao,
	},
}

// VaultSigningKey は VaultKeyStore が既定の接頭辞、テナント tenant-a、用途 signing、既定のスコープに使う鍵名である。
const VaultSigningKey = "idmagic-signing-tenant-a-signing-default"

// OpenBaoDataKey は OpenBaoMasterKeyProvider が既定の接頭辞とテナント tenant-a に使う鍵名である。
const OpenBaoDataKey = "idmagic-datakeys-tenant-a"

func recordVault(r *recorder) ([]testing_httprecord.Exchange, error) {
	keys := "/v1/transit/keys/" + VaultSigningKey
	sign := "/v1/transit/sign/" + VaultSigningKey
	digest := sha256.Sum256([]byte("idmagic transit recording"))
	input := base64.StdEncoding.EncodeToString(digest[:])
	steps := []step{
		{http.StatusNotFound, "read-missing-key", "EnsureKey の読み取り。鍵はまだない。", http.MethodGet, keys, nil},
		{http.StatusOK, "create-key", "EnsureKey の作成。", http.MethodPost, keys, map[string]any{"type": "rsa-2048"}},
		{http.StatusOK, "read-key-v1", "LatestPublicKey。", http.MethodGet, keys, nil},
		{http.StatusOK, "sign-v1-pss", "PS256 の署名（PSSSaltLengthEqualsHash）。", http.MethodPost, sign, map[string]any{
			"input": input, "prehashed": true, "signature_algorithm": "pss", "salt_length": "hash", "hash_algorithm": "sha2-256", "key_version": 1,
		}},
		{http.StatusOK, "read-key-before-rotation", "ローテーションの EnsureKey の読み取り。鍵はある。", http.MethodGet, keys, nil},
		{http.StatusOK, "rotate-key", "RotateKey。", http.MethodPost, keys + "/rotate", nil},
		{http.StatusOK, "read-key-v2", "回転後の LatestPublicKey。", http.MethodGet, keys, nil},
		{http.StatusOK, "sign-v2-pkcs1v15", "回転後の鍵での PKCS #1 v1.5 の署名（XML の署名と RS256）。", http.MethodPost, sign, map[string]any{
			"input": input, "prehashed": true, "signature_algorithm": "pkcs1v15", "hash_algorithm": "sha2-256", "key_version": 2,
		}},
	}
	return r.run(steps)
}

func recordOpenBao(r *recorder) ([]testing_httprecord.Exchange, error) {
	keys := "/v1/transit/keys/" + OpenBaoDataKey
	encrypt := "/v1/transit/encrypt/" + OpenBaoDataKey
	decrypt := "/v1/transit/decrypt/" + OpenBaoDataKey
	plaintext := map[string]any{"plaintext": base64.StdEncoding.EncodeToString(FixtureDEK)}

	var out []testing_httprecord.Exchange
	first, err := r.run([]step{
		{http.StatusNotFound, "read-missing-key", "EnsureKey の読み取り。鍵はまだない。", http.MethodGet, keys, nil},
		{http.StatusOK, "create-key", "EnsureKey の作成。", http.MethodPost, keys, map[string]any{"type": "aes256-gcm96"}},
		{http.StatusOK, "encrypt-v1", "EncryptDataKey。", http.MethodPost, encrypt, plaintext},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, first...)
	v1, err := ciphertext(first[2])
	if err != nil {
		return nil, err
	}
	second, err := r.run([]step{
		{http.StatusOK, "decrypt-v1", "DecryptDataKey。", http.MethodPost, decrypt, map[string]any{"ciphertext": v1}},
		{http.StatusOK, "rotate-key", "運用者の鍵の回転。アダプターは送らない。", http.MethodPost, keys + "/rotate", nil},
		{http.StatusOK, "encrypt-v2", "回転後の EncryptDataKey。", http.MethodPost, encrypt, plaintext},
		{http.StatusOK, "decrypt-v1-after-rotation", "回転前の暗号文の DecryptDataKey。", http.MethodPost, decrypt, map[string]any{"ciphertext": v1}},
		{http.StatusOK, "raise-min-decryption-version", "運用者の min_decryption_version の引き上げ。アダプターは送らない。", http.MethodPost, keys + "/config", map[string]any{"min_decryption_version": 2}},
		{http.StatusBadRequest, "decrypt-v1-below-min-version", "min_decryption_version より古い暗号文の DecryptDataKey。", http.MethodPost, decrypt, map[string]any{"ciphertext": v1}},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, second...)
	v2, err := ciphertext(second[2])
	if err != nil {
		return nil, err
	}
	last, err := r.run([]step{
		{http.StatusOK, "decrypt-v2", "min_decryption_version 以上の暗号文の DecryptDataKey。", http.MethodPost, decrypt, map[string]any{"ciphertext": v2}},
	})
	return append(out, last...), err
}

// step は記録する一つの往復と、期待する状態コードである。
type step struct {
	status  int
	name    string
	source  string
	method  string
	path    string
	payload any
}

func (r *recorder) run(steps []step) ([]testing_httprecord.Exchange, error) {
	out := make([]testing_httprecord.Exchange, 0, len(steps))
	for _, s := range steps {
		ex, err := r.expect(s.status, s.name, s.source, s.method, s.path, s.payload)
		if err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, nil
}

func ciphertext(ex testing_httprecord.Exchange) (string, error) {
	var body struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ex.Response.Body, &body); err != nil || body.Data.Ciphertext == "" {
		return "", fmt.Errorf("%s: no ciphertext in %s", ex.Name, ex.Response.Body)
	}
	return body.Data.Ciphertext, nil
}
