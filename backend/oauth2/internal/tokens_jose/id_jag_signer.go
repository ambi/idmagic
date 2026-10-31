package tokens_jose

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	jose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// IDJAGTTLSeconds は ID-JAG の有効期間を返す。
func (s *JWTSigner) IDJAGTTLSeconds() int { return idJagTTLSeconds }

// SignIDJAG は ID-JAG に専用の RS256 の鍵で署名する (REQ-SIGNINGKEYS-013)。
func (s *JWTSigner) SignIDJAG(ctx context.Context, in oauthports.IDJAGInput) (string, string, error) {
	key, err := s.KeyStore.GetActiveKey(signingports.WithKeyUsage(ctx, signingdomain.KeyUsageIdJagSigning))
	if err != nil {
		return "", "", err
	}
	jti, err := randomBase64URL(16)
	if err != nil {
		return "", "", err
	}
	now := nowUnix()
	claims := map[string]any{
		"iss":       tenantports.Issuer(ctx, s.Issuer),
		"sub":       in.Subject,
		"aud":       in.Audience,
		"client_id": in.ClientID,
		"jti":       jti,
		"iat":       now,
		"exp":       now + idJagTTLSeconds,
		"auth_time": in.AuthTime,
	}
	if len(in.Resources) == 1 {
		claims["resource"] = in.Resources[0]
	} else if len(in.Resources) > 1 {
		claims["resource"] = in.Resources
	}
	if len(in.Scopes) > 0 {
		claims["scope"] = strings.Join(in.Scopes, " ")
	}
	if len(in.AuthorizationDetails) > 0 {
		claims["authorization_details"] = in.AuthorizationDetails
	}
	if len(in.AMR) > 0 {
		claims["amr"] = in.AMR
	}
	if in.ACR != "" {
		claims["acr"] = in.ACR
	}
	if in.Email != "" {
		claims["email"] = in.Email
	}
	if sc := in.SenderConstraint; sc != nil {
		switch sc.Type {
		case spec.SenderConstraintDPoP:
			claims["cnf"] = map[string]string{"jkt": sc.JKT}
		case spec.SenderConstraintMTLS:
			claims["cnf"] = map[string]string{"x5t#S256": sc.X5TS256}
		}
	}
	token, err := jose.SignRS256(key.PrivateKey, key.Kid, map[string]string{"typ": jose.IDJAGType}, claims)
	if err != nil {
		return "", "", err
	}
	return token, jti, nil
}

// VerifyIDTokenAssertion は subject_token が自身の発行した有効な ID Token かを確かめる。
// ID Token は typ を持たないので、typ がなく token_use が id_token のものだけを受ける。
func (s *JWTSigner) VerifyIDTokenAssertion(ctx context.Context, token string) (*oauthports.IDTokenAssertion, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("id token: malformed")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}
	if _, typed := header["typ"]; typed {
		return nil, errors.New("id token: the typ identifies another kind of token")
	}
	keys, err := s.KeyStore.GetAllKeys(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := verifyPS256AnyKey(token, keys)
	if err != nil {
		return nil, err
	}
	if use, _ := payload["token_use"].(string); use != tokenUseIDToken {
		return nil, errors.New("id token: token_use mismatch")
	}
	if iss, _ := payload["iss"].(string); iss != tenantports.Issuer(ctx, s.Issuer) {
		return nil, errors.New("id token: issuer mismatch")
	}
	exp, ok := payload["exp"].(float64)
	if !ok || int64(exp) <= nowUnix() {
		return nil, errors.New("id token: expired")
	}
	sub, _ := payload["sub"].(string)
	if sub == "" {
		return nil, errors.New("id token: sub is required")
	}
	assertion := &oauthports.IDTokenAssertion{Subject: sub, Audiences: normalizeAudience(payload["aud"])}
	if iat, ok := payload["iat"].(float64); ok {
		assertion.IssuedAt = time.Unix(int64(iat), 0).UTC()
	}
	if authTime, ok := payload["auth_time"].(float64); ok {
		assertion.AuthTime = int64(authTime)
	}
	assertion.ACR, _ = payload["acr"].(string)
	if amr, ok := payload["amr"].([]any); ok {
		for _, item := range amr {
			if value, ok := item.(string); ok {
				assertion.AMR = append(assertion.AMR, value)
			}
		}
	}
	return assertion, nil
}
