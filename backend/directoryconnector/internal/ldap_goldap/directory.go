// Package ldap_goldap は、go-ldap で AD と OpenLDAP に TLS で接続し、bind してページングした検索を行う。
package ldap_goldap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

type Config struct {
	URL      string
	StartTLS bool
	// TLS は証明書の検証に使う。RootCAs が nil なら OS の信頼する CA を使う。
	TLS          *tls.Config
	Dialect      domain.Dialect
	BindDN       string
	BindPassword string
	PageSize     uint32
	Timeout      time.Duration
}

type Directory struct{ config Config }

// ErrPlaintextLDAP は、StartTLS を指定しない ldap:// の URL である (REQ-DIRECTORYCONNECTOR-002)。
var ErrPlaintextLDAP = errors.New("ldap: an ldap:// url requires start_tls")

// New は TLS を確立しない構成を拒否する。サーバーの名前を省いた TLS の構成には URL のホスト名を使う。
func New(config Config) (*Directory, error) {
	parsed, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("ldap url: %w", err)
	}
	switch parsed.Scheme {
	case "ldaps":
	case "ldap":
		if !config.StartTLS {
			return nil, ErrPlaintextLDAP
		}
	default:
		return nil, fmt.Errorf("ldap url: unsupported scheme %q", parsed.Scheme)
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if config.TLS != nil {
		tlsConfig = config.TLS.Clone()
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = parsed.Hostname()
	}
	config.TLS = tlsConfig
	return &Directory{config: config}, nil
}

// ValidateFilter は RFC 4515 のフィルターとして読めるかを確かめる。
func ValidateFilter(filter string) error {
	_, err := ldap.CompileFilter(filter)
	return err
}

// Connect は TLS を確立してから simple bind する。TLS の失敗では bind の要求を送らない。
func (d *Directory) Connect(ctx context.Context) (ports.DirectorySession, error) {
	options := []ldap.DialOpt{}
	if !d.config.StartTLS {
		options = append(options, ldap.DialWithTLSConfig(d.config.TLS))
	}
	conn, err := ldap.DialURL(d.config.URL, options...)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	if d.config.Timeout > 0 {
		conn.SetTimeout(d.config.Timeout)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	session := &session{conn: conn, config: d.config, stop: stop}
	if d.config.StartTLS {
		if err := conn.StartTLS(d.config.TLS); err != nil {
			_ = session.Close()
			return nil, fmt.Errorf("start tls: %w", err)
		}
	}
	if err := conn.Bind(d.config.BindDN, d.config.BindPassword); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("bind as %s: %w", d.config.BindDN, err)
	}
	return session, nil
}

type session struct {
	conn   *ldap.Conn
	config Config
	stop   func() bool
}

func (s *session) Close() error {
	s.stop()
	return s.conn.Close()
}

func (s *session) RootInfo(context.Context) (domain.RootInfo, error) {
	if s.config.Dialect != domain.ActiveDirectory {
		return domain.RootInfo{}, nil
	}
	result, err := s.conn.Search(ldap.NewSearchRequest("", ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=*)", []string{"highestCommittedUSN", "dsServiceName"}, nil))
	if err != nil {
		return domain.RootInfo{}, fmt.Errorf("root dse: %w", err)
	}
	if len(result.Entries) != 1 {
		return domain.RootInfo{}, errors.New("root dse: no entry")
	}
	root := result.Entries[0]
	usn, err := strconv.ParseInt(root.GetAttributeValue("highestCommittedUSN"), 10, 64)
	if err != nil {
		return domain.RootInfo{}, fmt.Errorf("root dse highestCommittedUSN: %w", err)
	}
	server := root.GetAttributeValue("dsServiceName")
	if server == "" {
		return domain.RootInfo{}, errors.New("root dse: no dsServiceName")
	}
	return domain.RootInfo{Server: server, HighestUSN: usn}, nil
}

// Search は RFC 2696 のページングで検索し、受け取ったページごとに page を呼ぶ。
func (s *session) Search(ctx context.Context, baseDN, filter string, attributes []string, page func([]domain.Entry) error) error {
	paging := ldap.NewControlPaging(s.config.PageSize)
	request := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, attributes, []ldap.Control{paging})
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, err := s.conn.Search(request)
		if err != nil {
			return fmt.Errorf("search %s: %w", baseDN, err)
		}
		entries := make([]domain.Entry, 0, len(result.Entries))
		for _, found := range result.Entries {
			entry := domain.Entry{DN: found.DN}
			for _, attribute := range found.Attributes {
				entry.Attributes = append(entry.Attributes, domain.Attribute{Name: attribute.Name, Values: attribute.ByteValues})
			}
			entries = append(entries, entry)
		}
		if err := page(entries); err != nil {
			return err
		}
		returned, ok := ldap.FindControl(result.Controls, ldap.ControlTypePaging).(*ldap.ControlPaging)
		if !ok || len(returned.Cookie) == 0 {
			return nil
		}
		paging.SetCookie(returned.Cookie)
	}
}
