package usecases

// backup recovery code の生成・消費・失効 (wi-26 / ADR-087)。平文は生成時に一度だけ返し、
// DB には SHA-256 hex のみ保存する。1 コードは single-use、再生成は既存 set を全置換する。
// recovery code は TOTP / WebAuthn 喪失時の backup で、単独では第二要素にしない。

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	// RecoveryCodePolicy (SCL) に対応: 10 桁 x 10 コード、曖昧文字を除いた 32 文字集合。
	recoveryCodeCount    = 10
	recoveryCodeLength   = 10
	recoveryCodeAlphabet = "23456789abcdefghijkmnpqrstuvwxyz" // len=32 (256%32==0 でモジュロバイアス無し)
)

// ErrRecoveryCodeInvalid は未知 / 使用済み / 形式不正な recovery code を消費しようとした場合。
var ErrRecoveryCodeInvalid = errors.New("recovery code is invalid")

// RecoveryCodesDeps は recovery code use case の依存。
type RecoveryCodesDeps struct {
	UserRepo         userports.UserRepository
	RecoveryCodeRepo authnports.RecoveryCodeRepository
	Emit             func(spec.DomainEvent)
}

// RecoveryCodesResult は GenerateRecoveryCodes の戻り値。Codes は平文で、呼び出し側は一度だけ
// ユーザーに提示し保存しない。
type RecoveryCodesResult struct {
	Codes       []string
	GeneratedAt time.Time
}

// RecoveryCodeStatusResult は残数表示用の状態。
type RecoveryCodeStatusResult struct {
	GeneratedAt *time.Time
	Total       int
	Remaining   int
}

// GenerateRecoveryCodes は新しい recovery code set を生成し、既存 set を全置換する。
func GenerateRecoveryCodes(
	ctx context.Context,
	deps RecoveryCodesDeps,
	sub string,
	now time.Time,
) (*RecoveryCodesResult, error) {
	now = normalizedNow(now)
	user, err := loadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return nil, err
	}
	plain := make([]string, 0, recoveryCodeCount)
	stored := make([]*domain.RecoveryCode, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		plain = append(plain, code)
		stored = append(stored, &domain.RecoveryCode{
			UserID: sub, CodeHash: hashRecoveryCode(code), GeneratedAt: now,
		})
	}
	if err := deps.RecoveryCodeRepo.ReplaceAll(ctx, sub, stored); err != nil {
		return nil, err
	}
	if deps.Emit != nil {
		deps.Emit(&domain.RecoveryCodesGenerated{
			At: now, TenantID: user.TenantID, UserID: user.ID, Count: len(plain),
		})
	}
	return &RecoveryCodesResult{Codes: plain, GeneratedAt: now}, nil
}

// RevokeRecoveryCodes は有効な recovery code をすべて失効する。
func RevokeRecoveryCodes(
	ctx context.Context,
	deps RecoveryCodesDeps,
	sub string,
	now time.Time,
) error {
	now = normalizedNow(now)
	user, err := loadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return err
	}
	if err := deps.RecoveryCodeRepo.DeleteAllForSub(ctx, sub); err != nil {
		return err
	}
	if deps.Emit != nil {
		deps.Emit(&domain.RecoveryCodesRevoked{At: now, TenantID: user.TenantID, UserID: user.ID})
	}
	return nil
}

// ConsumeRecoveryCode は第二要素として recovery code を 1 つ消費する。単一の MarkConsumed で
// single-use を保証し、消費後の残数を返す。未知 / 使用済みは ErrRecoveryCodeInvalid。
func ConsumeRecoveryCode(
	ctx context.Context,
	deps RecoveryCodesDeps,
	sub string,
	code string,
	now time.Time,
) (int, error) {
	now = normalizedNow(now)
	user, err := loadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return 0, err
	}
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return 0, ErrRecoveryCodeInvalid
	}
	ok, err := deps.RecoveryCodeRepo.MarkConsumed(ctx, sub, hashRecoveryCode(normalized), now)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrRecoveryCodeInvalid
	}
	remaining, err := remainingRecoveryCodes(ctx, deps.RecoveryCodeRepo, sub)
	if err != nil {
		return 0, err
	}
	if deps.Emit != nil {
		count := remaining
		deps.Emit(&domain.BackupCodeConsumed{
			At: now, TenantID: user.TenantID, UserID: user.ID, RemainingCount: &count,
		})
	}
	return remaining, nil
}

// RecoveryCodeStatusFor は残数表示用の状態を返す。
func RecoveryCodeStatusFor(
	ctx context.Context,
	repo authnports.RecoveryCodeRepository,
	sub string,
) (*RecoveryCodeStatusResult, error) {
	codes, err := repo.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	result := &RecoveryCodeStatusResult{Total: len(codes)}
	var generatedAt *time.Time
	for _, c := range codes {
		if c.ConsumedAt == nil {
			result.Remaining++
		}
		if generatedAt == nil || c.GeneratedAt.Before(*generatedAt) {
			at := c.GeneratedAt
			generatedAt = &at
		}
	}
	result.GeneratedAt = generatedAt
	return result, nil
}

func remainingRecoveryCodes(
	ctx context.Context,
	repo authnports.RecoveryCodeRepository,
	sub string,
) (int, error) {
	codes, err := repo.ListBySub(ctx, sub)
	if err != nil {
		return 0, err
	}
	remaining := 0
	for _, c := range codes {
		if c.ConsumedAt == nil {
			remaining++
		}
	}
	return remaining, nil
}

// generateRecoveryCode は曖昧文字を除いた 32 文字集合から crypto/rand で 1 コード生成する。
func generateRecoveryCode() (string, error) {
	buf := make([]byte, recoveryCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, recoveryCodeLength)
	for i, b := range buf {
		out[i] = recoveryCodeAlphabet[int(b)%len(recoveryCodeAlphabet)]
	}
	return string(out), nil
}

// normalizeRecoveryCode は表示上の空白 / ハイフンを除き小文字化する。
func normalizeRecoveryCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(code)) {
		if r == ' ' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hashRecoveryCode(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
