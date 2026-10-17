package tokens_jose

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

// SignPS256 は鍵の所有モジュールを知らずに JWT へ署名する。
// crypto.Signer を使うため、ローカル鍵とリモートの署名器を同じ経路で扱う。
func SignPS256(privateKey any, kid string, extraHeader map[string]string, claims map[string]any) (string, error) {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return "", errors.New("active key has no signer")
	}
	header := map[string]any{"alg": "PS256", "kid": kid}
	for k, v := range extraHeader {
		header[k] = v
	}
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := signer.Sign(rand.Reader, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
