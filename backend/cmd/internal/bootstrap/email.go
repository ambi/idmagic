package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
	emailConsole "github.com/ambi/idmagic/backend/shared/notification/email_console"
	emailSMTP "github.com/ambi/idmagic/backend/shared/notification/email_smtp"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// ResolveEmailSender builds the EmailSender adapter selected by
// cfg.EmailSender. cfg is assumed already validated by LoadSharedConfig
// (EMAIL_SENDER is a closed enum; SMTP_HOST/SMTP_FROM are required when
// EMAIL_SENDER=smtp), so this function only constructs the adapter.
func ResolveEmailSender(cfg SharedConfig) sharednotification.EmailSender {
	switch cfg.EmailSender {
	case "smtp":
		smtpCfg := emailSMTP.SMTPEmailSenderConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUser,
			Password: cfg.SMTPPass.Value(),
			From:     cfg.SMTPFrom,
			Hello:    cfg.SMTPHelo,
			TLSMode:  cfg.SMTPTLSMode,
			Timeout:  cfg.SMTPTimeout,
		}
		// from はサービス自身の送信元 (運用設定) であり data subject の PII ではない。
		// Username/Password は secret のためログに出さない。
		logging.Info(context.Background(), "email sender configured",
			"kind", "smtp", "host", smtpCfg.Host, "port", smtpCfg.Port, "tls", smtpCfg.TLSMode, "from", smtpCfg.From)
		return emailSMTP.NewSMTPEmailSender(smtpCfg)
	default:
		logging.Info(context.Background(), "email sender configured", "kind", "console")
		return emailConsole.ConsoleEmailSender{}
	}
}
