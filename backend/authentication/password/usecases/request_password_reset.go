package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

const PasswordResetTokenTTLSeconds = 1800

type RequestPasswordResetDeps struct {
	UserRepo   userports.UserRepository
	TokenStore passwordports.PasswordResetTokenStore
	// Notifier は文面 (件名 / テキスト / HTML) と locale を通知テンプレートカタログから
	// 解決する。use case は文面を組み立てない。
	Notifier sharednotification.Notifier
	Emit     func(spec.DomainEvent)
	Issuer   string
	TokenTTL time.Duration
}

type RequestPasswordResetInput struct {
	Email string
	Now   time.Time
}

func RequestPasswordReset(ctx context.Context, deps RequestPasswordResetDeps, in RequestPasswordResetInput) error {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if deps.Emit != nil {
		deps.Emit(&authdomain.PasswordResetRequested{At: now, TenantID: tenancy.TenantID(ctx), EmailHash: sha256Hex(email)})
	}
	if email == "" {
		return nil
	}

	user, err := deps.UserRepo.FindByEmail(ctx, tenancy.TenantID(ctx), email)
	if err != nil {
		return err
	}
	if user == nil || !user.EmailVerified || user.Email == nil {
		return nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(raw)
	ttl := deps.TokenTTL
	if ttl == 0 {
		ttl = PasswordResetTokenTTLSeconds * time.Second
	}
	if err := deps.TokenStore.Save(ctx, passwordports.PasswordResetTokenRecord{
		Sub:       user.ID,
		TokenHash: sha256Hex(rawToken),
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}); err != nil {
		return err
	}

	// The link is assembled here, not inside the template, so a tenant editing the
	// template can never take over URL construction.
	resetURL := strings.TrimRight(deps.Issuer, "/") + "/reset_password?token=" + url.QueryEscape(rawToken)
	minutes := int(ttl.Round(time.Minute) / time.Minute)
	// Send to the verified address stored on the account, not the raw request
	// input, so untrusted request data never reaches the email content (CWE-640).
	delivered := deps.Notifier.Notify(ctx, sharednotification.Notification{
		TenantID:        tenancy.TenantID(ctx),
		To:              *user.Email,
		Key:             sharednotification.TemplateKeyPasswordReset,
		RecipientLocale: user.LocaleAttribute(),
		Vars: map[string]string{
			"user_display_name":  user.DisplayName(),
			"reset_url":          resetURL,
			"expires_in_minutes": strconv.Itoa(minutes),
		},
	})
	if deps.Emit != nil {
		deps.Emit(&spec.EmailSent{
			At: now, ToHash: sha256Hex(email), Purpose: "password_reset", Delivered: delivered,
		})
	}
	return nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
