package ldap_goldap

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/go-ldap/ldap/v3"
)

var entryUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func identityFilter(dialect domain.Dialect, externalID string) (string, bool) {
	if dialect == domain.ActiveDirectory {
		raw, err := base64.StdEncoding.Strict().DecodeString(externalID)
		if err != nil || len(raw) != 16 {
			return "", false
		}
		var value strings.Builder
		for _, b := range raw {
			fmt.Fprintf(&value, `\%02x`, b)
		}
		return "(objectGUID=" + value.String() + ")", true
	}
	if !entryUUIDPattern.MatchString(externalID) {
		return "", false
	}
	return "(entryUUID=" + ldap.EscapeFilter(externalID) + ")", true
}

// VerifyPassword はサービス接続と利用者の bind を分ける。再 bind で同期中の接続を変更しない。
func (d *Directory) VerifyPassword(ctx context.Context, scope domain.Scope, externalID, password string) string {
	filter, ok := identityFilter(d.config.Dialect, externalID)
	if !ok || password == "" {
		return "invalid_credentials"
	}
	service, err := d.connect(ctx, d.config.BindDN, d.config.BindPassword)
	if err != nil {
		return "directory_unavailable"
	}
	defer func() { _ = service.Close() }()
	var dns []string
	for _, base := range scope.BaseDNs {
		result, err := service.conn.Search(ldap.NewSearchRequest(base, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, 0, false, "(&"+scope.Filter+filter+")", []string{"1.1"}, nil))
		if err != nil || len(result.Referrals) > 0 {
			return "directory_unavailable"
		}
		for _, entry := range result.Entries {
			dns = append(dns, entry.DN)
		}
		if len(dns) > 1 {
			return "invalid_credentials"
		}
	}
	if len(dns) != 1 || dns[0] == "" {
		return "invalid_credentials"
	}
	user, err := d.connect(ctx, dns[0], password)
	if err == nil {
		_ = user.Close()
		return "accepted"
	}
	if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
		if d.config.Dialect == domain.ActiveDirectory {
			return adPasswordRejection(err.Error())
		}
		return "invalid_credentials"
	}
	return "directory_unavailable"
}

// AD の診断値は利用者へ公開せず、管理者向けの分類にだけ使う。
var adBindData = regexp.MustCompile(`(?i)\bdata ([0-9a-f]+)(?:,|\s|$)`)

func adPasswordRejection(diagnostic string) string {
	match := adBindData.FindStringSubmatch(diagnostic)
	if len(match) != 2 {
		return "invalid_credentials"
	}
	switch strings.ToLower(match[1]) {
	case "775":
		return "account_locked"
	case "533":
		return "account_disabled"
	case "701":
		return "account_expired"
	case "532":
		return "password_expired"
	case "773":
		return "password_change_required"
	default:
		return "invalid_credentials"
	}
}
