// Package crypto: JWT 署名・検証 (PS256)。
// TokenIssuer + TokenIntrospector の両ポートを 1 つの型で実装する。
package tokens_jose

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

const (
	accessTokenTTLSeconds = 600
	idTokenTTLSeconds     = 3600
)

type JWTSigner struct {
	Issuer   string
	KeyStore signingports.KeyStore
}

func NewJWTSigner(issuer string, ks signingports.KeyStore) *JWTSigner {
	return &JWTSigner{Issuer: issuer, KeyStore: ks}
}

func (s *JWTSigner) AccessTokenTTLSeconds() int { return accessTokenTTLSeconds }
func (s *JWTSigner) IDTokenTTLSeconds() int     { return idTokenTTLSeconds }

func (s *JWTSigner) SignAccessToken(ctx context.Context, in oauthports.AccessTokenInput) (string, string, error) {
	key, err := s.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return "", "", err
	}
	jti, err := randomBase64URL(16)
	if err != nil {
		return "", "", err
	}
	now := nowUnix()
	issuer := tenancy.Issuer(ctx, s.Issuer)
	// aud は AllAccessTokensCarryAudience 不変条件により常に 1 個以上。
	// Audiences が指定されていればそれを使い (RFC 8707 / RFC 8693)、なければ
	// 従来どおり client_id を単一 audience とする。
	var aud any = in.Client.ClientID
	switch {
	case len(in.Audiences) == 1:
		aud = in.Audiences[0]
	case len(in.Audiences) > 1:
		aud = in.Audiences
	}
	claims := map[string]any{
		"iss":       issuer,
		"sub":       in.Sub,
		"aud":       aud,
		"client_id": in.Client.ClientID,
		"scope":     strings.Join(in.Scopes, " "),
		"jti":       jti,
		"iat":       now,
		"exp":       now + accessTokenTTLSeconds,
		"auth_time": in.AuthTime,
	}
	if in.Act != nil {
		claims["act"] = in.Act
	}
	// RFC 9396 — 構造化詳細を claim としてトークンに束縛する (RS の検証点, ADR-050)。
	if len(in.AuthorizationDetails) > 0 {
		claims["authorization_details"] = in.AuthorizationDetails
	}
	if in.SenderConstraint != nil {
		cnf := map[string]string{}
		switch in.SenderConstraint.Type {
		case spec.SenderConstraintDPoP:
			cnf["jkt"] = in.SenderConstraint.JKT
		case spec.SenderConstraintMTLS:
			cnf["x5t#S256"] = in.SenderConstraint.X5TS256
		}
		claims["cnf"] = cnf
	}
	if len(in.AMR) > 0 {
		claims["amr"] = in.AMR
	}
	if in.ACR != "" {
		claims["acr"] = in.ACR
	}
	// ADR-048: client_credentials トークンが Agent に束縛されているとき、principal を
	// 非人間 identity として識別できるよう agent_id / principal_type を付与する。
	if in.AgentID != "" {
		claims["agent_id"] = in.AgentID
		claims["principal_type"] = "agent"
	}
	tok, err := signPS256(key, map[string]string{"typ": "at+jwt"}, claims)
	if err != nil {
		return "", "", err
	}
	return tok, jti, nil
}

func (s *JWTSigner) SignIDToken(ctx context.Context, in oauthports.IDTokenInput) (string, error) {
	key, err := s.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return "", err
	}
	now := nowUnix()
	issuer := tenancy.Issuer(ctx, s.Issuer)
	claims := map[string]any{
		"iss":       issuer,
		"sub":       in.User.ID,
		"aud":       in.Client.ClientID,
		"iat":       now,
		"exp":       now + idTokenTTLSeconds,
		"auth_time": in.AuthTime,
	}
	if in.Nonce != nil && *in.Nonce != "" {
		claims["nonce"] = *in.Nonce
	}
	if in.AtHashFor != "" {
		claims["at_hash"] = atHash(in.AtHashFor)
	}
	if len(in.AMR) > 0 {
		claims["amr"] = in.AMR
	}
	if in.ACR != "" {
		claims["acr"] = in.ACR
	}
	if in.Sid != "" {
		claims["sid"] = in.Sid
	}
	if containsString(in.Scopes, "profile") {
		if in.User.Name != nil {
			claims["name"] = *in.User.Name
		}
		claims["preferred_username"] = in.User.PreferredUsername
	}
	if containsString(in.Scopes, "email") && in.User.Email != nil {
		claims["email"] = *in.User.Email
		claims["email_verified"] = in.User.EmailVerified
	}
	if in.ResolveAttributeDefs != nil {
		defs, err := in.ResolveAttributeDefs(ctx, in.User.TenantID)
		if err != nil {
			return "", err
		}
		// 標準 claim とキーが衝突した場合は標準 claim を優先する。
		for key, value := range userdomain.ClaimsForScopes(*in.User, defs, in.Scopes) {
			if _, exists := claims[key]; !exists {
				claims[key] = value
			}
		}
	}
	return signPS256(key, nil, claims)
}

