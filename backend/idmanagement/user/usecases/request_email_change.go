package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

const EmailChangeTokenTTLSeconds = 1800

var (
	ErrInvalidEmail   = errors.New("email is not a valid address")
	ErrEmailUnchanged = errors.New("email is unchanged")
	ErrEmailTaken     = errors.New("email is already in use")
)

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// RequestEmailChangeDeps / Input は primary email 変更の起票 (self-service, wi-21)。
// 新アドレスへワンタイムリンクを送り、確定は ConfirmEmailChange で行う。実際の
// User.Email 更新は確定時まで起きない (新アドレスの所有確認を経るまで反映しない)。
type RequestEmailChangeDeps struct {
	UserRepo   userports.UserRepository
	TokenStore userports.EmailChangeTokenStore
	// Notifier は文面 (件名 / テキスト / HTML) と locale を通知テンプレートカタログから
	// 解決する。use case は文面を組み立てない (ADR-142)。
	Notifier sharednotification.Notifier
	Emit     func(spec.DomainEvent)
	Issuer   string
	TokenTTL time.Duration
}

type RequestEmailChangeInput struct {
	Sub      string
	NewEmail string
	Now      time.Time
}

func RequestEmailChange(ctx context.Context, deps RequestEmailChangeDeps, in RequestEmailChangeInput) error {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(in.NewEmail))
	if err != nil {
		return ErrInvalidEmail
	}
	newEmail := strings.ToLower(addr.Address)

	user, err := deps.UserRepo.FindBySub(ctx, in.Sub)
	if err != nil {
		return err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) {
		return idmusecases.ErrUserNotFound
	}
	if user.Email != nil && user.EmailVerified && strings.EqualFold(*user.Email, newEmail) {
		return ErrEmailUnchanged
	}
	// tenant 内で他ユーザが使っているアドレスは拒否する。
	existing, err := deps.UserRepo.FindByEmail(ctx, user.TenantID, newEmail)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != user.ID {
		return ErrEmailTaken
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(raw)
	ttl := deps.TokenTTL
	if ttl == 0 {
		ttl = EmailChangeTokenTTLSeconds * time.Second
	}
	if err := deps.TokenStore.Save(ctx, userports.EmailChangeTokenRecord{
		Sub: user.ID, TokenHash: sha256Hex(rawToken), NewEmail: newEmail,
		CreatedAt: now, ExpiresAt: now.Add(ttl),
	}); err != nil {
		return err
	}

	// リンクはここで組み立てる。テナントがテンプレートを編集しても URL の構築を
	// 奪えないようにする (ADR-142 決定 5)。
	verifyURL := strings.TrimRight(deps.Issuer, "/") + "/account/email/verify?token=" + url.QueryEscape(rawToken)
	minutes := int(ttl.Round(time.Minute) / time.Minute)
	delivered := deps.Notifier.Notify(ctx, sharednotification.Notification{
		TenantID:        user.TenantID,
		To:              newEmail,
		Key:             sharednotification.TemplateKeyEmailChangeConfirmation,
		RecipientLocale: user.LocaleAttribute(),
		Vars: map[string]string{
			"user_display_name":  user.DisplayName(),
			"confirmation_url":   verifyURL,
			"expires_in_minutes": strconv.Itoa(minutes),
			"new_email":          newEmail,
		},
	})
	if deps.Emit != nil {
		deps.Emit(&idmdomain.EmailChangeRequested{
			At: now, TenantID: user.TenantID, UserID: user.ID, NewEmailHash: sha256Hex(newEmail),
		})
		deps.Emit(&spec.EmailSent{
			At: now, ToHash: sha256Hex(newEmail), Purpose: "email_change", Delivered: delivered,
		})
	}
	return nil
}
