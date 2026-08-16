// Package crypto: DPoP proof JWT 検証 (RFC 9449)。
package tokens_jose

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	dpopClockSkewPastSeconds   = 60
	dpopClockSkewFutureSeconds = 5
	dpopJTIReplayWindowSeconds = 600
)

type DPoPResult struct {
	JKT string
}

// AccessTokenHash returns the ath claim value base64url(SHA-256(access token)) of a
// DPoP proof (RFC 9449 §4.3). The hashed input is the access token string the client
// presented, never a post-introspection internal representation of it.
func AccessTokenHash(accessToken string) string {
	digest := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// VerifyDPoPForToken verifies a proof presented to a token endpoint (/token, /par).
// No ath is required there: the access token it would bind to does not exist yet
// (RFC 9449 §4.3). A missing proof means "DPoP not used", so it returns nil without error.
func VerifyDPoPForToken(
	ctx context.Context,
	dpopHeader, expectedHTM, expectedHTU string,
	replay ports.DpopReplayStore,
	now time.Time,
) (*DPoPResult, error) {
	if dpopHeader == "" {
		return nil, nil
	}
	return verifyDPoP(ctx, dpopHeader, expectedHTM, expectedHTU, "", replay, now)
}

// VerifyDPoPForResource verifies a proof presented to a protected resource
// (REQ-OAUTH2-045). Unlike the token endpoint path, the proof must be bound to
// accessToken through ath, and both a missing proof and a missing access token are
// errors. The two paths are separate functions so that a caller who forgets to pass
// the access token cannot silently skip the ath check.
func VerifyDPoPForResource(
	ctx context.Context,
	dpopHeader, expectedHTM, expectedHTU, accessToken string,
	replay ports.DpopReplayStore,
	now time.Time,
) (*DPoPResult, error) {
	if dpopHeader == "" {
		return nil, errors.New("dpop: proof required")
	}
	if accessToken == "" {
		return nil, errors.New("dpop: access token required")
	}
	return verifyDPoP(ctx, dpopHeader, expectedHTM, expectedHTU, accessToken, replay, now)
}

// verifyDPoP verifies a DPoP header JWT and returns its JWK thumbprint. It fails on an
// htm/htu mismatch, a bad signature, an iat outside the clock skew, or a replayed jti.
// A non-empty accessToken makes ath mandatory and checks it against that token.
func verifyDPoP(
	ctx context.Context,
	dpopHeader, expectedHTM, expectedHTU, accessToken string,
	replay ports.DpopReplayStore,
	now time.Time,
) (*DPoPResult, error) {
	parts := strings.Split(dpopHeader, ".")
	if len(parts) != 3 {
		return nil, errors.New("dpop: malformed proof")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("dpop: decode header: %w", err)
	}
	var header map[string]any
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, fmt.Errorf("dpop: parse header: %w", err)
	}
	if typ, _ := header["typ"].(string); typ != "dpop+jwt" {
		return nil, errors.New("dpop: typ must be dpop+jwt")
	}
	alg, _ := header["alg"].(string)
	if alg != "PS256" && alg != "ES256" {
		return nil, errors.New("dpop: alg must be PS256 or ES256")
	}
	jwk, _ := header["jwk"].(map[string]any)
	if jwk == nil {
		return nil, errors.New("dpop: jwk header required")
	}

	pub, err := publicKeyFromJWK(jwk)
	if err != nil {
		return nil, fmt.Errorf("dpop: import jwk: %w", err)
	}
	if err := verifyJWTSignature(parts, alg, pub); err != nil {
		return nil, fmt.Errorf("dpop: signature: %w", err)
	}

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("dpop: decode payload: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(pb, &payload); err != nil {
		return nil, fmt.Errorf("dpop: parse payload: %w", err)
	}
	if htm, _ := payload["htm"].(string); htm != expectedHTM {
		return nil, fmt.Errorf("dpop: htm mismatch (got %q, want %q)", htm, expectedHTM)
	}
	if htu, _ := payload["htu"].(string); htu != expectedHTU {
		return nil, fmt.Errorf("dpop: htu mismatch (got %q, want %q)", htu, expectedHTU)
	}
	if accessToken != "" {
		ath, _ := payload["ath"].(string)
		if ath == "" {
			return nil, errors.New("dpop: ath required")
		}
		if subtle.ConstantTimeCompare([]byte(ath), []byte(AccessTokenHash(accessToken))) != 1 {
			return nil, errors.New("dpop: ath mismatch")
		}
	}
	iat, _ := payload["iat"].(float64)
	skew := now.Unix() - int64(iat)
	if skew > dpopClockSkewPastSeconds || skew < -dpopClockSkewFutureSeconds {
		return nil, fmt.Errorf("dpop: iat outside clock skew")
	}
	jti, _ := payload["jti"].(string)
	if jti == "" {
		return nil, errors.New("dpop: jti required")
	}
	// jti は replay 表の主キーの成分になる。上限が無いと、client が決めた長さの
	// まま btree の索引行上限で落ちる。応答形は OAuth 2.0 が決めるので、型付きの
	// LengthError は返さない。
	if err := spec.CheckKeyString("jti", jti, spec.LengthProtocolMessageID, spec.BytesProtocolMessageID); err != nil {
		return nil, fmt.Errorf("dpop: %s", err.Error())
	}
	isNew, err := replay.RecordIfNew(ctx, jti, dpopJTIReplayWindowSeconds, now)
	if err != nil {
		return nil, err
	}
	if !isNew {
		return nil, errors.New("dpop: jti replay detected")
	}
	jkt, err := jwkThumbprint(jwk)
	if err != nil {
		return nil, err
	}
	return &DPoPResult{JKT: jkt}, nil
}

