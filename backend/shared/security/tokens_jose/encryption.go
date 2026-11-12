package tokens_jose

import (
	"crypto/ecdsa"
	"errors"

	"github.com/go-jose/go-jose/v4"
)

// ValidateEncryptionJWK は署名鍵と独立した公開 P-256 鍵だけを暗号化に使う。
func ValidateEncryptionJWK(jwk map[string]any) (string, error) {
	if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
		return "", errors.New("encryption requires a public P-256 key")
	}
	return ValidatePublicSigningJWK(jwk)
}

// EncryptCompact は暗号文の形式とアルゴリズムを固定する。呼び出し元が payload の束縛を所有する。
func EncryptCompact(jwk map[string]any, payload []byte) (string, error) {
	kid, err := ValidateEncryptionJWK(jwk)
	if err != nil {
		return "", err
	}
	key, err := publicKeyFromJWK(jwk)
	if err != nil {
		return "", err
	}
	encrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.ECDH_ES, Key: key, KeyID: kid}, nil)
	if err != nil {
		return "", err
	}
	sealed, err := encrypter.Encrypt(payload)
	if err != nil {
		return "", err
	}
	return sealed.CompactSerialize()
}

// DecryptCompact は圧縮と別の鍵への差し替えを拒否し、解析器の処理量を制限する。
func DecryptCompact(key *ecdsa.PrivateKey, kid, ciphertext string) ([]byte, error) {
	if len(ciphertext) > 32<<10 {
		return nil, errors.New("encrypted request is too large")
	}
	sealed, err := jose.ParseEncryptedCompact(ciphertext, []jose.KeyAlgorithm{jose.ECDH_ES}, []jose.ContentEncryption{jose.A256GCM})
	if err != nil {
		return nil, errors.New("invalid encrypted request")
	}
	if sealed.Header.KeyID != kid || sealed.Header.ExtraHeaders[jose.HeaderKey("zip")] != nil {
		return nil, errors.New("invalid encrypted request binding")
	}
	return sealed.Decrypt(key)
}