// VerifyIDTokenHint は /end_session の id_token_hint を検証する (OIDC RP-Initiated
// Logout 1.0, ADR-127)。署名 (登録済み鍵) と iss は fail-closed で検証するが、exp は
// 検証しない — ログアウト時点で ID Token が期限切れになっているのが通常の RP 実装
// であるため (ADR-127 決定4)。aud/sub/sid のクライアント一致判定は呼び出し側 (usecase)
// の責務とする。
func (s *JWTSigner) VerifyIDTokenHint(ctx context.Context, token string) (*oauthports.IDTokenHintClaims, error) {
	keys, err := s.KeyStore.GetAllKeys(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := verifyPS256AnyKey(token, keys)
	if err != nil {
		return nil, err
	}
	if iss, _ := payload["iss"].(string); iss != tenancy.Issuer(ctx, s.Issuer) {
		return nil, errors.New("id_token_hint: issuer mismatch")
	}
	claims := &oauthports.IDTokenHintClaims{}
	if v, ok := payload["sub"].(string); ok {
		claims.Subject = v
	}
	if v, ok := payload["sid"].(string); ok {
		claims.Sid = v
	}
	switch aud := payload["aud"].(type) {
	case string:
		claims.Audience = aud
	case []any:
		if len(aud) > 0 {
			if v, ok := aud[0].(string); ok {
				claims.Audience = v
			}
		}
	}
	return claims, nil
}

// IntrospectAccessToken は JWT を全鍵で検証する。
func (s *JWTSigner) IntrospectAccessToken(ctx context.Context, token string) (*oauthports.IntrospectionResult, error) {
	keys, err := s.KeyStore.GetAllKeys(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := verifyPS256AnyKey(token, keys)
	if err != nil {
		// RFC 7662 §2.2: invalid/expired/unparsable token → active:false で 200 OK。
		// 検証エラーは leak しない（呼び出し側 RS のクライアントに署名失敗を知らせない）。
		return &oauthports.IntrospectionResult{Active: false}, nil //nolint:nilerr // intentional per RFC 7662
	}
	if iss, _ := payload["iss"].(string); iss != tenancy.Issuer(ctx, s.Issuer) {
		return &oauthports.IntrospectionResult{Active: false}, nil
	}
	if expF, _ := payload["exp"].(float64); int64(expF) < nowUnix() {
		return &oauthports.IntrospectionResult{Active: false}, nil
	}
	res := &oauthports.IntrospectionResult{Active: true, TokenType: "access_token"}
	if v, ok := payload["jti"].(string); ok {
		res.JTI = v
	}
	if v, ok := payload["client_id"].(string); ok {
		res.ClientID = v
	}
	if v, ok := payload["sub"].(string); ok {
		res.Sub = v
	}
	if v, ok := payload["scope"].(string); ok {
		res.Scope = v
	}
	if v, ok := payload["exp"].(float64); ok {
		res.Exp = int64(v)
	}
	if v, ok := payload["iat"].(float64); ok {
		res.Iat = int64(v)
	}
	if cnf, ok := payload["cnf"].(map[string]any); ok {
		sc := &domain.SenderConstraint{}
		if jkt, ok := cnf["jkt"].(string); ok {
			sc.Type = spec.SenderConstraintDPoP
			sc.JKT = jkt
		} else if x5t, ok := cnf["x5t#S256"].(string); ok {
			sc.Type = spec.SenderConstraintMTLS
			sc.X5TS256 = x5t
		}
		if sc.Type != "" {
			res.SenderConstraint = sc
		}
	}
	res.Aud = normalizeAudience(payload["aud"])
	if act, ok := payload["act"].(map[string]any); ok {
		res.Act = act
	}
	if mayAct, ok := payload["may_act"].(map[string]any); ok {
		res.MayAct = mayAct
	}
	if raw, ok := payload["authorization_details"]; ok {
		if encoded, err := json.Marshal(raw); err == nil {
			var details []spec.AuthorizationDetail
			if json.Unmarshal(encoded, &details) == nil {
				res.AuthorizationDetails = details
			}
		}
	}
	return res, nil
}

// normalizeAudience は JWT の aud claim (単一文字列 / 文字列配列) を []string に正規化する。
func normalizeAudience(v any) []string {
	switch typed := v.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return typed
	}
	return nil
}

// =====================================================================
// PS256 署名・検証ヘルパ (jose ライブラリ非依存)
// =====================================================================

func signPS256(key *signingdomain.SigningKey, extraHeader map[string]string, claims map[string]any) (string, error) {
	// crypto.Signer 経由で署名する。Local / Postgres provider の *rsa.PrivateKey は
	// crypto.Signer を満たし、VaultTransit provider は署名を Vault へ委譲する Signer を渡す。
	signer, ok := key.PrivateKey.(crypto.Signer)
	if !ok {
		return "", errors.New("active key has no signer")
	}
	header := map[string]any{"alg": "PS256", "kid": key.Kid}
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

func verifyPS256AnyKey(token string, keys []*signingdomain.SigningKey) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed JWT")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header map[string]any
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, err
	}
	alg, _ := header["alg"].(string)
	if alg != "PS256" {
		return nil, fmt.Errorf("alg %q not allowed", alg)
	}
	kid, _ := header["kid"].(string)
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	for _, k := range keys {
		if kid != "" && k.Kid != kid {
			continue
		}
		pub, ok := k.PublicKey.(*rsa.PublicKey)
		if !ok {
			continue
		}
		if err := rsa.VerifyPSS(pub, crypto.SHA256, digest[:], sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}); err == nil {
			pb, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				return nil, err
			}
			var payload map[string]any
			if err := json.Unmarshal(pb, &payload); err != nil {
				return nil, err
			}
			return payload, nil
		}
	}
	return nil, errors.New("signature verification failed")
}

// =====================================================================
// 補助関数
// =====================================================================

func atHash(accessToken string) string {
	digest := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(digest[:len(digest)/2])
}

func nowUnix() int64 { return time.Now().Unix() }

func randomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func containsString(ss []string, s string) bool {
	return slices.Contains(ss, s)
}