// =====================================================================
// JWK → 公開鍵
// =====================================================================

func publicKeyFromJWK(jwk map[string]any) (crypto.PublicKey, error) {
	kty, _ := jwk["kty"].(string)
	switch kty {
	case "RSA":
		nB64, _ := jwk["n"].(string)
		eB64, _ := jwk["e"].(string)
		nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
		if err != nil {
			return nil, err
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = (e << 8) | int(b)
		}
		return &rsa.PublicKey{N: n, E: e}, nil
	case "EC":
		crv, _ := jwk["crv"].(string)
		if crv != "P-256" {
			return nil, fmt.Errorf("unsupported EC curve %q", crv)
		}
		xB64, _ := jwk["x"].(string)
		yB64, _ := jwk["y"].(string)
		xBytes, err := base64.RawURLEncoding.DecodeString(xB64)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(yB64)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}, nil
	}
	return nil, fmt.Errorf("unsupported kty %q", kty)
}

func rsaPublicJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(pub.E)).Bytes()),
	}
}

func jwkThumbprint(jwk map[string]any) (string, error) {
	required := []string{"e", "kty", "n"}
	canonical := map[string]any{}
	for _, key := range required {
		value, ok := jwk[key]
		if !ok {
			return "", fmt.Errorf("jwk missing required member: %s", key)
		}
		canonical[key] = value
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

// verifyJWTSignature は alg ごとに署名を検証する。
func verifyJWTSignature(parts []string, alg string, pub crypto.PublicKey) error {
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	switch alg {
	case "PS256":
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return errors.New("PS256 requires RSA public key")
		}
		return rsa.VerifyPSS(rsaPub, crypto.SHA256, digest[:], sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	case "RS256":
		// 外部 workload attestation issuer (Kubernetes projected ServiceAccount token、
		// クラウド instance identity token) は既定で RS256 を使うことが多い。本アプリが
		// 自己発行する JWT は PS256/ES256 に限定するが、外部 issuer の検証はその制約を
		// 課さない ([[wi-54-workload-identity-federation-spiffe]])。
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return errors.New("RS256 requires RSA public key")
		}
		return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest[:], sig)
	case "ES256":
		ecPub, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("ES256 requires EC public key")
		}
		if len(sig) != 64 {
			return errors.New("ES256 signature must be 64 bytes")
		}
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		if !ecdsa.Verify(ecPub, digest[:], r, s) {
			return errors.New("ES256 signature invalid")
		}
		return nil
	}
	return fmt.Errorf("unsupported alg %q", alg)
}
