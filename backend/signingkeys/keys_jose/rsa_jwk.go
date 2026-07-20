package keys_jose

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

func GenerateRSAJWKPair() (*rsa.PrivateKey, map[string]any, map[string]any, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, "", err
	}
	publicJWK := PublicJWK(&priv.PublicKey)
	kid, err := Thumbprint(publicJWK)
	if err != nil {
		return nil, nil, nil, "", err
	}
	publicJWK["kid"] = kid
	publicJWK["alg"] = string(signingdomain.SigAlgPS256)
	publicJWK["use"] = "sig"
	privateJWK := map[string]any{
		"kty": "RSA",
		"kid": kid,
		"alg": string(signingdomain.SigAlgPS256),
		"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntFromInt(priv.E)),
		"d":   base64.RawURLEncoding.EncodeToString(priv.D.Bytes()),
		"p":   base64.RawURLEncoding.EncodeToString(priv.Primes[0].Bytes()),
		"q":   base64.RawURLEncoding.EncodeToString(priv.Primes[1].Bytes()),
	}
	return priv, publicJWK, privateJWK, kid, nil
}

func ImportRSAJWK(publicJWK, privateJWK map[string]any) (crypto.PublicKey, crypto.PrivateKey, error) {
	pub, err := publicKeyFromJWK(publicJWK)
	if err != nil {
		return nil, nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("public JWK is not RSA")
	}
	decodeInt := func(name string) (*big.Int, error) {
		value, _ := privateJWK[name].(string)
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil || len(decoded) == 0 {
			return nil, errors.New("private JWK missing or invalid " + name)
		}
		return new(big.Int).SetBytes(decoded), nil
	}
	d, err := decodeInt("d")
	if err != nil {
		return nil, nil, err
	}
	p, err := decodeInt("p")
	if err != nil {
		return nil, nil, err
	}
	q, err := decodeInt("q")
	if err != nil {
		return nil, nil, err
	}
	priv := &rsa.PrivateKey{PublicKey: *rsaPub, D: d, Primes: []*big.Int{p, q}}
	if err := priv.Validate(); err != nil {
		return nil, nil, err
	}
	priv.Precompute()
	return rsaPub, priv, nil
}

func publicKeyFromJWK(jwk map[string]any) (crypto.PublicKey, error) {
	if kty, _ := jwk["kty"].(string); kty != "RSA" {
		return nil, errors.New("public JWK is not RSA")
	}
	nValue, _ := jwk["n"].(string)
	eValue, _ := jwk["e"].(string)
	nBytes, err := base64.RawURLEncoding.DecodeString(nValue)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eValue)
	if err != nil {
		return nil, err
	}
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 | int(b)
	}
	if len(nBytes) == 0 || exponent == 0 {
		return nil, errors.New("public JWK missing RSA modulus or exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

func PublicJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntFromInt(pub.E)),
	}
}

func bigIntFromInt(value int) []byte {
	return new(big.Int).SetInt64(int64(value)).Bytes()
}

func Thumbprint(jwk map[string]any) (string, error) {
	required := []string{"e", "kty", "n"}
	canonical := map[string]any{}
	for _, key := range required {
		value, ok := jwk[key]
		if !ok {
			return "", errors.New("jwk missing required member: " + key)
		}
		canonical[key] = value
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
