package usecases

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// Enroll は新しい P-256 の鍵対を作り、公開鍵を登録トークンで登録し、受理されたら秘密鍵と登録の応答を
// 保存する (REQ-DIRECTORYCONNECTOR-001)。拒否されたら何も保存しない。
func Enroll(ctx context.Context, api ports.EnrollmentAPI, store ports.CredentialStore, token string) (ports.Enrollment, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return ports.Enrollment{}, fmt.Errorf("generate key: %w", err)
	}
	point, err := key.PublicKey.Bytes()
	if err != nil {
		return ports.Enrollment{}, fmt.Errorf("encode public key: %w", err)
	}
	// 非圧縮の点は 0x04 の後に 32 バイトずつの x と y が続く。
	jwk := map[string]any{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(point[1:33]),
		"y": base64.RawURLEncoding.EncodeToString(point[33:65]),
	}
	enrollment, err := api.Enroll(ctx, token, jwk)
	if err != nil {
		return ports.Enrollment{}, fmt.Errorf("enroll: %w", err)
	}
	if err := store.SaveCredentials(ctx, key, enrollment); err != nil {
		return ports.Enrollment{}, fmt.Errorf("save credentials: %w", err)
	}
	return enrollment, nil
}
