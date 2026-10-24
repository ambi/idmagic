package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type Attempt struct {
	TenantID string `json:"tenant"`
	Target   string `json:"target"`
	Expires  int64  `json:"expires"`
}

func (a Attempt) Sign(secret []byte) string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ValidAttempt(value string, secret []byte, tenantID, target string, now time.Time) bool {
	if len(secret) < 32 || len(value) > 2048 {
		return false
	}
	encoded, signature, ok := strings.Cut(value, ".")
	if !ok {
		return false
	}
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return false
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	var a Attempt
	if json.Unmarshal(data, &a) != nil {
		return false
	}
	return a.TenantID == tenantID && a.Target == target && a.Expires > now.Unix() && a.Expires <= now.Add(5*time.Minute).Unix()
}
