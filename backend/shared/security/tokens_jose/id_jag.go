package tokens_jose

// ID-JAG (draft-ietf-oauth-identity-assertion-authz-grant-05) の署名と検証。検証は純関数で、
// 発行者の信頼の解決と jti の記憶は呼び出し側が行う。

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// IDJAGType は ID-JAG の typ ヘッダーの値 (§3.1)。
	IDJAGType = "oauth-id-jag+jwt"
	// IDJAGClockSkew は iat と exp の比較で許す時計のずれ。
	IDJAGClockSkew = 60 * time.Second
	// IDJAGMaxLifetime は受け入れる exp - iat の上限。
	IDJAGMaxLifetime = 600 * time.Second
)

// ErrIDJAGInvalid は ID-JAG が検証を通らなかったことを表す。理由は応答へ出さない。
var ErrIDJAGInvalid = errors.New("id-jag: invalid")

// IDJAGConfirmation は ID-JAG の cnf。Other は対応しない確認方式があったことを表す。
type IDJAGConfirmation struct {
	JKT     string
	X5TS256 string
	Other   bool
}

// IDJAGClaims は検証を通った ID-JAG のクレーム。
type IDJAGClaims struct {
	Issuer               string
	Subject              string
	ClientID             string
	JTI                  string
	IssuedAt             time.Time
	ExpiresAt            time.Time
	Resources            []string
	Scope                string
	AuthorizationDetails json.RawMessage
	HasActor             bool
	AuthTime             int64
	AMR                  []string
	ACR                  string
	Confirmation         *IDJAGConfirmation
}

// SignRS256 は RSASSA-PKCS1-v1_5 で JWT に署名する。ID-JAG の専用鍵だけが使う。
func SignRS256(privateKey any, kid string, extraHeader map[string]string, claims map[string]any) (string, error) {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return "", errors.New("active key has no signer")
	}
	header := map[string]any{"alg": "RS256", "kid": kid}
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
	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// UnverifiedIDJAGIssuer は信頼する発行者を探すために iss だけを読む。戻り値を信頼の根拠にしない。
func UnverifiedIDJAGIssuer(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", ErrIDJAGInvalid
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrIDJAGInvalid
	}
	var payload struct {
		Issuer any `json:"iss"`
	}
	if err := json.Unmarshal(pb, &payload); err != nil {
		return "", ErrIDJAGInvalid
	}
	iss, ok := payload.Issuer.(string)
	if !ok || iss == "" {
		return "", ErrIDJAGInvalid
	}
	return iss, nil
}

// VerifyIDJAG は §4.4.1 のうち、トークン自身で判定できる規則を検証する。
func VerifyIDJAG(token string, jwks []map[string]any, issuer, audience string, now time.Time) (*IDJAGClaims, error) {
	fail := func(reason string) (*IDJAGClaims, error) {
		return nil, fmt.Errorf("%w: %s", ErrIDJAGInvalid, reason)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fail("malformed")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fail("header")
	}
	var header map[string]any
	if err := json.Unmarshal(hb, &header); err != nil {
		return fail("header")
	}
	if typ, _ := header["typ"].(string); typ != IDJAGType {
		return fail("typ")
	}
	alg, _ := header["alg"].(string)
	if alg != "RS256" && alg != "ES256" && alg != "PS256" {
		return fail("alg")
	}
	kid, _ := header["kid"].(string)
	if !jwkAllowsAlg(jwks, kid, alg) {
		return fail("alg of the key")
	}
	pub, err := pickJWK(jwks, kid)
	if err != nil {
		return fail("key")
	}
	if err := verifyJWTSignature(parts, alg, pub); err != nil {
		return fail("signature")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fail("payload")
	}
	var payload map[string]any
	if err := json.Unmarshal(pb, &payload); err != nil {
		return fail("payload")
	}
	iss, _ := payload["iss"].(string)
	if iss == "" || iss != issuer {
		return fail("iss")
	}
	auds := audienceStrings(payload["aud"])
	if len(auds) != 1 || auds[0] != audience {
		return fail("aud")
	}
	sub, _ := payload["sub"].(string)
	clientID, _ := payload["client_id"].(string)
	jti, _ := payload["jti"].(string)
	if sub == "" || clientID == "" || jti == "" {
		return fail("required claim")
	}
	iatF, iatOK := payload["iat"].(float64)
	expF, expOK := payload["exp"].(float64)
	if !iatOK || !expOK {
		return fail("iat or exp")
	}
	iat := time.Unix(int64(iatF), 0).UTC()
	exp := time.Unix(int64(expF), 0).UTC()
	if iat.After(now.Add(IDJAGClockSkew)) {
		return fail("iat in the future")
	}
	if !exp.Add(IDJAGClockSkew).After(now) {
		return fail("expired")
	}
	if exp.Sub(iat) > IDJAGMaxLifetime || !exp.After(iat) {
		return fail("lifetime")
	}
	resources, ok := idJagResources(payload["resource"])
	if !ok {
		return fail("resource")
	}
	claims := &IDJAGClaims{
		Issuer: iss, Subject: sub, ClientID: clientID, JTI: jti, IssuedAt: iat, ExpiresAt: exp,
		Resources: resources,
	}
	claims.Scope, _ = payload["scope"].(string)
	_, claims.HasActor = payload["act"]
	if authTime, ok := payload["auth_time"].(float64); ok {
		claims.AuthTime = int64(authTime)
	}
	claims.ACR, _ = payload["acr"].(string)
	if amr, ok := payload["amr"].([]any); ok {
		for _, item := range amr {
			if value, ok := item.(string); ok {
				claims.AMR = append(claims.AMR, value)
			}
		}
	}
	if raw, ok := payload["authorization_details"]; ok {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return fail("authorization_details")
		}
		claims.AuthorizationDetails = encoded
	}
	if raw, ok := payload["cnf"]; ok {
		cnf, ok := raw.(map[string]any)
		if !ok {
			return fail("cnf")
		}
		claims.Confirmation = idJagConfirmation(cnf)
	}
	return claims, nil
}

// jwkAllowsAlg は、JWK が alg を宣言しているとき、ヘッダーの alg と一致するかを確かめる。
func jwkAllowsAlg(jwks []map[string]any, kid, alg string) bool {
	for _, jwk := range jwks {
		if id, _ := jwk["kid"].(string); kid != "" && id != kid {
			continue
		}
		if declared, ok := jwk["alg"].(string); ok && declared != "" && declared != alg {
			return false
		}
	}
	return true
}

// idJagResources は resource クレームを読む。文字列か、重複のない空でない文字列の配列を受ける。
// クレームがなければ空を返す。
func idJagResources(raw any) ([]string, bool) {
	switch value := raw.(type) {
	case nil:
		return nil, true
	case string:
		return []string{value}, value != ""
	case []any:
		if len(value) == 0 {
			return nil, false
		}
		seen := map[string]bool{}
		out := make([]string, 0, len(value))
		for _, item := range value {
			resource, ok := item.(string)
			if !ok || resource == "" || seen[resource] {
				return nil, false
			}
			seen[resource] = true
			out = append(out, resource)
		}
		return out, true
	}
	return nil, false
}

func idJagConfirmation(cnf map[string]any) *IDJAGConfirmation {
	confirmation := &IDJAGConfirmation{}
	for member, value := range cnf {
		text, _ := value.(string)
		switch {
		case member == "jkt" && text != "":
			confirmation.JKT = text
		case member == "x5t#S256" && text != "":
			confirmation.X5TS256 = text
		default:
			confirmation.Other = true
		}
	}
	return confirmation
}
