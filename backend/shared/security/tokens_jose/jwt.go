package tokens_jose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"strings"
	"time"
)

// es256CoordinateBytes は ES256 の JWS 署名で r と s のそれぞれが占める長さ (RFC 7518 §3.4)。
const es256CoordinateBytes = 32

// SignJWT は alg（RS256、PS256、ES256）で JWT に署名する。header の alg は alg で上書きする。
// crypto.Signer を使うため、ローカル鍵とリモートの署名器を同じ経路で扱う。
func SignJWT(privateKey any, alg string, header, claims map[string]any) (string, error) {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return "", errors.New("active key has no signer")
	}
	if alg != "RS256" && alg != "PS256" && alg != "ES256" {
		return "", fmt.Errorf("unsupported alg %q", alg)
	}
	protected := make(map[string]any, len(header)+1)
	maps.Copy(protected, header)
	protected["alg"] = alg
	hb, err := json.Marshal(protected)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := signDigest(signer, alg, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func signDigest(signer crypto.Signer, alg string, digest []byte) ([]byte, error) {
	switch alg {
	case "RS256":
		return signer.Sign(rand.Reader, digest, crypto.SHA256)
	case "PS256":
		return signer.Sign(rand.Reader, digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	}
	// ES256: 固定長の r || s は P-256 でだけ成り立つ。crypto.Signer は ASN.1 DER を返すので変換する。
	if public, ok := signer.Public().(*ecdsa.PublicKey); !ok || public.Curve != elliptic.P256() {
		return nil, errors.New("ES256 requires a P-256 key")
	}
	der, err := signer.Sign(rand.Reader, digest, crypto.SHA256)
	if err != nil {
		return nil, err
	}
	var rs struct{ R, S *big.Int }
	if rest, err := asn1.Unmarshal(der, &rs); err != nil || len(rest) != 0 {
		return nil, errors.New("ES256 signer returned a malformed signature")
	}
	sig := make([]byte, 2*es256CoordinateBytes)
	rs.R.FillBytes(sig[:es256CoordinateBytes])
	rs.S.FillBytes(sig[es256CoordinateBytes:])
	return sig, nil
}

// JWTExpectation は VerifyJWT がクレームへ求める条件。空の Issuer と Audience は照合しない。
// Now は exp と nbf の比較に使う現在時刻で、呼び出し側が渡す。
type JWTExpectation struct {
	Issuer            string
	Audience          string
	RequireExpiration bool
	Now               time.Time
}

// VerifyJWT は JWS Compact の JWT を alg の一つの方式に限って検証し、クレームを返す。
// keyFor は保護ヘッダーから検証鍵を選ぶ。署名を確かめてからクレームを判定する。
//
// exp と nbf は秒未満を切り捨てた NumericDate として、Now < exp、nbf <= Now を要求する。
// iat は判定しない。aud は文字列か文字列の配列で、Audience と一致する要素を一つ要求する。
func VerifyJWT(
	token, alg string,
	keyFor func(header map[string]any) (crypto.PublicKey, error),
	expect JWTExpectation,
) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: malformed")
	}
	header, err := decodeJWTObject(parts[0])
	if err != nil {
		return nil, fmt.Errorf("jwt: header: %w", err)
	}
	claims, err := decodeJWTObject(parts[1])
	if err != nil {
		return nil, fmt.Errorf("jwt: claims: %w", err)
	}
	if got, _ := header["alg"].(string); got != alg {
		return nil, fmt.Errorf("jwt: unexpected alg %q", header["alg"])
	}
	key, err := keyFor(header)
	if err != nil {
		return nil, fmt.Errorf("jwt: key: %w", err)
	}
	if err := verifyJWTSignature(parts, alg, key); err != nil {
		return nil, fmt.Errorf("jwt: signature: %w", err)
	}
	if err := validateJWTClaims(claims, expect); err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}
	return claims, nil
}

func decodeJWTObject(segment string) (map[string]any, error) {
	b, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("not a JSON object")
	}
	return out, nil
}

func validateJWTClaims(claims map[string]any, expect JWTExpectation) error {
	exp, hasExp, err := numericDateClaim(claims, "exp")
	if err != nil {
		return err
	}
	if hasExp && !expect.Now.Before(exp) {
		return errors.New("token is expired")
	}
	if !hasExp && expect.RequireExpiration {
		return errors.New("exp claim is required")
	}
	nbf, hasNbf, err := numericDateClaim(claims, "nbf")
	if err != nil {
		return err
	}
	if hasNbf && expect.Now.Before(nbf) {
		return errors.New("token is not valid yet")
	}
	if expect.Audience != "" {
		audience, err := audienceClaim(claims)
		if err != nil {
			return err
		}
		if !slices.Contains(audience, expect.Audience) {
			return errors.New("audience mismatch")
		}
	}
	if expect.Issuer != "" {
		if issuer, _ := claims["iss"].(string); issuer != expect.Issuer {
			return errors.New("issuer mismatch")
		}
	}
	return nil
}

// numericDateClaim は RFC 7519 の NumericDate を秒へ切り捨てて読む。数値でなければ拒否する。
func numericDateClaim(claims map[string]any, name string) (time.Time, bool, error) {
	value, present := claims[name]
	if !present {
		return time.Time{}, false, nil
	}
	seconds, ok := value.(float64)
	if !ok {
		return time.Time{}, false, fmt.Errorf("%s claim is not a number", name)
	}
	return time.Unix(int64(math.Floor(seconds)), 0), true, nil
}

func audienceClaim(claims map[string]any) ([]string, error) {
	switch value := claims["aud"].(type) {
	case string:
		return []string{value}, nil
	case []any:
		out := make([]string, 0, len(value))
		for _, member := range value {
			s, ok := member.(string)
			if !ok {
				return nil, errors.New("aud claim has a non-string member")
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, nil
}
